package ledger

import (
	"context"
	"database/sql"
	"dex-indexer/internal/infra/db/repository"
	"math/big"
)

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
		string(DEPOSIT),
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

func (s *LedgerService) RevertDepositTx(
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
		new(big.Int).Neg(amount),
		string(REVERSAL),
		depositID,
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

	return nil
}
