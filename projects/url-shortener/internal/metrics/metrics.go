package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "url_shortener_http_requests_total",
			Help: "Total number of http requests",
		},
		[]string{"method", "route", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "url_shortener_http_request_duration",
			Help: "Duration of http requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	// Cache
	CacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_cache_hits_total",
			Help: "Total number of Redis cache hits",
		},
	)

	CacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_cache_misses_total",
			Help: "Total number of Redis cache misses",
		},
	)

	CacheErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_cache_errors_total",
			Help: "Total number of Redis cache errors",
		},
	)

	// DynamoDB
	DynamoReads = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_dynamo_reads_total",
			Help: "Total number of DynamoDB reads",
		},
	)

	DynamoWrites = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_dynamo_writes_total",
			Help: "Total number of DynamoDB writes",
		},
	)

	DynamoErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_dynamo_errors_total",
			Help: "Total number of DynamoDB errors",
		},
	)

	// Kafka
	KafkaPublished = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_kafka_published_total",
			Help: "Total number of events published to Kafka",
		},
	)

	KafkaErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_kafka_errors_total",
			Help: "Total number of Kafka publish errors",
		},
	)

	// URL operations
	UrlsCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_urls_created_total",
			Help: "Total number of short URLs created",
		},
	)

	UrlsRedirected = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_urls_redirected_total",
			Help: "Total number of successful redirects",
		},
	)

	UrlNotFound = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_shortener_url_not_found_total",
			Help: "Total number of short code lookups that returned nothing",
		},
	)

	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "url_shortener_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})


)


func Init() {
	// This function is intentionally left blank. The act of importing this package will register the metrics.
}