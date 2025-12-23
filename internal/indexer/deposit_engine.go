package indexer

import (
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type DepositEngine struct {
	confirmations     uint64
	pending           map[string]*Deposit
	exchangeAddresses map[string]bool
}

type DepositStatus string

const (
	StatusPending   DepositStatus = "PENDING"
	StatusConfirmed DepositStatus = "CONFIRMED"
)

type Deposit struct {
	TxHash      string
	FromAddress string
	ToAddress   string
	Amount      *big.Int
	BlockNumber uint64
	Status      DepositStatus
}

func NewDepositEngine(confirmations uint64, exchangeAddress map[string]bool) *DepositEngine {
	return &DepositEngine{
		confirmations:     confirmations,
		pending:           make(map[string]*Deposit),
		exchangeAddresses: exchangeAddress,
	}
}

func (de *DepositEngine) OnTransfer(
	txHash string,
	from, to common.Address,
	amount *big.Int,
	blockNumber uint64,
) {
	// Ignore if 'to' address is not an exchange address
	if !de.exchangeAddresses[strings.ToLower(to.Hex())] {
		return
	}
	// Ignore if this transfer is already pending
	if _, exists := de.pending[txHash]; exists {
		return
	}

	de.pending[txHash] = &Deposit{
		TxHash:      txHash,
		FromAddress: from.Hex(),
		ToAddress:   to.Hex(),
		Amount:      amount,
		BlockNumber: blockNumber,
		Status:      StatusPending,
	}

	log.Println("[Pending]", txHash, "amount:", amount)
}

func (de *DepositEngine) Confirm(latestBlock uint64) {
	for txHash, deposit := range de.pending {
		if latestBlock-deposit.BlockNumber >= de.confirmations {
			deposit.Status = StatusConfirmed
			log.Println("[Confirmed]", txHash, "amount:", deposit.Amount)
			// write DB / ledger
			// de.persistConfirmedDeposit(deposit)
			delete(de.pending, txHash)
		}
	}
}
