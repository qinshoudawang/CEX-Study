package api

import (
	"dex-indexer/internal/middleware/db/model"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
)

type WithdrawRequest struct {
	UserID    int64  `json:"user_id" binding:"required,gt=0"`
	Asset     string `json:"asset" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
	ToAddress string `json:"to_address" binding:"required"`
}

type WithdrawResponse struct {
	WithdrawID int64  `json:"withdraw_id"`
	Status     string `json:"status"`
}

type WithdrawHandler struct {
	withdrawService WithdrawService
}

type WithdrawService interface {
	CreateWithdraw(
		userID int64,
		asset string,
		amount *big.Int,
		to string,
	) (int64, error)
}

func NewWithdrawHandler(s WithdrawService) *WithdrawHandler {
	return &WithdrawHandler{
		withdrawService: s,
	}
}

func (h *WithdrawHandler) CreateWithdraw(c *gin.Context) {
	var req WithdrawRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid amount",
		})
		return
	}

	withdrawID, err := h.withdrawService.CreateWithdraw(
		req.UserID,
		req.Asset,
		amount,
		req.ToAddress,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, WithdrawResponse{
		WithdrawID: withdrawID,
		Status:     string(model.WithdrawRequested),
	})
}
