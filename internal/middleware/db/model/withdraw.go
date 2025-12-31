package model

import "time"

type WithdrawStatus string

const (
	WithdrawRequested WithdrawStatus = "REQUESTED"
	WithdrawSent      WithdrawStatus = "SENT"
	WithdrawConfirmed WithdrawStatus = "CONFIRMED"
	WithdrawFailed    WithdrawStatus = "FAILED"
)

type Withdraw struct {
	ID          int64
	UserID      int64
	Asset       string
	Amount      string
	ToAddress   string
	Status      WithdrawStatus
	TxHash      string
	CreatedAt   time.Time
	ConfirmedAt *time.Time
}
