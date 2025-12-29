package repository

import (
	"context"
	"database/sql"
	"dex-indexer/internal/middleware/db/model"
	"math/big"
)

func InsertPending(
	ctx context.Context,
	db *sql.DB,
	d *model.Deposit,
) error {

	_, err := db.ExecContext(ctx, `
		INSERT INTO deposits
		(tx_hash, block_number, token_address, from_address, to_address, amount, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT DO NOTHING
	`,
		d.TxHash,
		d.BlockNumber,
		d.TokenAddress,
		d.FromAddress,
		d.ToAddress,
		d.Amount.String(),
		model.DepositPending,
	)

	return err
}

func ListPending(
	ctx context.Context,
	db *sql.DB,
) ([]*model.Deposit, error) {

	rows, err := db.QueryContext(ctx, `
		SELECT id, tx_hash, block_number, token_address, from_address, amount
		FROM deposits
		WHERE status = 'PENDING'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*model.Deposit
	for rows.Next() {
		d := &model.Deposit{}
		var amountStr string

		if err := rows.Scan(
			&d.ID,
			&d.TxHash,
			&d.BlockNumber,
			&d.TokenAddress,
			&d.FromAddress,
			&amountStr,
		); err != nil {
			return nil, err
		}

		d.Amount, _ = new(big.Int).SetString(amountStr, 10)
		res = append(res, d)
	}

	return res, nil
}

func ConfirmTx(
	ctx context.Context,
	id int64,
	tx *sql.Tx,
) error {

	_, err := tx.ExecContext(ctx, `
		UPDATE deposits
		SET status = 'CONFIRMED',
		    confirmed_at = now()
		WHERE id = $1
	`, id)
	return err
}

func GetUserIDByDepositAddressTx(
	ctx context.Context,
	chain, address string,
	tx *sql.Tx,
) (int64, error) {

	var userID int64
	err := tx.QueryRowContext(ctx, `
		SELECT user_id
		FROM deposit_addresses
		WHERE chain = $1 AND address = $2
	`, chain, address).Scan(&userID)

	return userID, err
}
