package repository

import (
	"context"
	"database/sql"
)

func SaveBlock(
	ctx context.Context,
	db *sql.DB,
	number uint64,
	hash string,
	parentHash string,
) error {

	_, err := db.ExecContext(ctx, `
		INSERT INTO blocks (number, hash, parent_hash)
		VALUES ($1,$2,$3)
		ON CONFLICT (number)
		DO UPDATE SET hash = EXCLUDED.hash,
		              parent_hash = EXCLUDED.parent_hash
	`,
		number, hash, parentHash,
	)
	return err
}

func GetHashByBlockNumber(
	ctx context.Context,
	db *sql.DB,
	number uint64,
) (string, error) {

	hash := ""
	err := db.QueryRowContext(ctx,
		`SELECT hash FROM blocks WHERE number = $1`,
		number,
	).Scan(&hash)

	if err == sql.ErrNoRows {
		return "", nil
	}

	return hash, err
}
