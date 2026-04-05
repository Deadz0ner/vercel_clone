package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"vercel-clone/internal/config"
	"vercel-clone/internal/deploy"
	"vercel-clone/internal/queue"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

func main() {
	fmt.Println("Deploying...")
	redisClient = queue.NewRedisClient()

	router := mux.NewRouter()
	router.Use(loggingMiddleware)

	router.HandleFunc("/health", healthHandler)
	router.HandleFunc("/deploy", deployHandler)

	cfg := config.Load()
	port := ":" + cfg.Port
	fmt.Println("Server is running on " + port)
	http.ListenAndServe(port, router)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Health check received")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func deployHandler(w http.ResponseWriter, r *http.Request) {
	var req deploy.DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	resp, err := deploy.Run(req)
	if err != nil {
		http.Error(w, "deployment failed", http.StatusInternalServerError)
		return
	}
	if err := queue.PublishProjectID(context.Background(), redisClient, resp.Id); err != nil {
		log.Printf("[deployer] failed to publish project id=%s err=%v", resp.Id, err)
		http.Error(w, "deployment created but queue publish failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request:", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
