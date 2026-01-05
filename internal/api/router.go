package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	r *gin.Engine,
	withdrawHandler *WithdrawHandler,
	promhttpHandler http.Handler,
) {

	v1 := r.Group("/api/v1")
	{
		v1.POST("/withdraw", withdrawHandler.CreateWithdraw)
		v1.GET("/metrics", gin.WrapH(promhttpHandler))
	}
}
