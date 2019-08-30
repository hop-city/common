package logger

import (
	"github.com/go-chi/chi/middleware"
	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	zero "github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"time"
)

type arguments struct {
	LogLevel string `env:"LOG_LEVEL" short:"l" long:"log-level" default:"info" description:"Minimum logging level"`
	Pretty   bool   `env:"LOG_PRETTY" short:"p" long:"log-pretty" description:"Will skipp JSON logging and in favor of colour output"`
	Caller   bool   `env:"LOG_CALLER" short:"c" long:"log-caller" description:"Will log file and line"`
	Revision string `env:"LOG_REVISION" long:"log-revision"`
}

var args = arguments{}
var output io.Writer

func init() {
	parser := flags.NewParser(&args, flags.Default|flags.IgnoreUnknown)
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}

	if args.Pretty {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else {
		output = os.Stdout
	}

	level, err := zerolog.ParseLevel(args.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
		zero.Error().Err(err).Msg("Logger: Invalid error level provided")
		os.Exit(1)
	}
	zerolog.SetGlobalLevel(level)
}

func New() *zerolog.Logger {
	ctx := zerolog.New(output).With().Timestamp()
	if args.Caller {
		ctx = ctx.Caller()
	}
	if args.Revision != "" {
		ctx = ctx.Str("revision", args.Revision)
	}
	logger := ctx.Logger()

	return &logger
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
