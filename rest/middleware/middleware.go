package middleware

import (
	"github.com/rs/zerolog"
	"net/http"
)

// TODO(mb) add moving headers to ctx

func Ping(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/ping" {
			log := zerolog.Ctx(r.Context())
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("pong"))
			if err != nil {
				log.Error().Err(err).Msg("Error writing pong :(")
			} else {
				log.Info().Msg("/ping -> pong")
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}
