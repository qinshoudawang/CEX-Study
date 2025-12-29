package repository

import (
	"context"
	"database/sql"
	"math/big"
)

func InsertLedgerEntry(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	asset string,
	amount *big.Int,
	entryType string,
	refID int64,
) error {

	_, err := tx.ExecContext(ctx, `
		INSERT INTO ledger_entries
		(user_id, asset, amount, entry_type, ref_id)
		VALUES ($1,$2,$3,$4,$5)
	`,
		userID,
		asset,
		amount.String(),
		entryType,
		refID,
	)
	return err
}
