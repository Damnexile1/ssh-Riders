package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/example/ssh-riders/internal/app/room"
	"github.com/example/ssh-riders/internal/config"
	"github.com/example/ssh-riders/pkg/logx"
)

func main() {
	cfg := config.LoadRoom()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	if err := room.New(cfg, logx.New("room")).Run(ctx); err != nil {
		log.Fatal(err)
	}
}
