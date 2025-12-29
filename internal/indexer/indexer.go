package indexer

import (
	"context"
	"database/sql"
	"log"
	"math/big"
	"time"

	"dex-indexer/internal/chain"
	"dex-indexer/internal/config"
	"dex-indexer/internal/middleware/db/repository"
	redisclient "dex-indexer/internal/middleware/redis/action"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
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
	log.Println("indexer started")

	// start from the latest block
	height, err := i.client.Eth.BlockNumber(ctx)
	if err != nil {
		errChan <- err
		return
	}
	height -= 1

	// Prepare for ERC20 transfer indexing
	transfer := common.HexToAddress(i.cfg.USDC_TOKEN)
	parsedABI, err := i.loadTransferABI()
	if err != nil {
		log.Println("Error loading transfer ABI:", err)
		errChan <- err
	}

	for {
		next := height + 1

		if err := i.processBlock(ctx, transfer, parsedABI, next); err != nil {
			log.Println("Error processing block:", err)
			time.Sleep(time.Second)
			continue
		}

		height = next
	}
}

func (i *Indexer) processBlock(
	ctx context.Context,
	transfer common.Address,
	parsedABI abi.ABI,
	number uint64,
) error {

	block, err := i.client.Eth.HeaderByNumber(ctx, big.NewInt(int64(number-1)))
	if err != nil {
		log.Println("Error fetching block:", err)
		return err
	}

	// 1. Reorg detect
	needReorg, err := i.DetectReorg(ctx, number-1, block.ParentHash.Hex())
	if err != nil {
		log.Println("Error detecting reorg:", err)
		return err
	}
	if needReorg {
		// stop fetching new blocks
		err = redisclient.PauseIndexer(ctx, i.redis)
		if err != nil {
			log.Println("Error pausing indexer:", err)
			return err
		}
		log.Println("Reorg detected at block:", number-1)
		if err := redisclient.PublishReorg(ctx, number-1, i.redis); err != nil {
			log.Println("Error publishing reorg:", err)
			return err
		}
	}

	// 2. Fetch receipts
	receipts, err := i.client.Eth.BlockReceipts(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(number-1)))
	if err != nil {
		log.Println("Error fetching receipts:", err)
		return err
	}

	// 3. Process logs
	for _, r := range receipts {
		for _, vLog := range r.Logs {
			i.handleTransfer(ctx, transfer, parsedABI, *vLog)
		}
	}

	// 4. Save block info
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
