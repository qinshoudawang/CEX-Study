package api

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	r *gin.Engine,
	withdrawHandler *WithdrawHandler,
) {

	v1 := r.Group("/api/v1")
	{
		v1.POST("/withdraw", withdrawHandler.CreateWithdraw)
	}
}
