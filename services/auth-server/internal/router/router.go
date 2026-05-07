package router

import (
	"github.com/gin-gonic/gin"

	"github.com/aescanero/dago/services/auth-server/internal/handler"
)

// NewRouter builds the Gin engine with all auth routes registered.
func NewRouter(authH *handler.AuthHandler, oauthH *handler.OAuthHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/.well-known/jwks.json", authH.JWKS)

	// Standard OAuth 2.1 endpoints — exception to /api/v1/ prefix per ADR-010 (same as JWKS).
	r.GET("/authorize", oauthH.GetAuthorize)
	r.POST("/authorize", oauthH.PostAuthorize)
	r.POST("/token", oauthH.PostToken)

	v1 := r.Group("/api/v1")
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
	}
	return r
}
