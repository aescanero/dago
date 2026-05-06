// Package main is the entry point for the auth-server service.
package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	authadapter "github.com/aescanero/dago/adapters/auth"
	"github.com/aescanero/dago/ent"
	"github.com/aescanero/dago/services/auth-server/internal/handler"
	"github.com/aescanero/dago/services/auth-server/internal/router"
	"github.com/aescanero/dago/services/auth-server/internal/usecase"
)

func main() {
	dsn := envOrDefault("DATABASE_URL", "host=localhost user=dago password=dago dbname=dago sslmode=disable")
	port := envOrDefault("AUTH_PORT", "8081")
	jwtIssuer := envOrDefault("JWT_ISSUER", "dago-auth")
	jwtAudience := envOrDefault("JWT_AUDIENCE", "dago-api")

	privKey, pubKey := loadOrGenerateRSAKeyPair()

	client, err := ent.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	if err := client.Schema.Create(context.Background()); err != nil {
		_ = client.Close()
		log.Fatalf("run schema migration: %v", err)
	}
	defer func() { _ = client.Close() }()

	jwksJSON, err := authadapter.PublicKeyToJWKSJSON(pubKey)
	if err != nil {
		log.Fatalf("generate JWKS: %v", err)
	}

	repo := authadapter.NewEntUserRepository(client)
	hasher := authadapter.NewArgon2idHasher()
	issuer := authadapter.NewJWTIssuer(privKey, jwtIssuer, jwtAudience, 3600)
	registerUC := usecase.NewRegisterUser(repo, hasher)
	loginUC := usecase.NewLoginUser(repo, hasher, issuer)
	authH := handler.NewAuthHandler(registerUC, loginUC, jwksJSON)
	r := router.NewRouter(authH)

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("auth-server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down auth-server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}

func loadOrGenerateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	privPath := os.Getenv("JWT_PRIVATE_KEY_PATH")
	if privPath == "" {
		log.Println("WARNING: JWT_PRIVATE_KEY_PATH not set — generating in-memory RSA keypair (dev only)")
		return authadapter.MustGenerateRSAKeyPair()
	}
	privBytes, err := os.ReadFile(privPath)
	if err != nil {
		log.Fatalf("read private key %s: %v", privPath, err)
	}
	block, _ := pem.Decode(privBytes)
	if block == nil {
		log.Fatalf("decode PEM from %s", privPath)
	}
	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("parse private key: %v", err)
	}
	privKey, ok := priv.(*rsa.PrivateKey)
	if !ok {
		log.Fatalf("expected RSA private key, got other type")
	}
	return privKey, &privKey.PublicKey
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
