package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dex-indexer/internal/config"
	"dex-indexer/internal/indexer"
	"dex-indexer/internal/ledger"
	"dex-indexer/internal/middleware/db"
	redisclient "dex-indexer/internal/middleware/redis"
	"dex-indexer/internal/workflow"
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

	// redis init
	rClient := redisclient.New(cfg.RedisConfig)

	// indexer init
	idx := indexer.New(cfg, dbConn, rClient)

	// ledger init
	ledgerService := ledger.NewLedgerService(dbConn)

	// workflow init
	dcw := workflow.NewDepositConfirmWorkflow(
		idx.DepositEngine,
		ledgerService,
		dbConn,
	)

	// start indexer & workflow
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 2)

	go idx.Start(ctx, errChan)
	go dcw.Start(ctx, errChan)

	// Wait for the first error (or cancellation), then stop the rest.
	err = <-errChan
	cancel()
}
