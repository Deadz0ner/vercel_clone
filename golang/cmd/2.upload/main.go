package main

import (
	"context"
	"log"

	"vercel-clone/internal/builder"
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
		projectPath := utils.GetProjectPath(projectID)

		log.Printf("[UPLOAD] starting build for project id=%s path=%s", projectID, projectPath)
		artifactPath, err := builder.RunBuildContainer(projectPath)
		if err != nil {
			log.Printf("[UPLOAD] failed to run build container for project id=%s: %v", projectID, err)
			continue
		}
		log.Printf("[UPLOAD] build completed for project id=%s artifact_path=%s", projectID, artifactPath)

		log.Printf("[UPLOAD] uploading artifact directory for project id=%s prefix=%s", projectID, projectID)
		err = utils.UploadDirectoryToSupabaseS3(context.Background(), s3Client, artifactPath, projectID)
		if err != nil {
			log.Printf("[UPLOAD] failed to upload project id=%s to s3: %v", projectID, err)
			continue
		}
		log.Printf("[UPLOAD] upload completed for project id=%s", projectID)
	}
}

func initSupabaseClient() (*s3.Client, error) {
	client, err := utils.NewSupabaseS3Client(context.Background())
	if err != nil {
		return nil, err
	}
	return client, nil
}
