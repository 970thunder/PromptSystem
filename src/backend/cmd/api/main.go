package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Tsundeer/isoumao-shared/contract"

	"promptos-backend/internal/api"
	"promptos-backend/internal/config"
	"promptos-backend/internal/platformapi"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	handler, ready, err := api.NewServerWithReadiness(cfg)
	if err != nil {
		log.Fatalf("initialize API: %v", err)
	}

	// 平台统一契约面（/api/platform/v1）：只新增端点，PromptSystem 自有路由不受影响。
	instanceID, _ := os.Hostname()
	platformHandler, perr := platformapi.New(platformapi.Deps{
		Version:      os.Getenv("PROMPTOS_VERSION"),
		Environment:  cfg.AppEnv,
		InstanceID:   instanceID,
		Token:        cfg.PlatformToken,
		TrustedAddrs: cfg.PlatformTrustedAddrs,
		AllowedPeers: cfg.PlatformAllowedPeers,
		// 复用站点自己的 MySQL/Redis 探活；不接的话平台面 /health 永远是 degraded（评审 P2-3）。
		Ping:          platformapi.PingFromReadiness(ready),
		StorageNote:   "上传方式 " + cfg.UploadProvider + "，目录 " + cfg.UploadDir,
		RequestIDFrom: api.RequestIDFrom,
	})
	if perr != nil {
		log.Fatalf("initialize platform API: %v", perr)
	}
	rootHandler := http.NewServeMux()
	// 平台面挂在站点自有链之外，必须自己套上请求 id 中间件，否则契约读不到站点 id（评审 P2-2）。
	rootHandler.Handle(contract.BasePath+"/", http.StripPrefix(contract.BasePath, api.WithRequestID(platformHandler)))
	rootHandler.Handle("/", handler)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           rootHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MiB
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Println("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("PromptOS Go backend listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
	log.Println("server stopped cleanly")
}
