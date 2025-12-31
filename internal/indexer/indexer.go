package indexer

import (
	"context"
	"database/sql"
	"log"
	"math/big"
	"time"

	"dex-indexer/internal/chain"
	"dex-indexer/internal/config"
	"dex-indexer/internal/ledger"
	"dex-indexer/internal/middleware/db/repository"
	redisclient "dex-indexer/internal/middleware/redis/action"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
)

type Indexer struct {
	cfg            *config.Config
	Client         *chain.Client
	db             *sql.DB
	redis          *redis.Client
	DepositEngine  *DepositEngine
	WithdrawEngine *WithdrawEngine
}

func New(
	cfg *config.Config,
	db *sql.DB,
	redis *redis.Client,
	ls *ledger.LedgerService,
) *Indexer {
	client := chain.NewClient(cfg.RPC)

	return &Indexer{
		cfg:            cfg,
		Client:         client,
		db:             db,
		redis:          redis,
		DepositEngine:  NewDepositEngine(cfg, client, db),
		WithdrawEngine: NewWithdrawEngine(cfg, ls, db),
	}
}

func (i *Indexer) Start(ctx context.Context, errChan chan error) {
	log.Println("Indexer started")

	// Prepare for ERC20 transfer indexing
	token := common.HexToAddress(i.cfg.USDC_TOKEN)
	parsedABI, err := i.loadTransferABI()
	if err != nil {
		log.Println("Error loading transfer ABI:", err)
		errChan <- err
		return
	}

	ticker := time.NewTicker(5 * time.Second) // every 5 seconds

	for {
		select {
		case <-ctx.Done():
			log.Println("Indexer stopped")
			errChan <- ctx.Err()
			return
		case <-ticker.C:
			needReorg, err := redisclient.IsIndexerPaused(ctx, i.redis)
			if err != nil {
				log.Println("Error checking indexer paused status:", err)
				continue
			}
			if needReorg {
				continue // Skip processing new blocks during reorg
			}
			// Get current indexer height
			height, err := redisclient.GetIndexerBlockHeight(ctx, i.redis)
			if err != nil {
				log.Println("Error getting indexer block height:", err)
				continue
			}
			latestBlock, err := i.Client.Eth.BlockNumber(ctx)
			if err != nil {
				log.Println("Error getting latest block number:", err)
				continue
			}
			if height == 0 {
				height = latestBlock // start from the latest if not set
			} else {
				height = min(height, latestBlock) // ensure we don't go past latest - 1
			}
			if err := i.processBlock(ctx, token, parsedABI, height); err != nil {
				log.Println("Error processing block:", err)
				continue
			}
		}
	}
}

func (i *Indexer) processBlock(
	ctx context.Context,
	token common.Address,
	parsedABI abi.ABI,
	number uint64,
) error {

	block, err := i.Client.Eth.HeaderByNumber(ctx, big.NewInt(int64(number)))
	if err != nil {
		log.Printf("Error fetching block %d: %v", number, err)
		return err
	}

	// 1. Reorg detect
	reorgNumber, err := i.DetectReorg(ctx, number, block.ParentHash.Hex(), block.Hash().Hex())
	if err != nil {
		log.Println("Error detecting reorg:", err)
		return err
	}
	if reorgNumber != 0 {
		log.Println("Reorg detected at block:", reorgNumber)
		// stop fetching new blocks
		err = redisclient.PauseIndexer(ctx, i.redis)
		if err != nil {
			log.Println("Error pausing indexer:", err)
			return err
		}
		if err := redisclient.PublishReorg(ctx, reorgNumber, i.redis); err != nil {
			log.Println("Error publishing reorg:", err)
			return err
		}
		return nil
	}

	// 2. Process ERC20 transfer logs
	err = i.processBlockRange(
		ctx,
		parsedABI,
		token,
		parsedABI.Events["Transfer"].ID,
		number,
		number,
	)
	if err != nil {
		log.Printf("Error processing block range %d to %d: %v", number, number, err)
		return err
	}

	// 3. Save block info
	err = repository.SaveBlock(
		ctx,
		i.db,
		number,
		block.Hash().Hex(),
		block.ParentHash.Hex(),
	)
	if err != nil {
		log.Println("Error saving block info:", err)
		return err
	}

	// 4. Update indexer height in Redis
	err = redisclient.SetIndexerBlockHeight(ctx, number+1, i.redis)
	if err != nil {
		log.Println("Error setting indexer block height:", err)
		return err
	}

	return nil
}

func (i *Indexer) DetectReorg(
	ctx context.Context,
	number uint64,
	parentHashOnChain string,
	blockHashOnChain string,
) (uint64, error) {

	var blockHashLocal string
	blockHashLocal, err := repository.GetHashByBlockNumber(ctx, i.db, number)
	if err != nil {
		log.Printf("Error getting hash by block number %d: %v", number, err)
		return 0, err
	}
	if blockHashLocal == "" {
		// No local record for this block, check parent block
		parentHashLocal, err := repository.GetHashByBlockNumber(ctx, i.db, number-1)
		if err != nil {
			log.Printf("Error getting parent hash by block number %d: %v", number-1, err)
			return 0, err
		}
		if parentHashLocal == "" {
			// No local record for parent block either, cannot determine reorg
			return 0, nil
		}
		if parentHashLocal != parentHashOnChain {
			return number - 1, nil
		}
		return 0, nil
	} else {
		if blockHashLocal != blockHashOnChain {
			return number, nil
		}
		return 0, nil
	}
}
