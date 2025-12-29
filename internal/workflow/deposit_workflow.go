package workflow

import (
	"context"
	"database/sql"
	"dex-indexer/internal/indexer"
	"dex-indexer/internal/ledger"
	"dex-indexer/internal/middleware/db/repository"
	redisclient "dex-indexer/internal/middleware/redis/action"
	"log"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

type DepositWorkflow struct {
	depositEngine *indexer.DepositEngine
	ledgerService *ledger.LedgerService
	db            *sql.DB
	redis         *redis.Client
}

func NewDepositWorkflow(
	depositEngine *indexer.DepositEngine,
	ledgerService *ledger.LedgerService,
	dbConn *sql.DB,
	redis *redis.Client,
) *DepositWorkflow {

	return &DepositWorkflow{
		depositEngine: depositEngine,
		ledgerService: ledgerService,
		db:            dbConn,
		redis:         redis,
	}
}

func (w *DepositWorkflow) Start(ctx context.Context, errChan chan error) {
	log.Println("DepositWorkflow started")

	ticker := time.NewTicker(10 * time.Second) // every 10 seconds

	for {
		select {
		case <-ctx.Done():
			log.Println("DepositWorkflow stopped")
			errChan <- ctx.Err()
			return
		case <-ticker.C:
			// need reorg?
			needReorg, err := redisclient.IsIndexerPaused(ctx, w.redis)
			if err != nil {
				log.Println("Error checking indexer paused status:", err)
				continue
			}
			if needReorg {
				log.Println("Reorg detected, starting reorg handling")
				continue
			} else {
				log.Println("No reorg detected, processing deposits")
				if err := w.confirmAndApply(ctx); err != nil {
					log.Println("Error in DepositWorkflow: ", err)
				}
			}
		}
	}
}

func (w *DepositWorkflow) reorgnization(ctx context.Context) error {
	msgID, startNumber, err := redisclient.FetchReorg(ctx, w.redis)
	if err != nil {
		log.Println("Error fetching reorg job:", err)
		return err
	}
	log.Printf("Handling reorg from block number: %d", startNumber)

	eth := w.depositEngine.Client.Eth
	for blockNumber := startNumber; ; blockNumber-- {
		block, err := eth.HeaderByNumber(ctx, big.NewInt(int64(blockNumber)))
		if err != nil {
			log.Println("Error fetching block during reorg:", err)
			return err
		}

		localHash, err := repository.GetHashByBlockNumber(ctx, w.db, blockNumber)
		if localHash == block.Hash().Hex() {
			break // no more reorg needed
		}

		log.Printf("Reverting deposits in block number: %d", blockNumber)
		err = w.RevertDepositsByBlockNumber(ctx, blockNumber)
		if err != nil {
			log.Println("Error reverting deposits during reorg:", err)
			return err
		}
	}

	redisclient.AcknowledgeReorg(ctx, w.redis, msgID)
	redisclient.ResumeIndexer(ctx, w.redis)
	log.Println("Reorg handling completed, indexer resumed")
	return nil
}

func (w *DepositWorkflow) RevertDepositsByBlockNumber(ctx context.Context, blockNumber uint64) error {
	// TODO
	return nil
}

func (w *DepositWorkflow) confirmAndApply(ctx context.Context) error {
	deposits, err := w.depositEngine.ListConfirmable(ctx)
	if err != nil {
		return err
	}

	for _, d := range deposits {

		tx, err := w.db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("Error starting transaction for deposit %d: %v", d.ID, err)
			continue
		}
		defer tx.Rollback()

		if err := w.depositEngine.MarkConfirmedTx(ctx, d.ID, tx); err != nil {
			log.Printf("Error marking deposit %d as confirmed: %v", d.ID, err)
			continue
		}

		userID, err := w.depositEngine.GetUserIDFromDepositTx(ctx, d, tx)
		if err != nil {
			log.Printf("Error getting userId from deposit %d : %v", d.ID, err)
			continue
		}

		if err := w.ledgerService.ApplyDepositTx(
			ctx,
			d.ID,
			userID,
			indexer.ASSET_USDC,
			d.Amount,
			tx,
		); err != nil {
			log.Printf("Error applying deposit %d to ledger: %v", d.ID, err)
			continue
		}

		tx.Commit()
		log.Printf("Deposit %d confirmed and applied to ledger", d.ID)
	}

	return nil
}
