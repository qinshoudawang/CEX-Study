package indexer

import (
	"context"
	"database/sql"
	"dex-indexer/internal/config"
	"dex-indexer/internal/ledger"
	"dex-indexer/internal/middleware/db/model"
	"dex-indexer/internal/middleware/db/repository"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type WithdrawEngine struct {
	hotWallet     string
	db            *sql.DB
	ledgerService *ledger.LedgerService
}

func NewWithdrawEngine(
	cfg *config.Config,
	ledgerService *ledger.LedgerService,
	db *sql.DB,
) *WithdrawEngine {
	return &WithdrawEngine{
		hotWallet:     cfg.HotWalletAddress,
		db:            db,
		ledgerService: ledgerService,
	}
}

func (we *WithdrawEngine) OnTransfer(
	ctx context.Context,
	txHash string,
	from common.Address,
	amount *big.Int,
	blockNumber uint64,
) {
	if from.Hex() != we.hotWallet {
		return
	}
	// Implementation for handling withdraw transfers
	withdraws, err := repository.GetWithdrawsByStatus(ctx, we.db, model.WithdrawSent)
	if err != nil {
		log.Println("Error fetching withdraws:", err)
	}
	for _, withdraw := range withdraws {
		if txHash != withdraw.TxHash {
			continue
		}

		err = we.ledgerService.FinalizeWithdraw(
			ctx,
			we.db,
			withdraw.UserID,
			withdraw.Asset,
			txHash,
			withdraw.ID,
		)
		if err != nil {
			log.Println("Error finalizing withdraw:", err)
			continue
		}

		log.Printf("Withdraw %d finalized for user %d", withdraw.ID, withdraw.UserID)
	}
}
