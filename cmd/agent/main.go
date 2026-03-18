package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lumiforge/coach_chuck_ai/internal/app"
	"github.com/lumiforge/coach_chuck_ai/internal/config"
)

func main() {
	ctx := context.Background()

	cfg := config.GetConfig()

	application, err := app.NewApp(ctx, cfg)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := application.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	if err := application.Run(ctx); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
