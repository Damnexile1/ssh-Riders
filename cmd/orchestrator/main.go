package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/example/ssh-riders/internal/app/orchestrator"
	"github.com/example/ssh-riders/internal/config"
	"github.com/example/ssh-riders/pkg/logx"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	if err := orchestrator.New(config.LoadOrchestrator(), logx.New("orchestrator")).Run(ctx); err != nil {
		log.Fatal(err)
	}
}
