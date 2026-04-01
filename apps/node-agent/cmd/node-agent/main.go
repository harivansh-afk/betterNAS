package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rathi/betternas/apps/node-agent/internal/nodeagent"
)

func main() {
	cfg, err := nodeagent.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("load node-agent config: %v", err)
	}

	app, err := nodeagent.New(cfg, log.New(os.Stderr, "", log.LstdFlags))
	if err != nil {
		log.Fatalf("build node-agent app: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.ListenAndServe(ctx); err != nil {
		log.Fatalf("run node-agent: %v", err)
	}
}
