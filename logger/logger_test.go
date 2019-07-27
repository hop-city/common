package logger

import (
	"context"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var emptyLog *zerolog.Logger

func init() {
	c := context.Background()
	emptyLog = zerolog.Ctx(c)
}

func TestNew(t *testing.T) {
	_ = os.Setenv("LOG_LEVEL", "wrong")
	_ = os.Setenv("LOG_PRETTY", "1")
	_ = os.Setenv("LOG_CALLER", "1")
	_ = os.Setenv("LOG_REVISION", "c345fe23gd")
	log1 := New()

	assert.Truef(t, parsed, "Env was parsed, but flag was not set")
	assert.NotEqual(t, emptyLog, log1,
		"Empty log was returned")

	log2 := New()
	assert.NotEqual(t, emptyLog, log2,
		"Empty log was returned")
	assert.False(t, log1 == log2,
		"New logger should be returned each time New is called")

	_ = os.Unsetenv("LOG_LEVEL")
	_ = os.Unsetenv("LOG_PRETTY")
	_ = os.Unsetenv("LOG_CALLER")
	_ = os.Unsetenv("LOG_REVISION")
}

func TestMiddleware(t *testing.T) {
	server := httptest.NewServer(
		Middleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				lo := zerolog.Ctx(ctx)

				assert.NotEqual(t, emptyLog, lo,
					"Empty log was returned - it means that context didn't contain a logger :(")
			})),
	)
	_, _ = http.Get(server.URL)
	server.Close()
}

func TestMiddleware_reqID(t *testing.T) {
	lastHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			lo := zerolog.Ctx(ctx)

			reqID := middleware.GetReqID(ctx)
			assert.NotEmpty(t, reqID, "Request id should not be empty")
			// we do no real update, but it allows us to access the context
			lo.UpdateContext(func(c zerolog.Context) zerolog.Context {
				LoggerReqID := middleware.GetReqID(ctx)
				assert.Equal(t, reqID, LoggerReqID, "Request id should not be empty")
				return c
			})
			assert.NotEqual(t, emptyLog, lo,
				"Empty log was returned - it means that context didn't contain a logger :(")
		})
	server := httptest.NewServer(
		middleware.RequestID(Middleware(lastHandler)),
	)
	_, _ = http.Get(server.URL)
	server.Close()
}
