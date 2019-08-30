package main

import "github.com/hop-city/common/logger"

func main() {
	l := logger.New()
	l.Debug().Msg("debug")
	l.Info().Msg("info")
	l.Warn().Msg("warn")
}
