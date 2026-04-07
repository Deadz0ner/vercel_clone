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
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	fmt.Println("Deploying...")
	redisClient = queue.NewRedisClient()

	router := mux.NewRouter()
	router.Use(corsMiddleware)
	router.Use(loggingMiddleware)

	router.HandleFunc("/health", healthHandler)
	router.HandleFunc("/deploy", deployHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/ws/{id}", wsHandler)

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
		log.Printf("[DEPLOYER] failed to publish project id=%s err=%v", resp.Id, err)
		http.Error(w, "deployment created but queue publish failed", http.StatusInternalServerError)
		return
	}

	// Set initial status so the WebSocket subscriber knows it's queued
	queue.PublishStatus(context.Background(), redisClient, resp.Id, "queued")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// wsHandler upgrades to WebSocket, subscribes to Redis Pub/Sub for the project,
// and forwards every status update to the frontend until "deployed" or "failed".
func wsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] upgrade failed id=%s err=%v", projectID, err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Read pump — detect client disconnect
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	sub := queue.SubscribeStatus(ctx, redisClient, projectID)
	defer sub.Close()

	ch := sub.Channel()
	log.Printf("[WS] client connected id=%s", projectID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[WS] client disconnected id=%s", projectID)
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			log.Printf("[WS] sending status id=%s status=%s", projectID, msg.Payload)
			if err := conn.WriteJSON(map[string]string{"status": msg.Payload}); err != nil {
				log.Printf("[WS] write failed id=%s err=%v", projectID, err)
				return
			}
			// Close connection after terminal states
			if msg.Payload == "deployed" || msg.Payload == "failed" {
				log.Printf("[WS] terminal status reached id=%s, closing", projectID)
				return
			}
		}
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request:", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
