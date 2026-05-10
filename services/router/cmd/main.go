// Package main is the entry point for the router service.
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
	log.Println("router: started (stub)")
	<-ctx.Done()
	log.Println("router: stopped")
}
