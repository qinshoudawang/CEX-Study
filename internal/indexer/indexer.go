package indexer

import (
	"context"
	"database/sql"
	"log"

	"dex-indexer/internal/chain"
	"dex-indexer/internal/config"
)

type Indexer struct {
	cfg           *config.Config
	client        *chain.Client
	DepositEngine *DepositEngine
}

func New(cfg *config.Config, db *sql.DB) *Indexer {
	client := chain.NewClient(cfg.RPC)

	return &Indexer{
		cfg:           cfg,
		client:        client,
		DepositEngine: NewDepositEngine(cfg, client, db),
	}
}

func (i *Indexer) Start(ctx context.Context, errChan chan error) {
	log.Println("indexer started")

	i.indexERC20Transfers(ctx, errChan)
}
