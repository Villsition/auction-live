package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequests tracks total HTTP requests by method, path and status.
	HTTPRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auction_http_requests_total",
			Help: "Total HTTP requests by method, path and status code.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPLatency tracks HTTP request latency in seconds.
	HTTPLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auction_http_request_duration_seconds",
			Help:    "HTTP request latency histogram.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// BidsTotal tracks bid outcomes.
	BidsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auction_bids_total",
			Help: "Total bids placed, partitioned by outcome (success/fail).",
		},
		[]string{"outcome"},
	)

	// BidLatency tracks Redis Lua bid execution time in seconds.
	BidLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "auction_bid_latency_seconds",
			Help:    "Bid execution latency (Redis Lua script).",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
	)

	// WSConnections tracks active WebSocket connections.
	WSConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "auction_ws_connections",
			Help: "Current number of active WebSocket connections.",
		},
	)

	// AuctionsFinalized tracks completed auctions.
	AuctionsFinalized = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auction_finalized_total",
			Help: "Total auctions finalized, partitioned by result (sold/unsold).",
		},
		[]string{"result"},
	)

	// RedisErrors tracks Redis operation errors.
	RedisErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "auction_redis_errors_total",
			Help: "Total Redis operation errors.",
		},
	)

	// OnlineUsers tracks concurrent online users.
	OnlineUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "auction_online_users",
			Help: "Current number of online users across all rooms.",
		},
	)
)
