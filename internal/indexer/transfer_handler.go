package indexer

import (
	"context"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type TransferEvent struct {
	From  common.Address
	To    common.Address
	Value *big.Int
}

func (i *Indexer) indexERC20Transfers(ctx context.Context, errChan chan error) {
	parsedABI, err := loadTransferABI()
	if err != nil {
		errChan <- err
		return
	}

	transfer := common.HexToAddress(i.cfg.USDC_TOKEN)
	eventID := parsedABI.Events["Transfer"].ID

	lastProcessedBlock, err := i.client.Eth.BlockNumber(ctx) // Track the last processed block
	if err != nil {
		errChan <- err
		return
	}
	lastProcessedBlock -= i.cfg.BatchSize + 1

	ticker := time.NewTicker(1 * time.Second) // Polling interval of 1 second
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping ERC20 transfer listener: context canceled")
			errChan <- ctx.Err()
			return
		case <-ticker.C:
			blockNumber, err := i.client.Eth.BlockNumber(ctx)
			if err != nil {
				log.Printf("Error fetching block number: %v", err)
				continue
			}

			fromBlock := lastProcessedBlock + 1
			toBlock := blockNumber

			err = i.processBlockRange(ctx, parsedABI, transfer, eventID, fromBlock, toBlock)
			if err != nil {
				log.Printf("Error processing block range %d to %d: %v", fromBlock, toBlock, err)
				continue
			}

			// Update the last processed block
			lastProcessedBlock = toBlock
		}
	}
}

func (i *Indexer) processBlockRange(
	ctx context.Context,
	parsedABI abi.ABI,
	transfer common.Address,
	eventID common.Hash,
	fromBlock, toBlock uint64,
) error {

	for start := fromBlock; start <= toBlock; start += i.cfg.BatchSize {
		end := min(start+i.cfg.BatchSize-1, toBlock)

		query := ethereum.FilterQuery{
			Addresses: []common.Address{transfer},
			Topics:    [][]common.Hash{{eventID}},
			FromBlock: big.NewInt(int64(start)),
			ToBlock:   big.NewInt(int64(end)),
		}

		logs, err := i.client.Eth.FilterLogs(ctx, query)
		if err != nil {
			log.Printf("Error fetching logs for blocks %d to %d: %v", start, end, err)
			continue
		}

		log.Printf("[%s] found %d transfer logs in blocks %d to %d", transfer.Hex(), len(logs), start, end)

		for _, vLog := range logs {
			i.handleTransfer(ctx, transfer, parsedABI, vLog)
		}
	}
	return nil
}

func (i *Indexer) handleTransfer(
	ctx context.Context,
	transfer common.Address,
	contractAbi abi.ABI,
	vLog types.Log,
) {

	var event TransferEvent

	err := contractAbi.UnpackIntoInterface(&event, "Transfer", vLog.Data)
	if err != nil {
		log.Println("unpack error:", err)
		return
	}

	i.DepositEngine.OnTransfer(
		ctx,
		vLog.TxHash.Hex(),
		transfer,
		common.HexToAddress(vLog.Topics[1].Hex()),
		common.HexToAddress(vLog.Topics[2].Hex()),
		event.Value,
		vLog.BlockNumber,
	)
}

func loadTransferABI() (abi.ABI, error) {
	data, err := os.ReadFile("internal/abi/erc20.json")
	if err != nil {
		return abi.ABI{}, err
	}

	return abi.JSON(strings.NewReader(string(data)))
}
