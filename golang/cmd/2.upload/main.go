package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pubsub := queue.SubscribeToProjectIDs(ctx, redisClient)
	defer pubsub.Close()

	log.Println("[upload] waiting for project ids from redis pub/sub")

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			log.Println("[upload] shutting down subscriber")
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}
			log.Printf("[upload] received project id=%s", msg.Payload)
		}
	}

}

func initSupabaseClient() (*s3.Client, error) {
	client, err := utils.NewSupabaseS3Client(context.Background())
	if err != nil {
		return nil, err
	}
	return client, nil
}
