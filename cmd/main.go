package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	tcpmeasurer "orchestrator/common/pkg/tcp_measurer"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"bitbucket.org/Taal_Orchestrator/orca-std-go/logger"
)

var (
	appPortStr = "8080"
	skipCMD    = "0"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	appLogger := newLogger()
	appPort, err := strconv.Atoi(appPortStr)
	if err != nil {
		appLogger.Fatal("unable to parse app port", err)
	}
	appLogger.Info("app starting", slog.String("port", appPortStr))

	srv := tcpmeasurer.NewService(ctx, appLogger, uint64(appPort), tcpmeasurer.WithSkipCMD(skipCMD))
	if err = srv.Init(); err != nil {
		appLogger.Fatal("unable to init service", err)
	}

	go func() {
		time.Sleep(6 * time.Hour)
		appLogger.Fatal("app shutting down due to timeout", fmt.Errorf("timeout"))
	}()

	go func() {
		if err = srv.Start(); err != nil {
			appLogger.Fatal("unable to start service", err)
		}
	}()

	// register app shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c // This blocks the main thread until an interrupt is received
	appLogger.Info("app shutting down")
	cancel()
	srv.Stop()
}

func newLogger() logger.AppLogger {
	appLogger, err := logger.NewAppSLogger(
		&logger.Config{
			Progname: "orca_tcp_measurer",
		},
		"",
	)
	if err != nil {
		log.Fatalf("Failed to create logger: %s", err)
	}
	return appLogger
}
