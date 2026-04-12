package metrics

import (
	"github.com/gin-gonic/gin"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsAllowed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limiter_requests_allowed_total",
			Help: "Total number of allowed requests",
		},
		[]string{"client_id"},
	)

	requestsRejected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limiter_requests_rejected_total",
			Help: "Total number of rejected requests",
		},
		[]string{"client_id"},
	)

	limitExceeded = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limiter_limit_exceeded_total",
			Help: "Total times rate limit was exceeded",
		},
		[]string{"client_id"},
	)

	limiterErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "rate_limiter_errors_total",
			Help: "Total number of limiter errors",
		},
	)
)

func init() {
	prometheus.MustRegister(requestsAllowed)
	prometheus.MustRegister(requestsRejected)
	prometheus.MustRegister(limitExceeded)
	prometheus.MustRegister(limiterErrors)
}

func Init() {
	// Metrics are registered in init()
}

func RecordAllowedRequest() {
	requestsAllowed.WithLabelValues("unknown").Inc()
}

func RecordRejectedRequest() {
	requestsRejected.WithLabelValues("unknown").Inc()
}

func RecordLimitExceeded(clientID string) {
	limitExceeded.WithLabelValues(clientID).Inc()
}

func RecordLimiterError() {
	limiterErrors.Inc()
}

func Handler(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}
