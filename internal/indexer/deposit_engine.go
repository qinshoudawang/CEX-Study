package indexer

import (
	"context"
	"database/sql"
	"dex-indexer/internal/config"
	"dex-indexer/internal/db/model"
	"dex-indexer/internal/db/repository"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type DepositEngine struct {
	confirmations     uint64
	db                *sql.DB
	exchangeAddresses map[string]bool
}

type DepositStatus string

func NewDepositEngine(cfg *config.Config, db *sql.DB) *DepositEngine {
	return &DepositEngine{
		confirmations:     cfg.Confirmations,
		db:                db,
		exchangeAddresses: cfg.ExchangeAddresses,
	}
}

func (de *DepositEngine) OnTransfer(
	ctx context.Context,
	txHash string,
	token, from, to common.Address,
	amount *big.Int,
	blockNumber uint64,
) {
	// Ignore if 'to' address is not an exchange address
	if !de.exchangeAddresses[strings.ToLower(to.Hex())] {
		return
	}

	deposit := &model.Deposit{
		TxHash:       txHash,
		BlockNumber:  blockNumber,
		TokenAddress: token.Hex(),
		FromAddress:  from.Hex(),
		ToAddress:    to.Hex(),
		Amount:       amount,
		Status:       model.DepositPending,
	}

	log.Println("[Pending]", txHash, "amount:", amount)
	// write DB / ledger
	err := repository.InsertPending(ctx, de.db, deposit)
	if err != nil {
		log.Println("Error inserting pending deposit:", err)
	}
}

func (de *DepositEngine) Confirm(ctx context.Context, latestBlock uint64) {
	deposits, err := repository.ListPending(ctx, de.db)
	if err != nil {
		log.Println("Error listing pending deposits:", err)
		return
	}

	for _, deposit := range deposits {
		if latestBlock-deposit.BlockNumber >= de.confirmations {
			deposit.Status = model.DepositConfirmed
			log.Println("[Confirmed]", deposit.TxHash, "amount:", deposit.Amount)
			// write DB / ledger
			err := repository.Confirm(ctx, de.db, deposit.ID)
			if err != nil {
				log.Println("Error confirming deposit:", err)
			}
		}
	}
}
