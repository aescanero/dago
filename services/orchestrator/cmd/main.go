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

	"github.com/aescanero/dago/adapters/storage"
	"github.com/aescanero/dago/ent"
	"github.com/aescanero/dago/services/orchestrator/internal/handler"
	"github.com/aescanero/dago/services/orchestrator/internal/router"
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

	graphRepo := storage.NewEntGraphRepository(client)
	execRepo := storage.NewEntExecutionRepository(client)
	graphUC := usecase.NewGraphUseCase(graphRepo, execRepo)
	execUC := usecase.NewExecutionUseCase(graphRepo, execRepo)
	graphH := handler.NewGraphHandler(graphUC)
	execH := handler.NewExecutionHandler(execUC)
	r := router.NewRouter(graphH, execH)

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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}
	log.Println("orchestrator stopped")
}
