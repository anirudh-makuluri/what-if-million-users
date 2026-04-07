package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/anirudh-makuluri/what-if-million-users/url-shortener/internal/cache"
	"github.com/anirudh-makuluri/what-if-million-users/url-shortener/internal/kafka"
	"github.com/anirudh-makuluri/what-if-million-users/url-shortener/internal/metrics"
	"github.com/anirudh-makuluri/what-if-million-users/url-shortener/internal/store"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	store    *store.DynamoStore
	cache    *cache.RedisCache
	producer *kafka.Producer
}

type shortenRequest struct {
	ShortCode string `json:"short_code"`
	LongURL   string `json:"long_url"`
}

func NewHandler(store *store.DynamoStore, cache *cache.RedisCache, producer *kafka.Producer) *Handler {
	return &Handler{
		store:    store,
		cache:    cache,
		producer: producer,
	}
}

func (h *Handler) Shorten(c *gin.Context) {
	ctx := context.Background()
	var req shortenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// 1. Validate input
	if req.ShortCode == "" || req.LongURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "short_code and long_url are required"})
		return
	}

	// 2. Store in DynamoDB
	record := store.URLRecord{
		ShortCode: req.ShortCode,
		LongURL:   req.LongURL,
		UserID:    "anonymous", // in a real app, we'd get this from auth context
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}


	if err := h.store.SaveURLIfNotExists(ctx, record); err != nil {
		if err == store.ErrShortCodeExists {
			c.JSON(http.StatusConflict, gin.H{"error": "short code already exists"})
			return
		}
		metrics.DynamoErrors.Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	metrics.DynamoWrites.Inc()
	c.JSON(http.StatusOK, gin.H{"message": "short URL created successfully"})
}

func (h *Handler) Redirect(c *gin.Context) {
	shortCode := c.Param("shortCode")
	ctx := context.Background()

	// 1. Check Redis cache first
	longURL, found, err := h.cache.Get(ctx, shortCode)
	if err != nil {
		metrics.CacheErrors.Inc()
	}

	if found {
		metrics.CacheHits.Inc()
		h.publishEvent(c, shortCode, longURL)
		c.Redirect(http.StatusMovedPermanently, longURL)
		return
	}

	// 2. Cache miss — query DynamoDB
	metrics.CacheMisses.Inc()

	record, err := h.store.GetURL(ctx, shortCode)
	if err != nil {
		metrics.DynamoErrors.Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	metrics.DynamoReads.Inc()

	if record == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "short code not found"})
		return
	}

	// 3. Populate cache for next time
	if err := h.cache.Set(ctx, shortCode, record.LongURL); err != nil {
		metrics.CacheErrors.Inc()
		// non-fatal, we still have the record from DynamoDB
	}

	// 4. Publish analytics event to Kafka
	h.publishEvent(c, shortCode, record.LongURL)

	c.Redirect(http.StatusMovedPermanently, record.LongURL)
}

func (h *Handler) publishEvent(c *gin.Context, shortCode, longURL string) {
	event := kafka.RedirectEvent{
		ShortCode: shortCode,
		LongURL:   longURL,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		UserAgent: c.Request.Header.Get("User-Agent"),
		IPAddress: c.ClientIP(),
	}

	if err := h.producer.PublishRedirectEvent(context.Background(), event); err != nil {
		metrics.KafkaErrors.Inc()
		// non-fatal, analytics loss is acceptable
	}
}
