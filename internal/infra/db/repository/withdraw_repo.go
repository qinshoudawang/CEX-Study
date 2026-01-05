package repository

import (
	"context"
	"database/sql"
	"dex-indexer/internal/infra/db/model"
)

func InsertWithdraw(
	ctx context.Context,
	db *sql.DB,
	userID int64,
	asset string,
	amountStr string,
	toAddress string,
) (int64, error) {

	var id int64

	err := db.QueryRowContext(ctx, `
		INSERT INTO withdraws
		(user_id, asset, amount, to_address, status)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id
		`,
		userID,
		asset,
		amountStr,
		toAddress,
		string(model.WithdrawRequested),
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func UpdateWithdrawStatus(
	ctx context.Context,
	db *sql.DB,
	withdrawID int64,
	status model.WithdrawStatus,
	txHash string,
) error {

	_, err := db.ExecContext(ctx, `
		UPDATE withdraws
		SET status = $1, tx_hash = $2, confirmed_at = now()
		WHERE id = $3
		`,
		string(status),
		txHash,
		withdrawID,
	)
	if err != nil {
		return err
	}

	return nil
}

func UpdateWithdrawStatusTx(
	ctx context.Context,
	tx *sql.Tx,
	withdrawID int64,
	status model.WithdrawStatus,
) error {

	_, err := tx.ExecContext(ctx, `
		UPDATE withdraws
		SET status = $1, confirmed_at = now()
		WHERE id = $2
		`,
		string(status),
		withdrawID,
	)
	if err != nil {
		return err
	}

	return nil
}

func GetWithdrawsByStatus(
	ctx context.Context,
	db *sql.DB,
	status model.WithdrawStatus,
) ([]*model.Withdraw, error) {

	rows, err := db.QueryContext(ctx, `
		SELECT id, user_id, asset, amount, to_address, status, tx_hash, created_at, confirmed_at
		FROM withdraws
		WHERE status = $1
		ORDER BY created_at ASC
		`,
		status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdraws []*model.Withdraw
	for rows.Next() {
		var w model.Withdraw
		err := rows.Scan(
			&w.ID,
			&w.UserID,
			&w.Asset,
			&w.Amount,
			&w.ToAddress,
			&w.Status,
			&w.TxHash,
			&w.CreatedAt,
			&w.ConfirmedAt,
		)
		if err != nil {
			return nil, err
		}
		withdraws = append(withdraws, &w)
	}

	return withdraws, nil
}
