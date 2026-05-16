package main

import (
	"log"
	"net/http"
	"time"

	"promptos-backend/internal/api"
	"promptos-backend/internal/config"
)

func main() {
	cfg := config.Load()
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.NewServer(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("PromptOS Go backend listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
