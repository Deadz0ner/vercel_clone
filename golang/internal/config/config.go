package config

import (
	"log"
	"os"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                      string
	BaseDir                   string
	RedisAddr                 string
	RedisPassword             string
	RedisDB                   string
	RedisQueue                string
	BuilderImage              string
	BuilderCommand            string
	S3Bucket                  string
	SupabaseS3Region          string
	SupabaseS3Endpoint        string
	SupabaseS3AccessKeyID     string
	SupabaseS3SecretAccessKey string
	ServeHost                 string
}

var (
	once   sync.Once
	loaded *Config
)

func Load() *Config {
	once.Do(func() {
		_ = godotenv.Load()
		loaded = &Config{
			Port:                      getEnv("PORT", "8020"),
			BaseDir:                   getEnv("BASE_DIR", "local"),
			RedisAddr:                 getEnv("REDIS_ADDR", "localhost:6379"),
			RedisPassword:             getEnv("REDIS_PASSWORD", ""),
			RedisDB:                   getEnv("REDIS_DB", "0"),
			RedisQueue:                getEnv("REDIS_QUEUE", "deploy:jobs"),
			BuilderImage:              getEnv("BUILDER_IMAGE", "node:20-alpine"),
			BuilderCommand:            getEnv("BUILDER_COMMAND", "npm install && npm run build"),
			S3Bucket:                  mustGetEnv("SUPABASE_S3_BUCKET"),
			SupabaseS3Region:          getEnv("SUPABASE_S3_REGION", "ap-southeast-1"),
			SupabaseS3Endpoint:        mustGetEnv("SUPABASE_S3_ENDPOINT"),
			SupabaseS3AccessKeyID:     mustGetEnv("SUPABASE_S3_ACCESS_KEY_ID"),
			SupabaseS3SecretAccessKey: mustGetEnv("SUPABASE_S3_SECRET_ACCESS_KEY"),
			ServeHost:                 getEnv("SERVE_HOST", "http://localhost:3001"),
		}
	})
	return loaded
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		val = strings.TrimSpace(val)
		if val != "" {
			return val
		}
	}
	return fallback
}

func mustGetEnv(key string) string {
	val, ok := os.LookupEnv(key)
	val = strings.TrimSpace(val)
	if !ok || val == "" {
		log.Fatalf("Missing required env: %s", key)
	}
	return val
}
