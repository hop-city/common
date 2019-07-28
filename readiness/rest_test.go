package readiness

import (
	"context"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

func setup() (context.Context, func()) {
	<-time.After(time.Millisecond * 20)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(context.Background())
	log := zerolog.New(os.Stdout)
	ctx = log.WithContext(ctx)
	_ = Handler(ctx)
	clearStatuses()
	return ctx, cancel
}

func body(resp *http.Response) string {
	b, _ := ioutil.ReadAll(resp.Body)
	return string(b)
}

const end = "end"

var endHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(end))
})

func TestStartServer_liveness(t *testing.T) {
	ctx, cancel := setup()
	StartServer(ctx, "")
	resp, err := http.Get("http://localhost:8080/liveness")
	assert.Nil(t, err, "Server should start")
	if resp != nil {
		assert.Equal(t,
			http.StatusOK, resp.StatusCode,
			"Liveness should return 200 when server is alive")
		assert.Equal(t, "OK", body(resp),
			"Incorrect body, 'OK' expected")
	}
	cancel()
}
func TestStartServer_readiness(t *testing.T) {
	ctx, cancel := setup()
	StartServer(ctx, "")
	resp, err := http.Get("http://localhost:8080/readiness")
	assert.Nil(t, err, "Server should start")
	if resp != nil {
		assert.Equal(t,
			http.StatusOK, resp.StatusCode,
			"Readiness should return 200 when nothing is registered")
		assert.Equal(t, "OK", body(resp),
			"Incorrect body, 'OK' expected")
	}
	Set("a", false)
	resp, err = http.Get("http://localhost:8080/readiness")
	assert.Nil(t, err, "Should be able to connect")

	if resp != nil {
		assert.Equal(t,
			http.StatusServiceUnavailable, resp.StatusCode,
			"Readiness should return 503 when not ready")
		assert.Equal(t, "Not ready", body(resp),
			"Incorrect body, 'OK' expected")
	}
	Set("b", false)
	Set("a", true)
	Set("b", true)
	resp, err = http.Get("http://localhost:8080/readiness")
	assert.Nil(t, err, "Should be able to connect")
	if resp != nil {
		assert.Equal(t,
			http.StatusOK, resp.StatusCode,
			"Readiness should return 200 when al is ready")
		assert.Equal(t, "OK", body(resp),
			"Incorrect body, 'OK' expected")
	}
	cancel()
}
