package http

import (
	"github.com/gin-gonic/gin"
)

// CORS middleware (placeholder for MVP)
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// AuthMiddleware is an authentication middleware (placeholder for future).
// Exported for use in production when authentication is implemented.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement authentication
		c.Next()
	}
}
