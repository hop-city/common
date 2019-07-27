package auth

import (
	"context"
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"
)

var respCode = http.StatusInternalServerError
var blockResponse = 0
var nextBody = "-default-"
var reqCount = 0
var lastQuery url.Values
var lastHeaders http.Header
var ts *httptest.Server

func TestMain(m *testing.M) {
	_ = os.Setenv("AUTH_TIMEOUT", "10")

	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		<-time.After(time.Millisecond * 5)
		reqCount++
		b, _ := ioutil.ReadAll(req.Body)
		query, _ := url.ParseQuery(string(b))
		lastQuery = query
		lastHeaders = req.Header

		if blockResponse != 0 {
			w.WriteHeader(blockResponse)
		} else {
			w.WriteHeader(respCode)
		}

		if nextBody != "-default-" {
			_, _ = w.Write([]byte(nextBody))
		} else {
			_, _ = w.Write([]byte(
				`{"access_token": "` + strconv.Itoa(reqCount) + `"}`,
			))
		}

		// 2+ request will succeed
		respCode = http.StatusOK
	}))
	defer ts.Close()
	retryWait.SetMultiplier(1e-3)
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func shutdown() {
	ts.Close()
}

func clear() (context.Context, func()) {
	<-time.After(time.Millisecond * 10)
	blockResponse = 0
	respCode = http.StatusOK
	reqCount = 0
	nextBody = "-default-"
	lastQuery = nil
	lastHeaders = nil
	authList = make([]*Auth, 0)
	readyCallbacks = make([]func(bool), 0)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	return ctx, cancel
}

func TestWatchGlobalReady(t *testing.T) {
	ctx, cancel := clear()
	defer cancel()
	a, err := ResourceOwner(ctx, ResourceOwnerOptions{
		Url:      "a",
		Username: "koala",
		Password: "pass",
		ClientID: "koID",
		Secret:   "koSec",
	})
	statusChanges := make([]bool, 0)
	WatchGlobalReady(func(status bool) {
		statusChanges = append(statusChanges, status)
	})
	assert.Nil(t, err, "Error should be nil")
	assert.False(t, statusChanges[0],
		"At the beginning ready status should be false")
	a.setValid(true)
	assert.True(t, statusChanges[1],
		"When valid was set to true, ready status should be true")
	a2, _ := ResourceOwner(ctx, ResourceOwnerOptions{
		Url:      "a",
		Username: "koala",
		Password: "pass",
		ClientID: "koID",
		Secret:   "koSec",
	})
	assert.Equal(t, 2, len(authList), "2 Auth objects should be on the list")
	assert.False(t, statusChanges[2],
		"Creating new Auth should revert ready to false")
	a2.setValid(true)
	assert.True(t, statusChanges[3],
		"Setting second auth to true should - now both are in ready state-  should return true")
	// zero list not to impact next tests
}

// tests connection, retry, ready changes, getToken
func TestResourceOwner(t *testing.T) {
	ctx, cancel := clear()
	defer cancel()
	statusChanges := make([]bool, 0)
	WatchGlobalReady(func(status bool) {
		statusChanges = append(statusChanges, status)
	})

	// first request will fail
	respCode = http.StatusInternalServerError
	reqCount = 0

	clientID := "koala-clientID"
	secret := "koala-secret"
	authHeader := "Basic " + base64.StdEncoding.
		EncodeToString(
			[]byte(clientID+":"+secret),
		)
	a, err := ResourceOwner(ctx, ResourceOwnerOptions{
		Url:      ts.URL,
		Username: "koala",
		Password: "pass",
		ClientID: clientID,
		Secret:   secret,
	})
	assert.Nil(t, err, "Should not return error if all options are set")
	// speed up retry process
	retryWait.SetMultiplier(1e-3)

	var token string
	select {
	case token = <-a.GetToken():
		assert.Equal(t, "2", token, "Supplied token doesn't match send token")
	case <-time.After(time.Millisecond * 50):
		assert.Fail(t, "Token not received")
	}

	assert.Equal(t, "koala", lastQuery.Get("username"), "Received incorrect username")
	assert.Equal(t, "pass", lastQuery.Get("password"), "Received incorrect password")
	assert.Equal(t, "password", lastQuery.Get("grant_type"), "Received incorrect grant type")
	assert.Equal(t, authHeader, lastHeaders.Get("authorization"), "Received incorrect header")

	// ready should be true
	assert.True(t, globalReadyStatus(), "We have retrieved the token, so global ready status should be true")
	respCode = http.StatusInternalServerError
	go a.Refresh()
	<-time.After(time.Millisecond)
	assert.False(t, globalReadyStatus(), "We have asked for refresh - global ready status should immediately be false")

	select {
	case token = <-a.GetToken():
		assert.Equal(t, "4", token, "Supplied token doesn't match send token")
		assert.True(t, globalReadyStatus(), "We have retrieved the token, so global ready status should be true")
	case <-time.After(time.Millisecond * 50):
		assert.Fail(t, "Token not received")
	}

	//assert.Fail(t, "-- to see output")
}

func TestClient(t *testing.T) {
	ctx, cancel := clear()
	defer cancel()
	clientID := "koala-clientID"
	secret := "koala-secret"
	//authHeader := "Basic " + base64.StdEncoding.
	//	EncodeToString(
	//		[]byte(clientID+":"+secret),
	//	)
	a, err := ClientCredentials(ctx, ClientCredentialsOptions{
		//a, err := ResourceOwner(ctx, ResourceOwnerOptions{
		Url:      ts.URL,
		ClientID: clientID,
		Secret:   secret,
	})

	assert.Nil(t, err, "Should not return error if all options are set")
	select {
	case token := <-a.GetToken():
		assert.Equal(t, "1", token, "Supplied token doesn't match encoded credentials token")
	case <-time.After(time.Millisecond * 50):
		assert.Fail(t, "Token not received")
	}
	a.Refresh() // should do nothing
	select {
	case token := <-a.GetToken():
		assert.Equal(t, "2", token, "Supplied token doesn't match encoded credentials token")
	case <-time.After(time.Millisecond * 50):
		assert.Fail(t, "Token not received")
	}
}

func TestAuth_AppendHeader(t *testing.T) {
	ctx, cancel := clear()
	defer cancel()
	clientID := "4f56g5y"
	secret := "hc74i5hwy34c"
	//authHeader := "Basic " + base64.StdEncoding.
	//	EncodeToString(
	//		[]byte(clientID+":"+secret),
	//	)
	a, err := ClientCredentials(ctx, ClientCredentialsOptions{
		Url:      ts.URL,
		ClientID: clientID,
		Secret:   secret,
	})
	assert.Nil(t, err, "Should not return error if all options are set")
	header := http.Header{}
	a.AppendHeader(&header)
	assert.Equal(t, "Bearer 1", header.Get("authorization"), "Invalid or no token appended")
}

func TestResourceOwner_WrongUrlError(t *testing.T) {
	ctx, cancel := clear()
	defer cancel()
	statusChanges := make([]bool, 0)
	WatchGlobalReady(func(status bool) {
		statusChanges = append(statusChanges, status)
	})

	// first request will fail
	respCode = http.StatusInternalServerError
	reqCount = 0

	clientID := "koala-clientID"
	secret := "koala-secret"
	//authHeader := "Bearer " + base64.StdEncoding.
	//	EncodeToString(
	//		[]byte(clientID+":"+secret),
	//	)
	a, err := ResourceOwner(ctx, ResourceOwnerOptions{
		Url:      "wrong-url",
		Username: "koala",
		Password: "pass",
		ClientID: clientID,
		Secret:   secret,
	})
	assert.Nil(t, err, "Should not return error if all options are set")
	// speed up retry process
	//t := <-a.GetToken()

	//var token string
	select {
	case token := <-a.GetToken():
		assert.Failf(t, "Token should not be returned", token)
	case <-time.After(time.Millisecond * 50):
	}

}

func TestResourceOwner_500(t *testing.T) {
	ctx, cancel := clear()
	defer cancel()
	statusChanges := make([]bool, 0)
	WatchGlobalReady(func(status bool) {
		statusChanges = append(statusChanges, status)
	})

	// first request will fail
	blockResponse = http.StatusInternalServerError

	clientID := "koala-clientID"
	secret := "koala-secret"

	a, err := ResourceOwner(ctx, ResourceOwnerOptions{
		Url:      ts.URL,
		Username: "koala",
		Password: "pass",
		ClientID: clientID,
		Secret:   secret,
	})
	assert.Nil(t, err, "Should not return error if all options are set")

	select {
	case token := <-a.GetToken():
		assert.Failf(t, "Token should not be returned", token)
	case <-time.After(time.Millisecond * 50):
	}

	assert.Truef(t, reqCount > 2, "Should retry, requests done: %d", reqCount)
}
func TestResourceOwner_RetriesCount(t *testing.T) {
	//<-time.After(time.Millisecond * 100)
	ctx, cancel := clear()
	defer cancel()

	// first request will fail
	blockResponse = http.StatusBadRequest
	maxRetries = 3

	clientID := "koala-clientID"
	secret := "koala-secret"

	a, err := ResourceOwner(ctx, ResourceOwnerOptions{
		Url:      ts.URL,
		Username: "koala",
		Password: "pass",
		ClientID: clientID,
		Secret:   secret,
	})
	assert.Nil(t, err, "Should not return error if all options are set")

	retryWait.SetMultiplier(1e-3)
	select {
	case token := <-a.GetToken():
		assert.Failf(t, "Token should not be returned", token)
	case <-time.After(time.Millisecond * 100):
	}
	assert.Equal(t, a.maxRetries, uint(reqCount-1), "Incorrect number of retries")
	retryWait.SetMultiplier(1e-3)
}

func TestResourceOwner_JSONError(t *testing.T) {
	ctx, cancel := clear()
	defer cancel()

	clientID := "koala-clientID"
	secret := "koala-secret"

	a, err := ResourceOwner(ctx, ResourceOwnerOptions{
		Url:      ts.URL,
		Username: "koala",
		Password: "pass",
		ClientID: clientID,
		Secret:   secret,
	})
	assert.Nil(t, err, "Should not return error if all options are set")

	a.retryMultiplier = 1e-5
	nextBody = `{"":"koala",`
	select {
	case token := <-a.GetToken():
		assert.Failf(t, "Token should not be returned", token)
	case <-time.After(time.Millisecond * 20):
	}
	assert.Truef(t, reqCount > 1, "Should retry on JSON error, got %d requests", reqCount)

	//assert.Fail(t, "-- to see output")
}
