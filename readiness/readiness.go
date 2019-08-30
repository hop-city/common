package readiness

import (
	"github.com/hop-city/common/logger"
	"sync"
)

type (
	status struct {
		mu    sync.Mutex
		ready map[string]bool
	}
)

var data = status{ready: make(map[string]bool)}
var logg = logger.New()

func Set(k string, v bool) {
	logg.Info().Msgf(">> %s - readiness = %t", k, v)
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
			logg.Info().Msgf(">> %s - is not ready yet", k)
			ready = false
		}
	}
	return ready
}
