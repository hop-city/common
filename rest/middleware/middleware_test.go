package middleware

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPing(t *testing.T) {
	lastHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not for your eyes"))
		})

	server := httptest.NewServer(
		Ping(lastHandler),
	)
	res, _ := http.Get(server.URL + "/ping")
	b, _ := ioutil.ReadAll(res.Body)
	str := string(b)
	assert.Equalf(t, "pong", str,
		"Invalid response, should see pong, received %s", str)
	server.Close()
}
func TestPing_isTransparent(t *testing.T) {
	resString := "not for your eyes"
	lastHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(resString))
		})

	server := httptest.NewServer(
		Ping(lastHandler),
	)
	res, _ := http.Get(server.URL)
	b, _ := ioutil.ReadAll(res.Body)
	str := string(b)
	assert.Equalf(t, resString, str,
		"Invalid response, should see '%s', received %s", resString, str)
	server.Close()
}
