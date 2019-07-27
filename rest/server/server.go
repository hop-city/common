package server

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"strings"
	"time"
)

type (
	ServerOptions struct {
		Router chi.Router
		Port   string
	}
)

func CreateRouter() chi.Router {
	r := chi.NewRouter()
	return r
}

func Start(ctx context.Context, opt ServerOptions) {
	log := zerolog.Ctx(ctx)

	if opt.Port == "" {
		opt.Port = os.Getenv("PORT")
	}
	if opt.Port == "" {
		opt.Port = "8080"
		log.Info().Msg("Network.StartServer: server port not provided - using default 8080")
	}

	// server
	server := http.Server{
		Addr:              ":" + opt.Port,
		Handler:           opt.Router,
		ReadHeaderTimeout: 30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// start server
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			if !strings.Contains(err.Error(), "Server closed") {
				log.Error().Err(err).Msg("Network.StartServer: Error starting server")
			}
		}
	}()
	log.Info().Msgf("Network.StartServer: starting listening on port %s", opt.Port)

	// stop server
	go func() {
		<-ctx.Done()
		err := server.Close()
		if err != nil {
			log.Error().Err(err).Msg("Network.StartServer: error stopping server")
		}
		log.Info().Msg("Network.StartServer: server shut down")
	}()
}
