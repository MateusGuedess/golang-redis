package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/teris-io/shortid"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

type ShortenRequest struct {
	LongURL string `json:"longUrl"`
}

type ShortenResponse struct {
	ShortURL string `json:"ShortUrl"`
}

func initRedis() {
	redisAddr := os.Getenv("ADDRESS")
	username := os.Getenv("USERNAME")
	pwd := os.Getenv("PASSWORD")

	if redisAddr == "" {
		log.Fatal("Not Found Redis Address")
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: username,
		Password: pwd,
		DB:       0,
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	fmt.Println("Connected to Redis!")
}

func shortenURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req ShortenRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.LongURL == "" {
		http.Error(w, "longUrl is required", http.StatusBadRequest)
	}

	var shortCode string

	for {
		sid, err := shortid.Generate()
		if err != nil {
			log.Printf("Error generating shrot ID: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		shortCode = sid

		value, err := rdb.Set(ctx, shortCode, req.LongURL, 30).Result()
		if err != nil {
			log.Printf("Redis SET error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}

		if value != "" {
			break
		}
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	shortURL := fmt.Sprintf("$s/$s", baseURL, shortCode)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ShortenResponse{ShortURL: shortURL})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	initRedis()

	http.HandleFunc("/api/shorten", shortenURLHandler)

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	fmt.Printf("Go backend server listening on :%s\n", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
