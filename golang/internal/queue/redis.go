package queue

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"vercel-clone/internal/config"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() *redis.Client {
	cfg := config.Load()
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
	return client.RPush(ctx, config.Load().RedisQueue, projectID).Err()
}

func PopProjectID(ctx context.Context, client *redis.Client) (string, error) {
	result, err := client.BLPop(ctx, 0, config.Load().RedisQueue).Result()
	if err != nil {
		return "", err
	}
	if len(result) < 2 {
		return "", nil
	}
	return result[1], nil
}

// PublishStatus publishes a status update to a project-specific Redis Pub/Sub channel.
// The build service calls this; the entry service subscribes and forwards to WebSocket.
func PublishStatus(ctx context.Context, client *redis.Client, projectID, status string) error {
	channel := "status:" + projectID
	return client.Publish(ctx, channel, status).Err()
}

// SubscribeStatus subscribes to status updates for a given project ID.
// Returns the pubsub object — caller is responsible for closing it.
func SubscribeStatus(ctx context.Context, client *redis.Client, projectID string) *redis.PubSub {
	channel := "status:" + projectID
	return client.Subscribe(ctx, channel)
}

func Ping(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}

func SelfTest(ctx context.Context, client *redis.Client) error {
	if err := Ping(ctx, client); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	testQueue := config.Load().RedisQueue + ":selftest"
	testValue := fmt.Sprintf("redis-test-%d", time.Now().UnixNano())

	if err := client.Del(ctx, testQueue).Err(); err != nil {
		return fmt.Errorf("failed to clear test queue: %w", err)
	}

	if err := client.RPush(ctx, testQueue, testValue).Err(); err != nil {
		return fmt.Errorf("failed to push test value: %w", err)
	}

	result, err := client.BLPop(ctx, 1*time.Second, testQueue).Result()
	if err != nil {
		return fmt.Errorf("failed to pop test value: %w", err)
	}
	if len(result) < 2 {
		return fmt.Errorf("redis returned an unexpected blpop response: %v", result)
	}
	if result[1] != testValue {
		return fmt.Errorf("redis queue mismatch: got %q want %q", result[1], testValue)
	}

	return nil
}
