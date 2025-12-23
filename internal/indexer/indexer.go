package indexer

import (
	"context"
	"errors"
	"log"

	"dex-indexer/internal/chain"
	"dex-indexer/internal/config"
)

type Indexer struct {
	cfg           *config.Config
	client        *chain.Client
	depositEngine *DepositEngine
}

func New(cfg *config.Config) *Indexer {
	client := chain.NewClient(cfg.RPC)

	return &Indexer{
		cfg:           cfg,
		client:        client,
		depositEngine: NewDepositEngine(cfg.Confirmations, cfg.ExchangeAddresses),
	}
}

func (i *Indexer) Start(ctx context.Context) error {
	log.Println("indexer started")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 2)

	go i.indexERC20Transfers(ctx, errChan)
	go i.ConfirmDeposit(ctx, errChan)

	// Wait for the first error (or cancellation), then stop the rest.
	err := <-errChan
	cancel()

	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		log.Println("indexer stopped: ", err)
		return nil
	}
	return err
}
