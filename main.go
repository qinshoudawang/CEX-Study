package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dex-indexer/internal/config"
	"dex-indexer/internal/indexer"
)

func main() {
	cfg := config.Load()
	if cfg.RPC == "" {
		log.Fatal("ETH_RPC_URL not set")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	idx := indexer.New(cfg)
	if err := idx.Start(ctx); err != nil {
		log.Fatal("indexer stopped with error: ", err)
	}
}
