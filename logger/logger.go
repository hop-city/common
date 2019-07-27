package logger

import (
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
	zero "github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"time"
)

type arguments struct {
	LogLevel  string `env:"LOG_LEVEL" short:"l" long:"log-level" default:"info"`
	PrettyLog string `env:"LOG_PRETTY" short:"p" long:"pretty"`
	Caller    string `env:"LOG_CALLER" long:"caller" description:"Shall I log caller?"`
	Revision  string `env:"REVISION" long:"revision" default:""`
}

var args = arguments{}
var parsed = false
var output io.Writer

func New() *zerolog.Logger {
	setupEnvironment()
	defer func() {
		parsed = true
	}()

	ctx := zerolog.New(output).With().Timestamp()
	if args.Caller != "" {
		ctx = ctx.Caller()
	}
	if args.Revision != "" {
		ctx = ctx.Str("revision", args.Revision)
	}
	logger := ctx.Logger()

	return &logger
}

func setupEnvironment() {
	if parsed {
		return
	}
	args.LogLevel = os.Getenv("LOG_LEVEL")
	args.PrettyLog = os.Getenv("LOG_PRETTY")
	args.Caller = os.Getenv("LOG_CALLER")
	args.Revision = os.Getenv("REVISION")

	if args.PrettyLog != "" {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else {
		output = os.Stdout
	}

	level, err := zerolog.ParseLevel(args.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
		zero.Error().Err(err).Msg("Logger: Invalid error level provided")
	}
	zerolog.SetGlobalLevel(level)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := New()
		ctx = l.WithContext(ctx)

		reqID := middleware.GetReqID(ctx)
		if reqID != "" {
			l.UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Str("requestId", reqID)
			})
		}
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
