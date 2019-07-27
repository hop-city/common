package app

import (
	"context"
	"github.com/hop-city/common/logger"
	"github.com/rs/zerolog"
	"os"
	"os/signal"
	"syscall"
)

var stopApp = make(chan os.Signal, 1)

func Scaffold() (context.Context, *zerolog.Logger) {
	log := logger.New()
	log.Info().Msgf("-- INITIALIZING app with pid %d", os.Getpid())
	// lifecycle control
	ctx, cancel := context.WithCancel(context.Background())
	ctx = log.WithContext(ctx) // inserts logger into context

	go func() {
		// app termination
		signal.Notify(
			stopApp, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL,
		)
		sgn := <-stopApp
		log.Info().Msgf("-- SHUTTING DOWN app: '%s' signal received", sgn)
		cancel()
	}()

	return ctx, log
}

func Interrupt() {
	pid := os.Getpid()
	pr, _ := os.FindProcess(pid)
	_ = pr.Signal(os.Interrupt)
}
