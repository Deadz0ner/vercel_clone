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
var knownProjects sync.Map

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
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		http.Error(w, "usage: /<projectId>/<file>", http.StatusBadRequest)
		return
	}

	parts := strings.SplitN(path, "/", 2)
	firstSegment := parts[0]

	var projectID, filePath string

	if _, ok := knownProjects.Load(firstSegment); ok {
		// First segment is a known project ID
		projectID = firstSegment
		if len(parts) == 2 && parts[1] != "" {
			filePath = parts[1]
		} else {
			filePath = "index.html"
		}
	} else if refID := projectIDFromReferer(r.Referer()); refID != "" {
		// First segment is NOT a project ID — prepend the one from Referer
		projectID = refID
		filePath = path
	} else {
		// No referer context — treat first segment as project ID (first visit)
		projectID = firstSegment
		if len(parts) == 2 && parts[1] != "" {
			filePath = parts[1]
		} else {
			filePath = "index.html"
		}
		knownProjects.Store(projectID, struct{}{})
	}

	objectKey := projectID + "/" + filePath

	cfg := config.Load()
	result, err := s3Client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(cfg.S3Bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		if isNoSuchKeyError(err) {
			log.Printf("[SERVE] 404 path=%s key=%s", r.URL.Path, objectKey)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("[SERVE] 500 path=%s key=%s err=%v", r.URL.Path, objectKey, err)
		http.Error(w, "failed to fetch file", http.StatusInternalServerError)
		return
	}
	defer result.Body.Close()

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

	knownProjects.Store(projectID, struct{}{})
	log.Printf("[SERVE] 200 path=%s key=%s type=%s bytes=%d", r.URL.Path, objectKey, contentType, written)
}

func projectIDFromReferer(referer string) string {
	if referer == "" {
		return ""
	}
	parsed, err := url.Parse(referer)
	if err != nil {
		return ""
	}
	p := strings.Trim(parsed.Path, "/")
	if p == "" {
		return ""
	}
	seg := strings.SplitN(p, "/", 2)[0]
	if _, ok := knownProjects.Load(seg); ok {
		return seg
	}
	return ""
}

func isNoSuchKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "NoSuchKey") || strings.Contains(msg, "not found")
}
