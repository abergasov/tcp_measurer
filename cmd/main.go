package main

import (
	"bitbucket.org/Taal_Orchestrator/orca-std-go/logger"
	"context"
	"github.com/joho/godotenv"
	"log"
	"log/slog"
	tcpmeasurer "orchestrator/common/pkg/tcp_measurer"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

var (
	appPortStr  = "8080"
	envFilePath = "/home/taal/.env"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	envFile, _ := godotenv.Read(envFilePath)
	appLogger := newLogger(envFile["COIN"])
	appPort, err := strconv.Atoi(appPortStr)
	if err != nil {
		appLogger.Fatal("unable to parse app port", err)
	}
	appLogger.Info("app starting", slog.String("port", appPortStr))

	srv := tcpmeasurer.NewService(ctx, appLogger, uint64(appPort))
	if err = srv.Init(); err != nil {
		appLogger.Fatal("unable to init service", err)
	}

	go func() {
		if err = srv.Start(); err != nil {
			appLogger.Fatal("unable to start service", err)
		}
	}()

	// register app shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c // This blocks the main thread until an interrupt is received
}

func newLogger(coinName string) logger.AppLogger {
	appLogger, err := logger.NewAppSLogger(
		&logger.Config{
			Progname: "orca_tcp_measurer",
		},
		"",
		logger.WithCoinS(strings.ToUpper(coinName)),
	)
	if err != nil {
		log.Fatalf("Failed to create logger: %s", err)
	}
	if coinName == "" {
		appLogger.Fatal("coin name is empty", nil)
	}
	return appLogger
}
