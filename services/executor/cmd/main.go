// Package main is the entry point for the executor service.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	valkeyeventbus "github.com/aescanero/dago/adapters/eventbus/valkey"
	"github.com/aescanero/dago/adapters/llm/anthropic"
	"github.com/aescanero/dago/adapters/llm/ollama"
	"github.com/aescanero/dago/libs/ports"
	"github.com/aescanero/dago/services/executor/internal/consumer"
	"github.com/aescanero/dago/services/executor/internal/handler"
)

func main() {
	valkeyAddr := envOrDefault("EXECUTOR_VALKEY_ADDR", "localhost:6379")
	llmProvider := envOrDefault("EXECUTOR_LLM_PROVIDER", "anthropic")
	blockMs := envIntOrDefault("EXECUTOR_BLOCK_DURATION_MS", 5000)

	llmClient := buildLLMClient(llmProvider)

	pub, err := valkeyeventbus.NewPublisher(valkeyAddr)
	if err != nil {
		log.Fatalf("executor: publisher: %v", err)
	}

	evtConsumer, err := valkeyeventbus.NewConsumer(valkeyAddr)
	if err != nil {
		if closeErr := pub.Close(); closeErr != nil {
			log.Printf("executor: close publisher: %v", closeErr)
		}
		log.Fatalf("executor: consumer: %v", err)
	}

	_ = blockMs

	llmHandler := handler.NewLLMCallHandler(llmClient, pub)
	dispatcher := handler.NewDispatcher(map[string]handler.NodeHandler{
		"llm_call": llmHandler,
	})
	svc := consumer.NewNodeExecuteConsumer(evtConsumer, dispatcher)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	log.Println("executor: started")
	if err := svc.Run(ctx); err != nil {
		log.Printf("executor: stopped with error: %v", err)
	}

	if closeErr := evtConsumer.Close(); closeErr != nil {
		log.Printf("executor: close consumer: %v", closeErr)
	}
	if closeErr := pub.Close(); closeErr != nil {
		log.Printf("executor: close publisher: %v", closeErr)
	}
	log.Println("executor: stopped")
}

func buildLLMClient(provider string) ports.LLMClient {
	switch provider {
	case "ollama":
		baseURL := envOrDefault("OLLAMA_BASE_URL", "http://localhost:11434")
		return ollama.NewOllamaClient(ollama.Config{BaseURL: baseURL})
	default: // anthropic
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			log.Fatal("executor: ANTHROPIC_API_KEY is required for anthropic provider")
		}
		c, err := anthropic.NewAnthropicClient(anthropic.Config{APIKey: apiKey})
		if err != nil {
			log.Fatalf("executor: anthropic client: %v", err)
		}
		return c
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envIntOrDefault(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return defaultVal
}
