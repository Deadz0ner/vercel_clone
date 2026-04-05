package deploy

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func uploadDirectoryToSupabaseS3(ctx context.Context, client *s3.Client, bucket, rootDir, keyPrefix string) error {
	return filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		normalizedRelativePath := filepath.ToSlash(relPath)
		if strings.HasPrefix(normalizedRelativePath, ".git/") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file=%q for upload: %w", path, err)
		}
		defer file.Close()

		objectKey := fmt.Sprintf("%s/%s", keyPrefix, normalizedRelativePath)
		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(objectKey),
			Body:   file,
		})
		if err != nil {
			return fmt.Errorf("failed to upload object key=%q: %w", objectKey, err)
		}

		log.Printf("[deploy.upload] uploaded object key=%q", objectKey)
		return nil
	})
}
