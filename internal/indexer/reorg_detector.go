package indexer

import (
	"context"
	"database/sql"
	"dex-indexer/internal/middleware/db/repository"
	"log"
)

func (i *Indexer) DetectReorg(
	ctx context.Context,
	number uint64,
	parentHash string,
) (bool, error) {

	var stored string
	stored, err := repository.GetHashByBlockNumber(ctx, i.db, number)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		log.Println("Error detecting reorg:", err)
		return false, err
	}

	return stored != parentHash, nil
}
