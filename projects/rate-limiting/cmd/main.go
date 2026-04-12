package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"rate-limiting/internal/handler"
	"rate-limiting/internal/limiter"
	"rate-limiting/internal/metrics"
	"rate-limiting/internal/producer"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Initialize Prometheus metrics
	metrics.Init()

	// Redis connection
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://redis:6379"
	}

	// Parse Redis URL to extract host:port
	parsedURL, err := url.Parse(redisURL)
	if err != nil {
		log.Fatalf("Invalid REDIS_URL: %v", err)
	}
	redisAddr := parsedURL.Host
	if redisAddr == "" {
		// Fallback if URL parsing fails, try to extract from string like "redis://host:port"
		redisAddr = strings.TrimPrefix(redisURL, "redis://")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	// Kafka connection
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "kafka:9092"
	}
	kafkaProducer := producer.NewProducer(kafkaBrokers)
	defer kafkaProducer.Close()

	// Rate limiter
	limiterInstance := limiter.NewRedisLimiter(redisClient)

	// Gin setup
	r := gin.Default()

	// Create handler with dependencies
	h := handler.NewHandler(limiterInstance, kafkaProducer)

	// Routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	r.POST("/api/request", h.RateLimitedRequest)
	r.GET("/metrics", metrics.Handler)

	port := ":8080"
	log.Printf("Starting server on %s", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
