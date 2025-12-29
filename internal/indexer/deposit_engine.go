package indexer

import (
	"context"
	"database/sql"
	"dex-indexer/internal/chain"
	"dex-indexer/internal/config"
	"dex-indexer/internal/middleware/db/model"
	"dex-indexer/internal/middleware/db/repository"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	CONFIRMATIONS = 2
	ETHEREUM      = "ethereum"
	ASSET_USDC    = "USDC"
)

type DepositEngine struct {
	db                *sql.DB
	client            *chain.Client
	exchangeAddresses map[string]bool
}

type DepositStatus string

func NewDepositEngine(cfg *config.Config, client *chain.Client, db *sql.DB) *DepositEngine {
	return &DepositEngine{
		db:                db,
		client:            client,
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
	if !de.exchangeAddresses[to.Hex()] {
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

func (de *DepositEngine) ListConfirmable(ctx context.Context) ([]*model.Deposit, error) {
	deposits, err := repository.ListPending(ctx, de.db)
	if err != nil {
		log.Println("Error listing pending deposits:", err)
		return nil, err
	}

	latestBlock, err := de.client.Eth.BlockNumber(ctx)
	if err != nil {
		log.Println("Error getting latest block number:", err)
		return nil, err
	}

	confirmable_deposits := []*model.Deposit{}
	for _, deposit := range deposits {
		if latestBlock-deposit.BlockNumber >= CONFIRMATIONS {
			deposit.Status = model.DepositConfirmed
			confirmable_deposits = append(confirmable_deposits, deposit)
		}
	}

	return confirmable_deposits, nil
}

func (de *DepositEngine) MarkConfirmedTx(
	ctx context.Context,
	depositID int64,
	tx *sql.Tx) error {
	log.Printf("[Confirmed] deposit ID: %d", depositID)
	return repository.ConfirmTx(
		ctx,
		depositID,
		tx,
	)
}

func (de *DepositEngine) GetUserIDFromDepositTx(
	ctx context.Context,
	deposit *model.Deposit,
	tx *sql.Tx,
) (int64, error) {
	return repository.GetUserIDByDepositAddressTx(
		ctx,
		ETHEREUM,
		deposit.FromAddress,
		tx,
	)
}
