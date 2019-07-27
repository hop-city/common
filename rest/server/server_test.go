package server

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

var lastReq *http.Request

func setup() (context.Context, func(), chi.Router) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(context.Background())
	log := zerolog.New(os.Stdout)
	ctx = log.WithContext(ctx)

	r := CreateRouter()
	r.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastReq = r
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello!"))
	}))
	return ctx, cancel, r
}

func TestStart(t *testing.T) {
	ctx, cancel, r := setup()
	defer cancel()
	Start(ctx, ServerOptions{r, "9019"})

	resp, err := http.Get("http://localhost:9019")
	assert.NoError(t, err, "We should be able to listen on custom port")
	if err != nil {
		return
	}
	b, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "hello!", string(b), "Incorrect body returned")
}

func TestStartport(t *testing.T) {
	ctx, cancel, r := setup()
	defer cancel()

	Start(ctx, ServerOptions{r, ""})

	resp, err := http.Get("http://localhost:8080")
	assert.NoError(t, err, "We should be able to listen on default port")
	if err != nil {
		return
	}
	b, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "hello!", string(b), "Incorrect body returned")
}
