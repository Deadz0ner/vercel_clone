package utils

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	log.Println("[utils.id] random source seeded")
}

func GenerateRandomID() string {
	id := uuid.New().String()
	log.Printf("[utils.id] generated uuid=%s", id)
	return id
}

// timeStamp based
func GenerateTimStampBasedID() string {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	log.Printf("[utils.id] generated timestamp_id=%s", id)
	return id
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateID(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	id := string(b)
	log.Printf("[utils.id] generated random_id len=%d value=%s", n, id)
	return id
}
