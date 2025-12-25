package ledger

import (
	"context"
	"database/sql"
	"dex-indexer/internal/db/repository"
	"math/big"
)

type EntryType string

const (
	Deposit  EntryType = "DEPOSIT"
	Withdraw EntryType = "WITHDRAW"
	Trade    EntryType = "TRADE"
	REVERSAL EntryType = "REVERSAL"
)

type LedgerService struct {
	db *sql.DB
}

func NewLedgerService(db *sql.DB) *LedgerService {
	return &LedgerService{db: db}
}

func (s *LedgerService) ApplyDepositTx(
	ctx context.Context,
	depositID int64,
	userID int64,
	asset string,
	amount *big.Int,
	tx *sql.Tx,
) error {

	if err := repository.InsertLedgerEntry(
		ctx, tx,
		userID,
		asset,
		amount,
		string(Deposit),
		depositID,
	); err != nil {
		return err
	}

	if err := repository.AddBalance(
		ctx, tx,
		userID,
		asset,
		amount,
	); err != nil {
		return err
	}

	return nil
}
