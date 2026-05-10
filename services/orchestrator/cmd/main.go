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
	dsn := envOrDefault("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=dago sslmode=disable")
	port := envOrDefault("PORT", "8080")
	valkeyAddr := envOrDefault("VALKEY_ADDR", "localhost:6379")

	client := mustOpenDB(dsn)
	defer closeDB(client)

	publisher := mustNewPublisher(valkeyAddr)
	eventConsumer := mustNewConsumer(valkeyAddr)

	graphRepo := storage.NewEntGraphRepository(client)
	execRepo := storage.NewEntExecutionRepository(client)

	sm := statemachine.NewExecutionStateMachine(execRepo, publisher)
	nodeConsumer := consumer.NewNodeResultConsumer(execRepo, graphRepo, sm)

	graphUC := usecase.NewGraphUseCase(graphRepo, execRepo)
	execUC := usecase.NewExecutionUseCase(graphRepo, execRepo, publisher)
	graphH := handler.NewGraphHandler(graphUC)
	execH := handler.NewExecutionHandler(execUC)
	r := router.NewRouter(graphH, execH)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startConsumers(ctx, eventConsumer, nodeConsumer)

	srv := &http.Server{Addr: ":" + port, Handler: r}
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
	if err := publisher.Close(); err != nil {
		log.Printf("publisher close: %v", err)
	}
	if err := eventConsumer.Close(); err != nil {
		log.Printf("consumer close: %v", err)
	}
	log.Println("orchestrator stopped")
}

func startConsumers(ctx context.Context, ec ports.EventConsumer, nc *consumer.NodeResultConsumer) {
	subscribe := func(stream, name string, fn ports.EventHandler) {
		go func() {
			if err := ec.Subscribe(ctx, ports.ConsumeOptions{
				Stream:        stream,
				Group:         "orchestrator-group",
				ConsumerName:  name,
				BlockDuration: 1 * time.Second,
				MaxRetries:    3,
			}, fn); err != nil {
				log.Printf("%s consumer stopped: %v", stream, err)
			}
		}()
	}
	subscribe(domain.StreamNodeExecuted, "orchestrator-node-executed-1", nc.HandleNodeExecuted)
	subscribe(domain.StreamNodeExecuteFailed, "orchestrator-node-failed-1", nc.HandleNodeExecuteFailed)
}

func mustOpenDB(dsn string) *ent.Client {
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	return client
}

func closeDB(client *ent.Client) {
	if err := client.Close(); err != nil {
		log.Printf("db close: %v", err)
	}
}

func mustNewPublisher(addr string) *valkeybus.Publisher {
	p, err := valkeybus.NewPublisher(addr)
	if err != nil {
		log.Fatalf("failed to create event publisher: %v", err)
	}
	return p
}

func mustNewConsumer(addr string) *valkeybus.Consumer {
	c, err := valkeybus.NewConsumer(addr)
	if err != nil {
		log.Fatalf("failed to create event consumer: %v", err)
	}
	return c
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
