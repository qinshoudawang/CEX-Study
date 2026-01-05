package service

import (
	"context"
	"database/sql"
	"dex-indexer/internal/chain"
	"dex-indexer/internal/config"
	"dex-indexer/internal/infra/db/model"
	"dex-indexer/internal/infra/db/repository"
	"dex-indexer/internal/ledger"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type WithdrawServiceImpl struct {
	cfg           config.Config
	ctx           context.Context
	client        *chain.Client
	ledgerService *ledger.LedgerService
	db            *sql.DB
}

func NewWithdrawServiceImpl(
	ctx context.Context,
	client *chain.Client,
	ledgerService *ledger.LedgerService,
	db *sql.DB,
) *WithdrawServiceImpl {

	return &WithdrawServiceImpl{
		ctx:           ctx,
		client:        client,
		ledgerService: ledgerService,
		db:            db,
	}
}

func (w *WithdrawServiceImpl) CreateWithdraw(
	userID int64,
	asset string,
	amount *big.Int,
	to string,
) (int64, error) {

	withdrawID, err := repository.InsertWithdraw(
		w.ctx,
		w.db,
		userID,
		asset,
		amount.String(),
		to,
	)
	if err != nil {
		log.Println("Error inserting withdraw:", err)
		return 0, err
	}

	err = w.ledgerService.HoldWithdraw(w.ctx, userID, asset, amount, withdrawID)
	if err != nil {
		log.Println("Error holding withdraw in ledger:", err)
		return 0, err
	}

	txHash, err := w.BuildAndSendERC20Tx(
		common.HexToAddress(w.cfg.HotWalletAddress),
		common.HexToAddress(to),
		common.HexToAddress(w.cfg.USDC_TOKEN),
		amount,
	)
	if err != nil {
		log.Println("Error building and sending ERC20 tx:", err)
		return 0, err
	}

	err = repository.UpdateWithdrawStatus(
		w.ctx,
		w.db,
		withdrawID,
		model.WithdrawSent,
		txHash,
	)
	if err != nil {
		log.Println("Error updating withdraw status to SENT:", err)
		return 0, err
	}

	log.Printf("Withdraw %d created and sent with tx hash %s", withdrawID, txHash)

	return withdrawID, nil
}

func (w *WithdrawServiceImpl) BuildAndSendERC20Tx(
	hotWallet common.Address,
	to common.Address,
	token common.Address,
	amount *big.Int,
) (string, error) {

	// TODO: need to get redis lock for hot wallet

	// Alchemy couldn't handle concurrent nonce correctly
	// nonce, err := w.client.Eth.PendingNonceAt(w.ctx, hotWallet)
	// if err != nil {
	// 	log.Println("Error getting nonce:", err)
	// 	return err
	// }

	// function selector for transfer(address,uint256)
	data := crypto.Keccak256([]byte("transfer(address,uint256)"))[:4]
	data = append(data, common.LeftPadBytes(to.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)

	tip, _ := w.client.Eth.SuggestGasTipCap(w.ctx)
	fee, _ := w.client.Eth.SuggestGasPrice(w.ctx)

	tx := types.NewTransaction(
		15,
		token,
		big.NewInt(0),
		tip.Uint64(),
		fee,
		data,
	)

	chainId, err := w.client.Eth.ChainID(w.ctx)
	if err != nil {
		log.Println("Error getting chain ID:", err)
		return "", err
	}

	privateKeyHex := os.Getenv("PRIVATE_KEY")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Println("Error parsing private key:", err)
		return "", err
	}

	signedTx, err := types.SignTx(
		tx,
		types.NewLondonSigner(chainId),
		privateKey,
	)
	if err != nil {
		log.Println("Error signing transaction:", err)
		return "", err
	}

	err = w.client.Eth.SendTransaction(w.ctx, signedTx)
	if err != nil {
		log.Println("Error sending transaction:", err)
		return "", err
	}

	return signedTx.Hash().Hex(), nil
}
