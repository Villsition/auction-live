package middleware

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"auction/internal/metrics"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Request ID: accept from client or generate
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = fmt.Sprintf("%x-%x", time.Now().UnixNano(), rand.Int31())
		}
		c.Set("request_id", reqID)
		c.Header("X-Request-ID", reqID)

		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		cost := time.Since(start)

		// Prometheus metrics
		status := strconv.Itoa(c.Writer.Status())
		metrics.HTTPRequests.WithLabelValues(c.Request.Method, path, status).Inc()
		metrics.HTTPLatency.WithLabelValues(c.Request.Method, path).Observe(cost.Seconds())

		logger.Info("request",
			zap.String("req_id", reqID),
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", cost),
		)
	}
}
