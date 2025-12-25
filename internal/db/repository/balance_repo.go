package repository

import (
	"context"
	"database/sql"
	"math/big"
)

func AddBalance(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	asset string,
	amount *big.Int,
) error {

	_, err := tx.ExecContext(ctx, `
		INSERT INTO balances (user_id, asset, balance)
		VALUES ($1,$2,$3)
		ON CONFLICT (user_id, asset)
		DO UPDATE SET balance = balances.balance + EXCLUDED.balance
	`,
		userID,
		asset,
		amount.String(),
	)
	return err
}
