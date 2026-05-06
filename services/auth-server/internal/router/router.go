package router

import (
	"github.com/gin-gonic/gin"

	"github.com/aescanero/dago/services/auth-server/internal/handler"
)

// NewRouter builds the Gin engine with all auth routes registered.
func NewRouter(authH *handler.AuthHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/.well-known/jwks.json", authH.JWKS)

	v1 := r.Group("/api/v1")
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
	}
	return r
}
