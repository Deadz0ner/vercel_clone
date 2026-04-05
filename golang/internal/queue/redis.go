package queue

import (
	"context"
	"strconv"

	"vercel-clone/internal/config"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() *redis.Client {
	cfg := config.Load()
	// db converts the Redis database number from a string configuration value to an integer.
	// strconv.Atoi (ASCII to Integer) parses the string representation of the database number
	// from the configuration and returns it as an int, or an error if the string is not a valid integer.
	db, err := strconv.Atoi(cfg.RedisDB)
	if err != nil {
		db = 0
	}

	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       db,
	})
}

func PublishProjectID(ctx context.Context, client *redis.Client, projectID string) error {
	return client.Publish(ctx, config.Load().RedisChannel, projectID).Err()
}

func SubscribeToProjectIDs(ctx context.Context, client *redis.Client) *redis.PubSub {
	return client.Subscribe(ctx, config.Load().RedisChannel)
}
