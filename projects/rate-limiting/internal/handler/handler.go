package handler

import (
	"context"
	"net/http"
	"time"

	"rate-limiting/internal/limiter"
	"rate-limiting/internal/metrics"
	"rate-limiting/internal/producer"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	limiter  *limiter.RedisLimiter
	producer *producer.KafkaProducer
}

type RateLimitRequest struct {
	ClientID string `json:"client_id" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

type RateLimitResponse struct {
	Allowed    bool   `json:"allowed"`
	TokensLeft int    `json:"tokens_left"`
	Message    string `json:"message"`
}

func NewHandler(limiter *limiter.RedisLimiter, producer *producer.KafkaProducer) *Handler {
	return &Handler{
		limiter:  limiter,
		producer: producer,
	}
}

// RateLimitedRequest handles API requests with rate limiting
func (h *Handler) RateLimitedRequest(c *gin.Context) {
	var req RateLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check rate limit: 100 tokens per minute (refill 1 token per 0.6 seconds)
	result := h.limiter.CheckLimit(ctx, req.ClientID, 100, 1, 1)

	if !result.Allowed {
		metrics.RecordLimitExceeded(req.ClientID)
		c.JSON(http.StatusTooManyRequests, RateLimitResponse{
			Allowed: false,
			Message: "Rate limit exceeded",
		})
		// Log rejection async
		go h.producer.SendEvent("rate_limit_rejected", map[string]string{
			"client_id": req.ClientID,
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Process request
	c.JSON(http.StatusOK, RateLimitResponse{
		Allowed:    true,
		TokensLeft: result.TokensLeft,
		Message:    "Request allowed",
	})

	// Log allowed request async
	go h.producer.SendEvent("rate_limit_allowed", map[string]string{
		"client_id": req.ClientID,
		"action":    req.Action,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
