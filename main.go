package main

import (
	"context"
	"github.com/hop-city/common/backoff"
	"github.com/hop-city/common/logger"
)

func main() {
	l := logger.New()
	l.Debug().Msg("debug")
	l.Info().Msg("info")
	l.Warn().Msg("warn")

	ctx := context.Background()
	bo := backoff.New().
		SetBaseDuration(100).
		SetJitter(0.2).
		SetMaxDelay(100e3)
	<-bo.Wait(ctx, 3)
}
