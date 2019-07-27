package server

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"net/http"
)

func Respond(w http.ResponseWriter, status int, data interface{}) error {
	var err error
	var body []byte

	if data == nil {
		w.Header().Set("content-type", "text/plain")
		body = make([]byte, 0)
	} else if err, ok := data.(error); ok {
		w.Header().Set("content-type", "text/plain")
		body = []byte(err.Error())
	} else if str, ok := data.(string); ok {
		w.Header().Set("content-type", "text/plain")
		body = []byte(str)
	} else if raw, ok := data.([]byte); ok {
		w.Header().Set("content-type", "application/json")
		body = raw
	} else {
		body, err = json.Marshal(data)
		if err != nil {
			err = errors.Wrap(err, "Error marshalling data")
			w.Header().Set("content-type", "text/plain")
			body = []byte("Marshalling error")
		} else {
			w.Header().Set("content-type", "application/json")
		}
	}

	w.WriteHeader(status)
	_, err = w.Write(body)
	if err != nil {
		log.Error().Err(err).Msg("Error writing response")
	}
	return err
}
