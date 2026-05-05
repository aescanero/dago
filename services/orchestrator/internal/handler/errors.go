package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aescanero/dago/libs/domain"
)

type errResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func errorResp(code, message string) errResponse {
	return errResponse{Code: code, Message: message}
}

func mapDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResp("GRAPH_NOT_FOUND", err.Error()))
	case errors.Is(err, domain.ErrInvalidGraphStatus):
		c.JSON(http.StatusConflict, errorResp("GRAPH_NOT_DRAFT", err.Error()))
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, errorResp("GRAPH_DUPLICATE_VERSION", err.Error()))
	case errors.Is(err, domain.ErrValidation):
		c.JSON(http.StatusUnprocessableEntity, errorResp("VALIDATION_ERROR", err.Error()))
	default:
		c.JSON(http.StatusInternalServerError, errorResp("INTERNAL_ERROR", err.Error()))
	}
}
