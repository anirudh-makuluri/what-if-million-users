package main

import (
	"log"
	"os"
	"net/http"
	"time"

	"github.com/anirudh-makuluri/url-shortener/internal/cache"
	"github.com/anirudh-makuluri/url-shortener/internal/handler"
	"github.com/anirudh-makuluri/url-shortener/internal/kafka"
	"github.com/anirudh-makuluri/url-shortener/internal/metrics"
	"github.com/anirudh-makuluri/url-shortener/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// 1. Init dependencies
	dynamoStore, err := store.NewDynamoStore()
	if err != nil {
		log.Fatalf("failed to init DynamoDB: %v", err)
	}

	redisCache, err := cache.NewRedisCache()
	if err != nil {
		log.Fatalf("failed to init Redis cache: %v", err)
	}

	kafkaProducer := kafka.NewProducer()
	defer kafkaProducer.Close()

	// 2. Init metrics
	metrics.Init()

	// 3. Wire up handler
	h := handler.NewHandler(dynamoStore, redisCache, kafkaProducer)

	// 4. Set up router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(metricsMiddleware())

	r.GET("/:shortCode", h.Redirect)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 5. Expose Prometheus metrics on a separate port
	go func() {
		metricsPort := os.Getenv("METRICS_PORT")
		if metricsPort == "" {
			metricsPort = "9090"
		}
		http.ListenAndServe(":"+metricsPort, promhttp.Handler())
	}()

	// 6. Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		metrics.RequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
		).Observe(duration)
	}
}