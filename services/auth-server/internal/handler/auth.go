package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aescanero/dago/libs/domain"
)

// RegisterUseCasePort is the interface the handler depends on for registration.
type RegisterUseCasePort interface {
	Execute(ctx context.Context, creds domain.Credentials) (*domain.User, error)
}

// LoginUseCasePort is the interface the handler depends on for login.
type LoginUseCasePort interface {
	Execute(ctx context.Context, creds domain.Credentials) (*domain.TokenPair, error)
}

// AuthHandler handles register, login, and JWKS endpoints.
type AuthHandler struct {
	register RegisterUseCasePort
	login    LoginUseCasePort
	jwksJSON []byte
}

// NewAuthHandler builds an AuthHandler with pre-computed JWKS JSON.
func NewAuthHandler(reg RegisterUseCasePort, login LoginUseCasePort, jwksJSON []byte) *AuthHandler {
	return &AuthHandler{register: reg, login: login, jwksJSON: jwksJSON}
}

type registerInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type userResponse struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var in registerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, errResp("BAD_REQUEST", err.Error()))
		return
	}
	user, err := h.register.Execute(c.Request.Context(), domain.Credentials{
		Email:    in.Email,
		Password: in.Password,
	})
	if err != nil {
		mapAuthError(c, err)
		return
	}
	tags := user.Tags
	if tags == nil {
		tags = []string{}
	}
	c.JSON(http.StatusCreated, userResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Tags:      tags,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

type loginInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var in loginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, errResp("BAD_REQUEST", err.Error()))
		return
	}
	pair, err := h.login.Execute(c.Request.Context(), domain.Credentials{
		Email:    in.Email,
		Password: in.Password,
	})
	if err != nil {
		mapAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, tokenResponse{
		AccessToken: pair.AccessToken,
		TokenType:   pair.TokenType,
		ExpiresIn:   pair.ExpiresIn,
	})
}

// JWKS handles GET /.well-known/jwks.json.
func (h *AuthHandler) JWKS(c *gin.Context) {
	c.Data(http.StatusOK, "application/json", h.jwksJSON)
}

type errResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func errResp(code, message string) errResponse {
	return errResponse{Code: code, Message: message}
}

func mapAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, errResp("INVALID_CREDENTIALS", "invalid email or password"))
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, errResp("EMAIL_CONFLICT", "email already registered"))
	case errors.Is(err, domain.ErrValidation):
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", err.Error()))
	default:
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", "internal server error"))
	}
}
