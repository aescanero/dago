package router

import (
	"github.com/aescanero/dago/services/orchestrator/internal/handler"
	"github.com/gin-gonic/gin"
)

// NewRouter builds the Gin engine with all routes registered under /api/v1/.
func NewRouter(graphH *handler.GraphHandler, execH *handler.ExecutionHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

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
	}
	return r
}
