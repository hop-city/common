package readiness

import (
	"context"
	"github.com/rs/zerolog"
	"sync"
)

type (
	status struct {
		mu    sync.Mutex
		ready map[string]bool
	}
)

var data = status{ready: make(map[string]bool)}
var log = &zerolog.Logger{}

func Init(ctx context.Context) {
	log = zerolog.Ctx(ctx)
}

func Set(k string, v bool) {
	log.Info().Msgf(">> %s - readiness <- %t", k, v)
	data.mu.Lock()
	data.ready[k] = v
	data.mu.Unlock()
}

func IsReady() bool {
	ready := true
	data.mu.Lock()
	defer data.mu.Unlock()
	for k, v := range data.ready {
		if !v {
			log.Info().Msgf(">> %s - is not ready yet", k)
			ready = false
		}
	}
	return ready
}
