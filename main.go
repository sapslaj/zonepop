package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sapslaj/zonepop/config"
	"github.com/sapslaj/zonepop/controller"
)

var (
	configFileName = flag.String("config-file", "config.lua", "Path to configuration file (default: config.lua)")
	interval       = flag.Duration("interval", 1*time.Minute, "The interval between two consecutive synchronizations in duration format (default: 1m)")
	once           = flag.Bool("once", false, "When enabled, exits the synchronization loop after the first iteration (default: disabled)")
	dryRun         = flag.Bool("dry-run", false, "When enabled, prints DNS record changes rather than actually performing them (default: disabled)")
)

func main() {
	log.Printf("Starting ZonePop v" + VERSION)
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	go handleSigterm(cancel)

	if *dryRun {
		ctx = context.WithValue(ctx, config.DryRunContextKey, true)
	}

	c, err := config.NewLuaConfig(*configFileName)
	if err != nil {
		panic(err)
	}
	err = c.Parse()
	if err != nil {
		panic(err)
	}
	sources, err := c.Sources()
	if err != nil {
		panic(err)
	}
	providers, err := c.Providers()
	if err != nil {
		panic(err)
	}

	ctrl := controller.Controller{
		Sources:   sources,
		Providers: providers,
		Interval:  *interval,
	}

	if *once {
		err := ctrl.RunOnce(ctx)
		if err != nil {
			panic(err)
		}
		os.Exit(0)
		return
	}

	ctrl.ScheduleRunOnce(time.Now())
	ctrl.Run(ctx)
}

func handleSigterm(cancel func()) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals
	log.Printf("Received SIGTERM. Terminating...")
	cancel()
}
