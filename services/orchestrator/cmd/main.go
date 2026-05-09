package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	valkeybus "github.com/aescanero/dago/adapters/eventbus/valkey"
	"github.com/aescanero/dago/adapters/storage"
	"github.com/aescanero/dago/ent"
	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
	"github.com/aescanero/dago/services/orchestrator/internal/consumer"
	"github.com/aescanero/dago/services/orchestrator/internal/handler"
	"github.com/aescanero/dago/services/orchestrator/internal/router"
	"github.com/aescanero/dago/services/orchestrator/internal/statemachine"
	"github.com/aescanero/dago/services/orchestrator/internal/usecase"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=dago sslmode=disable"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	valkeyAddr := os.Getenv("VALKEY_ADDR")
	if valkeyAddr == "" {
		valkeyAddr = "localhost:6379"
	}

	client, err := ent.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := client.Schema.Create(context.Background()); err != nil {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("close: %v", closeErr)
		}
		log.Fatalf("failed to run schema migration: %v", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("close: %v", closeErr)
		}
	}()

	publisher, err := valkeybus.NewPublisher(valkeyAddr)
	if err != nil {
		log.Fatalf("failed to create event publisher: %v", err)
	}
	defer publisher.Close()

	eventConsumer, err := valkeybus.NewConsumer(valkeyAddr)
	if err != nil {
		log.Fatalf("failed to create event consumer: %v", err)
	}
	defer eventConsumer.Close()

	graphRepo := storage.NewEntGraphRepository(client)
	execRepo := storage.NewEntExecutionRepository(client)

	sm := statemachine.NewExecutionStateMachine(execRepo, publisher)
	nodeResultConsumer := consumer.NewNodeResultConsumer(execRepo, graphRepo, sm)

	graphUC := usecase.NewGraphUseCase(graphRepo, execRepo)
	execUC := usecase.NewExecutionUseCase(graphRepo, execRepo, publisher)
	graphH := handler.NewGraphHandler(graphUC)
	execH := handler.NewExecutionHandler(execUC)
	r := router.NewRouter(graphH, execH)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start node.executed consumer goroutine.
	go func() {
		if err := eventConsumer.Subscribe(ctx, ports.ConsumeOptions{
			Stream:        domain.StreamNodeExecuted,
			Group:         "orchestrator-group",
			ConsumerName:  "orchestrator-node-executed-1",
			BlockDuration: 1 * time.Second,
			MaxRetries:    3,
		}, nodeResultConsumer.HandleNodeExecuted); err != nil {
			log.Printf("node.executed consumer stopped: %v", err)
		}
	}()

	// Start node.execute.failed consumer goroutine.
	go func() {
		if err := eventConsumer.Subscribe(ctx, ports.ConsumeOptions{
			Stream:        domain.StreamNodeExecuteFailed,
			Group:         "orchestrator-group",
			ConsumerName:  "orchestrator-node-failed-1",
			BlockDuration: 1 * time.Second,
			MaxRetries:    3,
		}, nodeResultConsumer.HandleNodeExecuteFailed); err != nil {
			log.Printf("node.execute.failed consumer stopped: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("orchestrator listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("orchestrator shutting down…")

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}
	log.Println("orchestrator stopped")
}
