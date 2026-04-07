package main

import (
	"context"
	"log"
	"os"

	"vercel-clone/internal/builder"
	"vercel-clone/internal/config"
	"vercel-clone/internal/queue"
	"vercel-clone/internal/utils"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
)

var s3Client *s3.Client
var redisClient *redis.Client

func main() {
	client, err := initSupabaseClient()
	if err != nil {
		log.Fatalf("failed to initialize Supabase client: %v", err)
	}
	s3Client = client

	redisClient = queue.NewRedisClient()
	cfg := config.Load()
	log.Println("[UPLOAD] waiting for project ids from redis queue")
	for {
		projectID, err := queue.PopProjectID(context.Background(), redisClient)
		if err != nil {
			log.Printf("[UPLOAD] failed to pop project id err=%v", err)
			continue
		}
		if projectID == "" {
			continue
		}
		log.Printf("[UPLOAD] received project id=%s", projectID)

		// Notify frontend: building
		queue.PublishStatus(context.Background(), redisClient, projectID, "building")

		projectPath := utils.GetProjectPath(projectID)

		log.Printf("[UPLOAD] starting build for project id=%s path=%s", projectID, projectPath)
		artifactPath, err := builder.RunBuildContainer(projectPath)
		if err != nil {
			log.Printf("[UPLOAD] failed to run build container for project id=%s: %v", projectID, err)
			queue.PublishStatus(context.Background(), redisClient, projectID, "failed")
			continue
		}
		log.Printf("[UPLOAD] build completed for project id=%s artifact_path=%s", projectID, artifactPath)

		// Notify frontend: uploading artifacts
		queue.PublishStatus(context.Background(), redisClient, projectID, "uploading")

		log.Printf("[UPLOAD] uploading artifact directory for project id=%s prefix=%s", projectID, projectID)
		err = utils.UploadDirectoryToSupabaseS3(context.Background(), s3Client, artifactPath, projectID)
		if err != nil {
			log.Printf("[UPLOAD] failed to upload project id=%s to s3: %v", projectID, err)
			queue.PublishStatus(context.Background(), redisClient, projectID, "failed")
			continue
		}
		log.Printf("[UPLOAD] upload completed for project id=%s", projectID)

		log.Printf("[UPLOAD] cleaning up local files for project id=%s", projectID)
		if err := os.RemoveAll(projectPath); err != nil {
			log.Printf("[UPLOAD] failed to remove project directory id=%s path=%s: %v", projectID, projectPath, err)
		}
		if artifactPath != projectPath {
			if err := os.RemoveAll(artifactPath); err != nil {
				log.Printf("[UPLOAD] failed to remove artifact directory id=%s path=%s: %v", projectID, artifactPath, err)
			}
		}

		// Notify frontend: deployed — include the URL
		deployedURL := cfg.ServeHost + "/" + projectID + "/index.html"
		queue.PublishStatus(context.Background(), redisClient, projectID, "deployed:"+deployedURL)
	}
}

func initSupabaseClient() (*s3.Client, error) {
	client, err := utils.NewSupabaseS3Client(context.Background())
	if err != nil {
		return nil, err
	}
	return client, nil
}
