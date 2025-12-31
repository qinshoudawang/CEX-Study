package ledger

import (
	"context"
	"database/sql"
	"dex-indexer/internal/middleware/db/model"
	"dex-indexer/internal/middleware/db/repository"
	"log"
	"math/big"
)

func (s *LedgerService) HoldWithdraw(
	ctx context.Context,
	userID int64,
	asset string,
	amount *big.Int,
	withdrawID int64,
) error {

	tx, _ := s.db.BeginTx(ctx, nil)
	defer tx.Rollback()

	if err := repository.InsertLedgerEntry(
		ctx, tx,
		userID,
		asset,
		new(big.Int).Neg(amount),
		string(WITHDRAW_HOLD),
		withdrawID,
	); err != nil {
		return err
	}

	if err := repository.AddBalance(
		ctx, tx,
		userID,
		asset,
		new(big.Int).Neg(amount),
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *LedgerService) FinalizeWithdraw(
	ctx context.Context,
	db *sql.DB,
	userID int64,
	asset string,
	txHash string,
	withdrawID int64,
) error {

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Println("Error beginning finalize withdraw transaction:", err)
		return err
	}
	defer tx.Rollback()

	if err := repository.InsertLedgerEntry(
		ctx,
		tx,
		userID,
		asset,
		big.NewInt(0), // already deducted in HOLD phase
		string(WITHDRAW_FINAL),
		withdrawID,
	); err != nil {
		return err
	}

	if err = repository.UpdateWithdrawStatusTx(
		ctx,
		tx,
		withdrawID,
		model.WithdrawConfirmed,
	); err != nil {
		return err
	}

	return nil
}
