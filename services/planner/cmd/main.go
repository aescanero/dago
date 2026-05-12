// Package main is the entry point for the planner service.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	log.Println("planner: started (stub)")
	<-ctx.Done()
	log.Println("planner: stopped")
}
