package indexer

import (
	"context"
	"database/sql"
	"log"
	"math/big"
	"strings"
	"time"

	"dex-indexer/internal/chain"
	"dex-indexer/internal/config"
	"dex-indexer/internal/middleware/db/repository"
	redisclient "dex-indexer/internal/middleware/redis/action"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
)

type Indexer struct {
	cfg           *config.Config
	client        *chain.Client
	db            *sql.DB
	redis         *redis.Client
	DepositEngine *DepositEngine
}

func New(cfg *config.Config, db *sql.DB, redis *redis.Client) *Indexer {
	client := chain.NewClient(cfg.RPC)

	return &Indexer{
		cfg:           cfg,
		client:        client,
		db:            db,
		redis:         redis,
		DepositEngine: NewDepositEngine(cfg, client, db),
	}
}

func (i *Indexer) Start(ctx context.Context, errChan chan error) {
	log.Println("Indexer started")

	// start from the latest block‘s parent
	height, err := i.client.Eth.BlockNumber(ctx)
	if err != nil {
		errChan <- err
		return
	}
	height--

	// Prepare for ERC20 transfer indexing
	transfer := common.HexToAddress(i.cfg.USDC_TOKEN)
	parsedABI, err := i.loadTransferABI()
	if err != nil {
		log.Println("Error loading transfer ABI:", err)
		errChan <- err
	}

	ticker := time.NewTicker(time.Second) // every 1 seconds

	for {
		select {
		case <-ctx.Done():
			log.Println("Indexer stopped")
			errChan <- ctx.Err()
			return
		case <-ticker.C:
			if err := i.processBlock(ctx, transfer, parsedABI, height); err != nil {
				if strings.Contains(err.Error(), "not found") {
					// Block not yet mined, wait for the next tick
					continue
				}
				log.Println("Error processing block:", err)
				continue
			}
			height++
		}
	}
}

func (i *Indexer) processBlock(
	ctx context.Context,
	transfer common.Address,
	parsedABI abi.ABI,
	number uint64,
) error {

	block, err := i.client.Eth.HeaderByNumber(ctx, big.NewInt(int64(number)))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return err // Block not yet mined, just return
		}
		log.Printf("Error fetching block %d: %v", number, err)
		return err
	}

	// 1. Reorg detect
	needReorg, err := i.DetectReorg(ctx, number-1, block.ParentHash.Hex())
	if err != nil {
		log.Println("Error detecting reorg:", err)
		return err
	}
	if needReorg {
		log.Println("Reorg detected at block:", number-1)
		// stop fetching new blocks
		err = redisclient.PauseIndexer(ctx, i.redis)
		if err != nil {
			log.Println("Error pausing indexer:", err)
			return err
		}
		if err := redisclient.PublishReorg(ctx, number-1, i.redis); err != nil {
			log.Println("Error publishing reorg:", err)
			return err
		}
	}

	// 2. Process ERC20 transfer logs
	err = i.processBlockRange(
		ctx,
		parsedABI,
		transfer,
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

	return nil
}

func (i *Indexer) DetectReorg(
	ctx context.Context,
	number uint64,
	parentHash string,
) (bool, error) {

	var stored string
	stored, err := repository.GetHashByBlockNumber(ctx, i.db, number)

	if err != nil {
		log.Printf("Error getting hash by block number %d: %v", number, err)
		return false, err
	}

	if stored == "" {
		log.Println("No stored hash for block number:", number)
		return false, nil
	}

	return stored != parentHash, nil
}
