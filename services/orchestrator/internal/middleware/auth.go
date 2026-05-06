package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/aescanero/dago/libs/domain"
)

const claimsKey = "claims"

// TokenValidatorPort is the interface the middleware depends on for JWT validation.
type TokenValidatorPort interface {
	Validate(ctx context.Context, token string) (*domain.Claims, error)
}

// NewAuthMiddleware returns a Gin middleware that validates Bearer JWTs.
// When required is false the middleware calls c.Next() without any check (dev bypass).
func NewAuthMiddleware(required bool, validator TokenValidatorPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !required {
			c.Next()
			return
		}
		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorBody("MISSING_TOKEN", "missing or malformed Authorization header"))
			return
		}
		token := strings.TrimPrefix(raw, "Bearer ")
		claims, err := validator.Validate(c.Request.Context(), token)
		if err != nil || claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorBody("INVALID_TOKEN", "token is invalid or expired"))
			return
		}
		c.Set(claimsKey, claims)
		c.Next()
	}
}

// ClaimsFromContext extracts the validated Claims from a gin.Context.
func ClaimsFromContext(c *gin.Context) (*domain.Claims, bool) {
	v, exists := c.Get(claimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := v.(*domain.Claims)
	return claims, ok
}

func errorBody(code, message string) gin.H {
	return gin.H{"code": code, "message": message}
}
