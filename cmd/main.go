package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dex-indexer/internal/config"
	"dex-indexer/internal/db"
	"dex-indexer/internal/indexer"
)

func main() {
	cfg := config.Load()
	if cfg.RPC == "" {
		log.Fatal("ETH_RPC_URL not set")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// db init
	dbConn, err := db.New(cfg.DBConfig)
	if err != nil {
		log.Fatal("failed to connect to database: ", err)
	}
	defer dbConn.Close()

	// indexer init
	idx := indexer.New(cfg, dbConn)
	if err := idx.Start(ctx); err != nil {
		log.Fatal("indexer stopped with error: ", err)
	}
}
