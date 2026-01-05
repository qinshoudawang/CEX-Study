package httpinfra

import (
	"time"

	"github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start).Seconds()
		method := c.Request.Method
		path := c.FullPath()
		status := statusGroup(c.Writer.Status())

		if path == "" {
			path = "unknown"
		}

		httpRequestsTotal.WithLabelValues(
			method,
			path,
			status,
		).Inc()

		httpRequestLatency.WithLabelValues(
			method,
			path,
		).Observe(latency)
	}
}

func statusGroup(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	default:
		return "2xx"
	}
}
