package ledger

import "database/sql"

type EntryType string

const (
	DEPOSIT          EntryType = "DEPOSIT"
	TRADE            EntryType = "TRADE"
	REVERSAL         EntryType = "REVERSAL"
	WITHDRAW_HOLD    EntryType = "WITHDRAW_HOLD"
	WITHDRAW_RELEASE EntryType = "WITHDRAW_RELEASE"
	WITHDRAW_FINAL   EntryType = "WITHDRAW_FINAL"
)

type LedgerService struct {
	db *sql.DB
}

func NewLedgerService(db *sql.DB) *LedgerService {
	return &LedgerService{db: db}
}
