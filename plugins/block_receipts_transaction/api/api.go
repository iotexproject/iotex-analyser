package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iotexproject/iotex-analyser/api/common"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugins/block_receipts_transaction/model"
	"github.com/shopspring/decimal"
)

type totalAmountReq struct {
	Sender string `form:"sender"            binding:"required,max=42"`
}

type totalAmountResp struct {
	Amount decimal.Decimal `form:"amount"         `
}

func TotalAmount(c *gin.Context) {
	req := &totalAmountReq{}
	if err := c.ShouldBindQuery(req); err != nil {
		slog.Error("failed to bind request", "error", err)
		c.JSON(http.StatusBadRequest, common.NewErrResp("Invalid request payload"))
		return
	}

	var total decimal.Decimal
	if err := db.DB().Model(&model.BlockReceiptTransaction{}).Where("sender = ?", req.Sender).
		Pluck("COALESCE(SUM(amount),0)", &total).Error; err != nil {
		slog.Error("failed to sum amount", "error", err)
		c.JSON(http.StatusInternalServerError, common.NewErrResp("Failed to sum amount"))
		return
	}
	c.JSON(http.StatusOK, &totalAmountResp{
		Amount: total,
	})
}
