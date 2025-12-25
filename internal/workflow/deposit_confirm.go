package workflow

import (
	"context"
	"database/sql"
	"dex-indexer/internal/indexer"
	"dex-indexer/internal/ledger"
	"log"
	"time"
)

type DepositConfirmWorkflow struct {
	depositEngine *indexer.DepositEngine
	ledgerService *ledger.LedgerService
	db            *sql.DB
}

func NewDepositConfirmWorkflow(
	depositEngine *indexer.DepositEngine,
	ledgerService *ledger.LedgerService,
	dbConn *sql.DB,
) *DepositConfirmWorkflow {

	return &DepositConfirmWorkflow{
		depositEngine: depositEngine,
		ledgerService: ledgerService,
		db:            dbConn,
	}
}

func (w *DepositConfirmWorkflow) Start(ctx context.Context, errChan chan error) {
	log.Println("DepositConfirmWorkflow started")

	ticker := time.NewTicker(10 * time.Second) // every 10 seconds

	for {
		select {
		case <-ctx.Done():
			log.Println("DepositConfirmWorkflow stopped")
			errChan <- ctx.Err()
			return
		case <-ticker.C:
			if err := w.confirmAndApply(ctx); err != nil {
				log.Println("Error in DepositConfirmWorkflow: ", err)
			}
		}
	}
}

func (w *DepositConfirmWorkflow) confirmAndApply(ctx context.Context) error {
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
