package main

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/config"
	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/controller"
	"github.com/sapslaj/zonepop/pkg/log"
)

var (
	configFileName = flag.String("config-file", "config.lua", "Path to configuration file (default: config.lua)")
	interval       = flag.Duration("interval", 1*time.Minute, "The interval between two consecutive synchronizations in duration format (default: 1m)")
	once           = flag.Bool("once", false, "When enabled, exits the synchronization loop after the first iteration (default: disabled)")
	dryRun         = flag.Bool("dry-run", false, "When enabled, prints DNS record changes rather than actually performing them (default: disabled)")
)

func main() {
	logger := log.MustNewLogger().Named("main")
	defer func() {
		err := logger.Sync()
		var perr *fs.PathError
		if !errors.As(err, &perr) {
			panic(err)
		}
	}()
	logger.Info("Starting ZonePop v" + VERSION)

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	go handleSigterm(cancel, logger)

	if *dryRun {
		ctx = context.WithValue(ctx, configtypes.DryRunContextKey, true)
	}

	c, err := config.NewLuaConfig(*configFileName)
	if err != nil {
		logger.Sugar().Panicf("could not create new configuration: %v", err)
	}
	err = c.Parse()
	if err != nil {
		logger.Sugar().Panicf("could not parse configuration: %v", err)
	}
	sources, err := c.Sources()
	if err != nil {
		logger.Sugar().Panicf("could not get sources from configuration: %v", err)
	}
	providers, err := c.Providers()
	if err != nil {
		logger.Sugar().Panicf("could not get providers from configuration: %v", err)
	}

	ctrl := controller.Controller{
		Sources:   sources,
		Providers: providers,
		Interval:  *interval,
		Logger:    logger.Named("controller"),
	}

	if *once {
		err := ctrl.RunOnce(ctx)
		if err != nil {
			logger.Sugar().Panicf("could not execute controller loop: %v", err)
		}
		return
	}

	ctrl.ScheduleRunOnce(time.Now())
	ctrl.Run(ctx)
}

func handleSigterm(cancel func(), logger *zap.Logger) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals
	logger.Info("Received SIGTERM. Terminating...")
	cancel()
}
