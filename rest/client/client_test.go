package client

import (
	"context"
	"github.com/hop-city/common/logger"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

type Server struct {
	Ts              *httptest.Server
	LastReq         *http.Request
	LastBody        string
	LastCType       string
	NextStatus      int
	NextBody        []byte
	NextContentType string
	ReqCount        int
}

type AuthMock struct {
	token string
}

func (a *AuthMock) GetToken() <-chan string {
	ch := make(chan string, 1)
	ch <- a.token
	return ch
}
func (a *AuthMock) AppendHeader(h *http.Header) {
	h.Set("authorization", a.token)
}
func (a *AuthMock) Refresh() {
	a.token = a.token + "+"
}

func setup() (context.Context, func(), *Server) {
	_ = os.Setenv("LOG_LEVEL", "debug")
	ctx := context.Background()
	l := logger.New()
	ctx = l.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	s := server(ctx)
	return ctx, cancel, s
}
func server(ctx context.Context) *Server {
	s := &Server{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//log.Println("Server", s)
		s.ReqCount++
		s.LastReq = r
		s.LastCType = r.Header.Get("content-type")
		b, _ := ioutil.ReadAll(r.Body)
		s.LastBody = string(b)

		if s.NextContentType != "" {
			w.Header().Set("Content-Type", s.NextContentType)
		} else {
			w.Header().Set("content-type", "text/plain")
		}

		// can not set headers after WriteHeader!
		if s.NextStatus != 0 {
			w.WriteHeader(s.NextStatus)
			s.NextStatus = 0
			return
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_, _ = w.Write(s.NextBody)
	}))
	s.Ts = ts
	go func() {
		select {
		case <-ctx.Done():
			ts.Close()
		}
	}()
	return s
}
func read(resp *http.Response) []byte {
	b, _ := ioutil.ReadAll(resp.Body)
	return b
}

// ---------------- tests

func TestNew(t *testing.T) {
	ctx, cancel, _ := setup()
	defer cancel()
	client := New(ctx, nil)
	assert.Equal(t, uint(0), client.maxRetries,
		"Incorrect default number of max retries %d", client.maxRetries)
	assert.Equal(t, time.Duration(30*time.Second), client.httpClient.Timeout,
		"Incorrect default timeout - %d", client.httpClient.Timeout)

	client.SetMaxRetries(11)
	assert.Equal(t, uint(11), client.maxRetries,
		"Incorrect default number of max retries %d", client.maxRetries)

	client.SetTimeout(123)
	assert.Equal(t, time.Duration(123*time.Second), client.httpClient.Timeout,
		"Incorrect default timeout - %d", client.httpClient.Timeout)
}

func TestClient_Fetch_GET(t *testing.T) {
	ctx, cancel, s := setup()
	defer cancel()
	client := New(ctx, nil).FavourContentHeaders(true).SetMaxRetries(4)
	s.NextBody = []byte(`{"Name":"koala"}`)
	resp, err := client.Fetch(FetchOptions{
		Method: "GET",
		Url:    s.Ts.URL,
	})
	if err != nil {
		assert.Nil(t, err, "Should connect successfully")
		return
	}
	assert.Equal(t, s.NextBody, read(resp), "Returned body doesn't match")

	s.NextContentType = "application/json"
	user := struct{ Name string }{}
	resp, err = client.Fetch(FetchOptions{
		Method: "GET",
		Url:    s.Ts.URL,
		Expect: &user,
	})
	if err != nil {
		assert.Nil(t, err, "Should connect successfully")
		return
	}
	assert.Equal(t, "koala", user.Name, "Data was not returned, or not parsed properly - %s", resp)

	s.NextContentType = "application/x-www-form-urlencoded"
	s.NextBody = []byte(`name=koala&surname=theFirst`)
	s.NextStatus = http.StatusGatewayTimeout
	user2 := make(map[string]string)
	resp, err = client.Fetch(FetchOptions{
		Method: "GET",
		Url:    s.Ts.URL,
		Expect: user2,
	})
	if err != nil {
		assert.Nil(t, err, "Should connect successfully")
		return
	}

	surname, found := user2["surname"]
	assert.True(t, found, "surname field should be found in the response")
	assert.Equal(t, "theFirst", surname, "Data was not returned, or not parsed properly - %s", resp)

	user2 = make(map[string]string)
	resp, err = client.Fetch(FetchOptions{
		Method: "GET",
		Url:    s.Ts.URL,
		Expect: &user2, // pointer should return an error
	})
	assert.NotNil(t, err, "Should return error - map should not be passed as  pointer")
	assert.Equal(t, s.NextBody, read(resp), "Data was not returned, or not parsed properly - %s", resp)
}
func TestClient_Fetch_UnamrshallingGuess(t *testing.T) {
	ctx, cancel, s := setup()
	defer cancel()
	au := &AuthMock{token: "token"}
	client := New(ctx, au).FavourContentHeaders(true)

	// string
	s.NextContentType = "invalid"
	s.NextBody = []byte("something")
	b, err := client.Fetch(FetchOptions{
		Method: "GET",
		Url:    s.Ts.URL,
		Expect: nil,
	})
	assert.Nil(t, err, "There should be no error")
	assert.Equal(t, s.NextBody, read(b), "Incorrect body received")

	// json
	s.NextContentType = "invalid"
	s.NextBody = []byte(`{"Name":"koala"}`)
	out := struct{ Name string }{}
	_, err = client.Fetch(FetchOptions{
		Method: "GET",
		Url:    s.Ts.URL,
		Expect: &out,
	})
	assert.Nil(t, err, "There should be no error")
	assert.Equal(t, "koala", out.Name, "Incorrect body received")

	// json
	s.NextContentType = "invalid"
	s.NextBody = []byte("Name=koala")
	out2 := make(map[string]string)
	_, err = client.Fetch(FetchOptions{
		Method: "GET",
		Url:    s.Ts.URL,
		Expect: out2,
	})
	assert.Nil(t, err, "There should be no error")
	name, _ := out2["Name"]
	assert.Equal(t, "koala", name, "Incorrect body received")

	s.NextContentType = "text/plain"
	s.NextBody = []byte("koala12")
	str := "empty"
	resp, err := client.Fetch(FetchOptions{
		Method: "GET",
		Url:    s.Ts.URL,
		Expect: &str,
	})
	if err != nil {
		assert.Nil(t, err, "Should connect successfully")
		return
	}
	assert.Equal(t, "koala12", string(read(resp)), "Data was not returned, or not parsed properly - %s", resp)

	s.NextContentType = "application/json"
	s.NextBody = []byte(`{"name":"koala12"}`)
	expB := make([]byte, 0)
	resp, err = client.Fetch(FetchOptions{
		Method: "GET",
		Url:    s.Ts.URL,
		Expect: &expB,
	})
	if err != nil {
		assert.Nil(t, err, "Should connect successfully")
		return
	}
	assert.Equal(t, s.NextBody, expB, "Data was not returned, or not parsed properly - %s", resp)
}
func TestClient_Fetch_Auth(t *testing.T) {
	ctx, cancel, s := setup()
	defer cancel()
	au := &AuthMock{token: "token"}
	client := New(ctx, au)

	// check header existance
	_, _ = client.Fetch(FetchOptions{
		Method: "GET",
		Url:    s.Ts.URL,
	})
	assert.NotNil(t, "token", s.LastReq.Header.Get("authorization"), "Authorization token didn't reach server")

	// check 401 - reauthorisation
	s.NextStatus = http.StatusUnauthorized
	s.ReqCount = 0
	_, _ = client.Fetch(FetchOptions{
		Method: "GET",
		Url:    s.Ts.URL,
	})
	// should block untill retry is finished
	// AuthMock.Refresh should update token to "token+"
	assert.NotNil(t, "token+", s.LastReq.Header.Get("authorization"),
		"Authorization token should be updated after 401")
	assert.NotNil(t, 2, s.ReqCount,
		"There should be 2 requests, 401 and 200")
}

func TestClient_Fetch_Sending(t *testing.T) {
	ctx, cancel, s := setup()
	defer cancel()
	au := &AuthMock{token: "token"}
	client := New(ctx, au)

	str := "koala"
	_, _ = client.Fetch(FetchOptions{
		Method: "POST",
		Url:    s.Ts.URL,
		Send:   str,
	})
	assert.Equal(t, str, s.LastBody,
		"Body not received by server")
	assert.Equal(t, "text/plain", s.LastCType,
		"Invalid content type received by server")

	data := struct{ Name string }{Name: "koala"}
	_, _ = client.Fetch(FetchOptions{
		Method: "POST",
		Url:    s.Ts.URL,
		Send:   data,
	})
	assert.Equal(t, `{"Name":"koala"}`, s.LastBody,
		"Body not received by server")
	assert.Equal(t, "application/json", s.LastCType,
		"Invalid content type received by server")

	urlEnc := map[string]string{"Name": "koala"}
	_, _ = client.Fetch(FetchOptions{
		Method: "POST",
		Url:    s.Ts.URL,
		Send:   urlEnc,
	})
	assert.Equal(t, "Name=koala", s.LastBody,
		"Body not received by server")
	assert.Equal(t, "application/x-www-form-urlencoded", s.LastCType,
		"Invalid content type received by server")

	urlEnc = map[string]string{"Name": "koala"}
	_, _ = client.Fetch(FetchOptions{
		Method:  "POST",
		Url:     s.Ts.URL,
		Send:    urlEnc,
		Headers: map[string]string{"content-type": "koala"},
	})
	assert.Equal(t, "Name=koala", s.LastBody,
		"Body not received by server")
	assert.Equal(t, "koala", s.LastCType,
		"Header should be overridden")
}
