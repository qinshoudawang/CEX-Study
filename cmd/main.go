package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dex-indexer/internal/api"
	"dex-indexer/internal/chain"
	"dex-indexer/internal/config"
	"dex-indexer/internal/indexer"
	"dex-indexer/internal/ledger"
	"dex-indexer/internal/middleware/db"
	redisclient "dex-indexer/internal/middleware/redis"
	"dex-indexer/internal/service"
	"dex-indexer/internal/workflow"

	"github.com/gin-gonic/gin"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

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
	rClient, err := redisclient.New(ctx, cfg.RedisConfig)
	if err != nil {
		log.Fatal("failed to connect to redis: ", err)
	}
	defer rClient.Close()

	// ledger init
	ledgerService := ledger.NewLedgerService(dbConn)

	// indexer init
	idx := indexer.New(cfg, dbConn, rClient, ledgerService)

	// workflow init
	dcw := workflow.NewDepositWorkflow(
		idx.DepositEngine,
		ledgerService,
		dbConn,
		rClient,
	)

	// start indexer & workflow
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error)

	go idx.Start(ctx, errChan)
	go dcw.Start(ctx, errChan)

	// gin server init & start
	go InitializeGinServer(ctx, idx.Client, ledgerService, dbConn, errChan)

	// Wait for the first error (or cancellation), then stop the rest.
	err = <-errChan
	if err != nil {
		log.Println("Error occurred: ", err)
	}
	cancel()
}

func InitializeGinServer(
	ctx context.Context,
	client *chain.Client,
	ledgerService *ledger.LedgerService,
	dbConn *sql.DB,
	errChan chan error,
) {
	r := gin.Default()

	withdrawService := service.NewWithdrawServiceImpl(
		ctx,
		client,
		ledgerService,
		dbConn,
	)
	withdrawHandler := api.NewWithdrawHandler(withdrawService)

	api.RegisterRoutes(r, withdrawHandler)

	// Create http.Server for graceful shutdown
	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	log.Println("Gin server started on :8080")

	// Wait for context cancellation
	<-ctx.Done()

	// Gracefully shut down the server
	log.Println("Shutting down Gin server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Gin server forced to shutdown: %v", err)
	}
}
