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
				if err := w.reorgnization(ctx); err != nil {
					log.Println("Error in DepositWorkflow reorg handling: ", err)
				}
			} else {
				if err := w.confirmAndApply(ctx); err != nil {
					log.Println("Error in DepositWorkflow confirm and apply: ", err)
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
	var blockNumber uint64
	for blockNumber = startNumber; blockNumber > 0; blockNumber-- {
		block, err := eth.HeaderByNumber(ctx, big.NewInt(int64(blockNumber)))
		if err != nil {
			log.Println("Error fetching block during reorg:", err)
			return err
		}

		stored, err := repository.GetHashByBlockNumber(ctx, w.db, blockNumber)
		if err != nil {
			log.Printf("Error getting local block %d hash: %v", blockNumber, err)
			return err
		}
		if stored == "" {
			log.Printf("No local block hash found for block number: %d", blockNumber)
			break
		}
		if stored == block.Hash().Hex() {
			log.Printf("Reorg handling reached common ancestor at block number: %d", blockNumber)
			break
		}

		log.Printf("Reverting deposits in block number: %d", blockNumber)
		err = w.RevertDepositsByBlockNumber(ctx, blockNumber)
		if err != nil {
			log.Println("Error reverting deposits during reorg:", err)
			return err
		}

		log.Printf("Reset block record for block number: %d", blockNumber)
		err = repository.SaveBlock(
			ctx,
			w.db,
			blockNumber,
			block.Hash().Hex(),
			block.ParentHash.Hex(),
		)
		if err != nil {
			log.Println("Error resetting block record during reorg:", err)
			return err
		}
	}

	err = redisclient.AcknowledgeReorg(ctx, w.redis, msgID)
	if err != nil {
		log.Println("Error acknowledging reorg job:", err)
		return err
	}
	err = redisclient.SetIndexerBlockHeight(ctx, blockNumber, w.redis)
	if err != nil {
		log.Println("Error setting indexer block height:", err)
		return err
	}
	err = redisclient.ResumeIndexer(ctx, w.redis)
	if err != nil {
		log.Println("Error resuming indexer:", err)
		return err
	}

	log.Println("Reorg handling completed, indexer resumed")
	return nil
}

func (w *DepositWorkflow) RevertDepositsByBlockNumber(ctx context.Context, blockNumber uint64) error {

	deposits, err := w.depositEngine.ListDepositsByBlockNumber(ctx, blockNumber)
	if err != nil {
		log.Printf("Error listing deposits with block number %d: %v", blockNumber, err)
		return err
	}

	for _, d := range deposits {

		tx, err := w.db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("Error revert deposit with block number %d: %v", blockNumber, err)
			return err
		}
		defer tx.Rollback()

		userID, err := w.depositEngine.GetUserIDFromDepositTx(ctx, d, tx)
		if err != nil {
			log.Printf("Error getting userId from deposit %d : %v", d.ID, err)
			continue
		}

		err = w.ledgerService.RevertDepositTx(
			ctx,
			d.ID,
			userID,
			indexer.ASSET_USDC,
			d.Amount,
			tx,
		)
		if err != nil {
			log.Printf("Error reverting deposit %d: %v", d.ID, err)
			return err
		}

		err = w.depositEngine.MarkRevertedTx(ctx, d.ID, tx)
		if err != nil {
			log.Printf("Error marking deposit %d as reverted: %v", d.ID, err)
			return err
		}

		tx.Commit()
		log.Printf("Deposit %d reverted successfully", d.ID)
	}

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
