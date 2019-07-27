package readiness

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
	"net/http"
	"strings"
	"time"
)

func Attach(ctx context.Context, r chi.Router) {
	log = zerolog.Ctx(ctx)

	r.Get("/liveness", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Error().Err(err).Msgf("Error responding to \"%s\" check", r.URL.Path)
		}
	}))

	r.Get("/readiness", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		if IsReady() {
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, err = w.Write([]byte("Not ready"))
		}
		if err != nil {
			log.Error().Err(err).Msgf("Error responding to \"%s\" check", r.URL.Path)
		}
	}))

}

func StartServer(ctx context.Context, port string) {
	log = zerolog.Ctx(ctx)
	r := chi.NewRouter()

	Attach(ctx, r)

	if port == "" {
		port = "8080"
	}

	server := http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Info().Msg("Readiness.StartServer: starting listening on port 8080")
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			// skip error on normal close
			if !strings.Contains(err.Error(), "Server closed") {
				log.Error().Err(err).Msg("Readiness.StartServer: Error starting health check server")
			}
		}
	}()

	// stop server
	go func() {
		<-ctx.Done()
		err := server.Close()
		if err != nil {
			log.Error().Err(err).Msg("Readiness.StartServer: error stopping health check server")
		}
		log.Info().Msg("Readiness.StartServer: health check server shut down")
	}()
}
