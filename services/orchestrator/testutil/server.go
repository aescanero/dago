package testutil

import (
	"net/http/httptest"

	"github.com/aescanero/dago/libs/ports"
	"github.com/aescanero/dago/services/orchestrator/internal/handler"
	"github.com/aescanero/dago/services/orchestrator/internal/router"
	"github.com/aescanero/dago/services/orchestrator/internal/usecase"
)

// NewTestServer creates an httptest.Server wired with fake repositories and a no-op publisher.
func NewTestServer(graphRepo ports.GraphRepository, execRepo ports.ExecutionRepository) *httptest.Server {
	return NewTestServerWithPublisher(graphRepo, execRepo, noopPublisher{})
}

// NewTestServerWithPublisher creates an httptest.Server with an explicit publisher.
func NewTestServerWithPublisher(
	graphRepo ports.GraphRepository,
	execRepo ports.ExecutionRepository,
	publisher ports.EventPublisher,
) *httptest.Server {
	graphUC := usecase.NewGraphUseCase(graphRepo, execRepo)
	execUC := usecase.NewExecutionUseCase(graphRepo, execRepo, publisher)
	graphH := handler.NewGraphHandler(graphUC)
	execH := handler.NewExecutionHandler(execUC)
	r := router.NewRouter(graphH, execH)
	return httptest.NewServer(r)
}
