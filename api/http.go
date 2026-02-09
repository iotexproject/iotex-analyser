package api

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/iotexproject/iotex-analyser/plugins/block_receipts_transaction/api"
	"github.com/pkg/errors"
)

type httpServer struct {
	router gin.IRoutes
}

func (s *httpServer) ready(c *gin.Context) {
	c.Status(http.StatusOK)
}

func Run(addr string) error {
	e := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = append(config.AllowHeaders, "Authorization")
	router := e.Use(cors.New(config))

	s := &httpServer{
		router: router,
	}

	s.router.GET("/v1/ready", s.ready)
	s.router.GET("/v1/block-receipt-transaction/total-amount", api.TotalAmount)

	err := e.Run(addr)
	return errors.Wrap(err, "failed to start http server")
}
