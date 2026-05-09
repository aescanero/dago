package router

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/aescanero/dago/adapters/auth"
	"github.com/aescanero/dago/services/orchestrator/internal/handler"
	"github.com/aescanero/dago/services/orchestrator/internal/middleware"
)

// NewRouter builds the Gin engine with all routes registered under /api/v1/.
func NewRouter(graphH *handler.GraphHandler, execH *handler.ExecutionHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	authRequired := os.Getenv("AUTH_REQUIRED") == "true"
	jwksURL := os.Getenv("JWKS_URL")
	jwtIssuer := os.Getenv("JWT_ISSUER")
	jwtAudience := os.Getenv("JWT_AUDIENCE")
	if jwtAudience == "" {
		jwtAudience = "dago-api"
	}

	var validator middleware.TokenValidatorPort
	if authRequired && jwksURL != "" {
		validator = auth.NewJWKSHTTPValidator(jwksURL, jwtIssuer, jwtAudience)
	}
	authMW := middleware.NewAuthMiddleware(authRequired, validator)

	v1 := r.Group("/api/v1")
	{
		g := v1.Group("/graphs")
		g.POST("", graphH.Create)
		g.GET("", graphH.List)
		g.GET("/:id", graphH.GetByID)
		g.PUT("/:id", graphH.Update)
		g.DELETE("/:id", graphH.Archive)

		e := v1.Group("/executions")
		e.POST("", execH.Start)

		// Protected group — empty for now; SPRINT-005 will apply authMW to /graphs and /executions.
		protected := v1.Group("/protected", authMW)
		protected.GET("/ping", func(c *gin.Context) { c.Status(200) })
	}
	return r
}
