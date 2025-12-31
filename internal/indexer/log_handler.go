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

const (
	ERC20_ABI_PATH = "internal/chain/abi/erc20.json"
)

type TransferEvent struct {
	From  common.Address
	To    common.Address
	Value *big.Int
}

// deprecated
func (i *Indexer) indexERC20Transfers(ctx context.Context, errChan chan error) {
	parsedABI, err := i.loadTransferABI()
	if err != nil {
		errChan <- err
		return
	}

	transfer := common.HexToAddress(i.cfg.USDC_TOKEN)
	eventID := parsedABI.Events["Transfer"].ID

	lastProcessedBlock, err := i.Client.Eth.BlockNumber(ctx) // Track the last processed block
	if err != nil {
		errChan <- err
		return
	}
	lastProcessedBlock -= i.cfg.BatchSize + 1

	ticker := time.NewTicker(5 * time.Second) // Polling interval of 1 second
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping ERC20 transfer listener: context canceled")
			errChan <- ctx.Err()
			return
		case <-ticker.C:
			blockNumber, err := i.Client.Eth.BlockNumber(ctx)
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
	token common.Address,
	eventID common.Hash,
	fromBlock, toBlock uint64,
) error {

	for start := fromBlock; start <= toBlock; start += i.cfg.BatchSize {
		end := min(start+i.cfg.BatchSize-1, toBlock)

		query := ethereum.FilterQuery{
			Addresses: []common.Address{token},
			Topics:    [][]common.Hash{{eventID}},
			FromBlock: big.NewInt(int64(start)),
			ToBlock:   big.NewInt(int64(end)),
		}

		logs, err := i.Client.Eth.FilterLogs(ctx, query)
		if err != nil {
			log.Printf("Error fetching logs for blocks %d to %d: %v", start, end, err)
			continue
		}

		log.Printf("found %d transfer logs in blocks %d to %d", len(logs), start, end)

		for _, vLog := range logs {
			i.handleTransfer(ctx, token, parsedABI, &vLog)
		}
	}
	return nil
}

func (i *Indexer) handleTransfer(
	ctx context.Context,
	token common.Address,
	contractAbi abi.ABI,
	vLog *types.Log,
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
		token,
		common.HexToAddress(vLog.Topics[1].Hex()),
		common.HexToAddress(vLog.Topics[2].Hex()),
		event.Value,
		vLog.BlockNumber,
	)

	i.WithdrawEngine.OnTransfer(
		ctx,
		vLog.TxHash.Hex(),
		common.HexToAddress(vLog.Topics[1].Hex()),
		event.Value,
		vLog.BlockNumber,
	)
}

func (i *Indexer) loadTransferABI() (abi.ABI, error) {
	data, err := os.ReadFile(ERC20_ABI_PATH)
	if err != nil {
		return abi.ABI{}, err
	}

	return abi.JSON(strings.NewReader(string(data)))
}
