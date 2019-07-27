package app

import (
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"syscall"
	"testing"
	"time"
)

func TestScaffold(t *testing.T) {
	ctx, log := Scaffold()

	assert.Equal(t, zerolog.Ctx(ctx), log,
		"Logger should be the same")

	for {
		select {
		case <-time.After(time.Millisecond * 10):
			assert.Fail(t, "Context Done should be closed on interrupt")
			return
		case <-ctx.Done():
			return
		default:
			stopApp <- syscall.SIGINT
		}
	}
}
