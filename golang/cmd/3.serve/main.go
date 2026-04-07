package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"vercel-clone/internal/config"
	"vercel-clone/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client

var pathAliasCache sync.Map

func main() {
	client, err := utils.NewSupabaseS3Client(context.Background())
	if err != nil {
		log.Fatalf("failed to initialize S3 client: %v", err)
	}
	s3Client = client

	cfg := config.Load()
	port := ":3001"
	fmt.Printf("[SERVE] serving deployed apps on %s\n", port)
	fmt.Printf("[SERVE] route pattern: /<projectId>/<filepath>\n")
	fmt.Printf("[SERVE] s3 bucket: %s\n", cfg.S3Bucket)

	http.HandleFunc("/", serveHandler)
	log.Fatal(http.ListenAndServe(port, nil))
}

// serveHandler resolves /<projectId>/path/to/file → S3 key: <projectId>/path/to/file
func serveHandler(w http.ResponseWriter, r *http.Request) {
	// Strip leading slash, expect at least projectId/filename
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		http.Error(w, "usage: /<projectId>/<file>", http.StatusBadRequest)
		return
	}

	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] == "" {
		// Default to index.html if only projectId is given
		path = parts[0] + "/index.html"
	}

	cfg := config.Load()
	objectKeys := resolveObjectKeys(path, r.Referer())

	var (
		result    *s3.GetObjectOutput
		err       error
		objectKey string
	)

	for _, candidate := range objectKeys {
		result, err = s3Client.GetObject(context.Background(), &s3.GetObjectInput{
			Bucket: aws.String(cfg.S3Bucket),
			Key:    aws.String(candidate),
		})
		if err == nil {
			objectKey = candidate
			rememberResolvedAlias(path, objectKey)
			break
		}

		if !isNoSuchKeyError(err) {
			log.Printf("[SERVE] 500 path=%s key=%s err=%v", r.URL.Path, candidate, err)
			http.Error(w, "failed to fetch file", http.StatusInternalServerError)
			return
		}
	}

	if result == nil {
		log.Printf("[SERVE] 404 path=%s", r.URL.Path)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer result.Body.Close()

	// Set content type from file extension
	ext := filepath.Ext(objectKey)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Access-Control-Allow-Origin", "*")

	written, copyErr := io.Copy(w, result.Body)
	if copyErr != nil {
		log.Printf("[SERVE] write_err path=%s key=%s bytes=%d err=%v", r.URL.Path, objectKey, written, copyErr)
		return
	}

	log.Printf(
		"[SERVE] 200 path=%s key=%s type=%s bytes=%d",
		r.URL.Path,
		objectKey,
		contentType,
		written,
	)
}

func resolveObjectKeys(path, referer string) []string {
	keys := []string{path}

	if aliasKey := resolveObjectKeyFromAlias(path); aliasKey != "" {
		keys = append(keys, aliasKey)
	}

	refererProjectID := projectIDFromReferer(referer)
	if refererProjectID == "" {
		return dedupeStrings(keys)
	}

	trimmedPath := strings.Trim(path, "/")
	if trimmedPath == "" {
		return dedupeStrings(keys)
	}

	parts := strings.Split(trimmedPath, "/")
	var candidate string
	if len(parts) == 1 {
		candidate = refererProjectID + "/" + parts[0]
	} else {
		candidate = refererProjectID + "/" + strings.Join(parts[1:], "/")
	}

	if candidate != path {
		keys = append(keys, candidate)
	}

	return dedupeStrings(keys)
}

func resolveObjectKeyFromAlias(path string) string {
	trimmedPath := strings.Trim(path, "/")
	if trimmedPath == "" {
		return ""
	}

	parts := strings.Split(trimmedPath, "/")
	if len(parts) < 2 {
		return ""
	}

	aliasPrefix := parts[0]
	projectIDAny, ok := pathAliasCache.Load(aliasPrefix)
	if !ok {
		return ""
	}

	projectID, ok := projectIDAny.(string)
	if !ok || strings.TrimSpace(projectID) == "" {
		return ""
	}

	return projectID + "/" + strings.Join(parts[1:], "/")
}

func projectIDFromReferer(referer string) string {
	if strings.TrimSpace(referer) == "" {
		return ""
	}

	parsed, err := url.Parse(referer)
	if err != nil {
		return ""
	}

	refererPath := strings.Trim(parsed.Path, "/")
	if refererPath == "" {
		return ""
	}

	parts := strings.SplitN(refererPath, "/", 2)
	if len(parts) == 0 {
		return ""
	}

	return parts[0]
}

func isNoSuchKeyError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	return strings.Contains(msg, "NoSuchKey") || strings.Contains(msg, "not found")
}

func dedupeStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func rememberResolvedAlias(requestPath, objectKey string) {
	requestParts := strings.Split(strings.Trim(requestPath, "/"), "/")
	objectParts := strings.Split(strings.Trim(objectKey, "/"), "/")
	if len(requestParts) < 2 || len(objectParts) < 2 {
		return
	}

	requestPrefix := requestParts[0]
	objectPrefix := objectParts[0]
	if requestPrefix == "" || objectPrefix == "" || requestPrefix == objectPrefix {
		return
	}

	pathAliasCache.Store(requestPrefix, objectPrefix)
}
