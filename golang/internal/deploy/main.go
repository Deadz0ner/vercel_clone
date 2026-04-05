package deploy

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vercel-clone/internal/config"
	"vercel-clone/internal/utils"

	git "github.com/go-git/go-git/v5"
)

func Run(req DeployRequest) (DeployResponse, error) {
	start := time.Now()
	log.Printf("[deploy.run] request project=%q repoURL=%q", req.ProjectName, req.RepoURL)

	if strings.TrimSpace(req.RepoURL) == "" {
		err := fmt.Errorf("repoURL is required")
		log.Printf("[deploy.run] validation failed: %v", err)
		return DeployResponse{}, err
	}

	// create ID for this deployment
	id := utils.GenerateID(8)
	log.Printf("[deploy.run] generated deployment_id=%s", id)

	cfg := config.Load()
	baseDir := cfg.BaseDir
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		log.Printf("[deploy.run] failed to create base directory=%q err=%v", baseDir, err)
		return DeployResponse{}, err
	}

	targetDir := filepath.Join(baseDir, id)
	log.Printf("[deploy.run] cloning repository into=%q", targetDir)

	// clone repo
	_, err := git.PlainClone(targetDir, false, &git.CloneOptions{
		URL: req.RepoURL,
	})

	if err != nil {
		log.Printf("[deploy.run] clone failed after=%s err=%v", time.Since(start), err)
		return DeployResponse{}, err
	}
	log.Printf("[deploy.run] clone successful duration=%s", time.Since(start))

	// ctx := context.Background()
	// log.Printf("[deploy.run] uploading directory to s3 bucket=%q prefix=%q", cfg.S3Bucket, id)
	// if err := uploadDirectoryToSupabaseS3(ctx, s3Client, cfg.S3Bucket, targetDir, id); err != nil {
	// 	log.Printf("[deploy.run] upload failed err=%v", err)
	// 	return DeployResponse{}, err
	// }
	// log.Printf("[deploy.run] upload successful")

	// if err := os.RemoveAll(targetDir); err != nil {
	// 	log.Printf("[deploy.run] failed to cleanup local directory=%q err=%v", targetDir, err)
	// 	return DeployResponse{}, err
	// }
	// log.Printf("[deploy.run] cleaned up local directory=%q", targetDir)

	resp := DeployResponse{
		Id:     id,
		Status: "success",
	}
	log.Printf("[deploy.run] completed response=%+v total_duration=%s", resp, time.Since(start))
	return resp, nil
}
