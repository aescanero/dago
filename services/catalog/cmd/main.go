// Package main is the entry point for the catalog service.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	port := envOrDefault("CATALOG_PORT", "8082")
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	log.Printf("catalog listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("catalog: %v", err)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
