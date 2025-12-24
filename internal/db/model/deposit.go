package model

import "math/big"

type DepositStatus string

const (
	DepositPending   DepositStatus = "PENDING"
	DepositConfirmed DepositStatus = "CONFIRMED"
)

type Deposit struct {
	ID           int64
	TxHash       string
	BlockNumber  uint64
	TokenAddress string
	FromAddress  string
	ToAddress    string
	Amount       *big.Int
	Status       DepositStatus
}
