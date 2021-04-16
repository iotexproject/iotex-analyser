package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/server"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var Server = &cli.Command{
	Name:        "server",
	Usage:       "start server",
	Description: `start server`,
	Action:      runServer,
}

func runServer(c *cli.Context) error {

	srv := server.New()
	go func() {
		ctx := kernel.WithConfigCtx(c.Context, c.String("config"))
		if err := srv.Start(ctx); err != nil {
			log.L().Fatal("Failed to start the indexer", zap.Error(err))
		}
	}()

	return handleShutdown(c.Context, srv)
}

type Stopper interface {
	Stop(context.Context) error
}

func handleShutdown(ctx context.Context, service ...Stopper) error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	log.L().Info("shutting down ...")
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	for _, s := range service {
		if err := s.Stop(ctx); err != nil {
			return err
		}
	}
	return nil
}
