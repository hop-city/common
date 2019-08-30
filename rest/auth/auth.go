package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/hop-city/common/backoff"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type (
	Auth struct {
		ctx    context.Context
		url    string
		typ    string
		client *http.Client

		tokens *tokenResponse

		m             sync.RWMutex
		tokenRequests []chan string
		valid         *atomic.Value
		fetching      *atomic.Value
		reqCount      uint

		username string
		password string
		clientID string
		secret   string
		scope    string

		retryMultiplier float64
		maxRetries      uint
	}

	tokenResponse struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
		RefreshToken string `json:"refresh_token"`
	}
)

var maxRetries = uint(3)
var authList = make([]*Auth, 0)
var readyCallbacks = make([]func(bool), 0)
var oldReady = false

var retryWait *backoff.Backoff

// TODO(mb) support refresh
// TODO(mb) add other authorisation methods as needed

func init() {
	retryWait = backoff.New().SetJitter(0.3).SetBaseDuration(1000)
}

// creates and initialises base object fields
func authBase() *Auth {
	a := Auth{
		url:             os.Getenv("AUTH_URL"),
		tokenRequests:   make([]chan string, 0),
		valid:           &atomic.Value{},
		fetching:        &atomic.Value{},
		retryMultiplier: 1,
		tokens:          &tokenResponse{},
		maxRetries:      maxRetries,
		reqCount:        0,
	}
	a.valid.Store(false)
	a.fetching.Store(false)
	authList = append(authList, &a)

	// timeout
	to := 5000
	toS := os.Getenv("AUTH_TIMEOUT")
	if len(toS) > 0 {
		t, err := strconv.Atoi(toS)
		if err == nil {
			to = t
		}
	}
	a.client = &http.Client{
		Timeout: time.Millisecond * time.Duration(to),
	}

	notifyReadyChange()
	return &a
}

func (a *Auth) setValid(newState bool) {
	// first set
	current := a.valid.Load().(bool)
	if current != newState {
		a.valid.Store(newState)
		notifyReadyChange()
	}
}
func notifyReadyChange() {
	newReady := globalReadyStatus()
	if oldReady != newReady {
		oldReady = newReady
		for _, callback := range readyCallbacks {
			callback(newReady)
		}
	}
}

func globalReadyStatus() bool {
	globalReady := true
	for _, auth := range authList {
		globalReady = globalReady && auth.valid.Load().(bool)
	}
	return globalReady
}

// Fans out tokens to awaiting parties (subscribed)
// Requests to fetch if token is not valid
func (a *Auth) distributeTokens() {
	if !a.valid.Load().(bool) {
		// TODO(mb) support refresh
		a.fetchToken()
		return
	}
	a.m.Lock()
	defer a.m.Unlock()
	for _, c := range a.tokenRequests {
		c <- a.tokens.AccessToken
	}
	a.tokenRequests = make([]chan string, 0)
}

func (a *Auth) fetchToken() {
	//if a.typ == TypeClientCredentials {
	//	credentials := []byte(a.clientID + ":" + a.secret)
	//	encoded := base64.StdEncoding.EncodeToString(credentials)
	//	authHeader := "Basic " + encoded
	//	a.tokens.AccessToken = authHeader
	//
	//	a.fetching.Store(false)
	//	a.setValid(true)
	//	a.distributeTokens()
	//	return
	//}

	a.reqCount++

	// disallow fetch retry chain start if already in process
	if a.reqCount == 1 && a.fetching.Load().(bool) {
		return
	} else {
		select {
		case <-a.ctx.Done():
			return
		case <-retryWait.Wait(a.ctx, a.reqCount-1):
		}
	}
	a.fetching.Store(true)

	// Query -> buffer
	query := url.Values{}
	if a.scope != "" {
		query.Add("scope", a.scope)
	}
	query.Add("grant_type", a.typ)
	if a.typ == TypeResourceOwner {
		query.Add("username", a.username)
		query.Add("password", a.password)
	}
	body := bytes.NewBuffer([]byte(query.Encode()))

	req, err := http.NewRequest("POST", a.url, body)
	if err != nil {
		return
	}
	// headers

	credentials := []byte(a.clientID + ":" + a.secret)
	encoded := base64.StdEncoding.EncodeToString(credentials)
	authHeader := "Basic " + encoded
	req.Header.Set("authorization", authHeader)
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(req)
	// the definition of madness is to try the same thing
	// multiple times hoping for different result
	if err != nil {
		if a.reqCount <= a.maxRetries {
			a.fetchToken()
			return
		}
		err = errors.Wrapf(
			err,
			"Auth.fetchToken: Connection error, can not authorise user '%s' with method '%s'. Blocked!",
			a.username,
			a.typ,
		)
		log.SetOutput(os.Stderr)
		log.Println(err)
		log.SetOutput(os.Stdout)
		return
	}
	// if 4xx most probably we do something wrong and it doesn't make sense to retry
	// die :( - if readiness is hooked to auth state changes, pod will probably be restarted
	if resp.StatusCode%400 < 100 {
		if a.reqCount <= a.maxRetries {
			a.fetchToken()
			return
		}
		log.SetOutput(os.Stderr)
		log.Printf(
			"Auth.fetchToken: problem with request to authorise user '%s' with method '%s'. Code: %d. Blocked!",
			a.username,
			a.typ,
			resp.StatusCode,
		)
		log.SetOutput(os.Stdout)
		return
	}

	if resp.StatusCode != http.StatusOK {
		a.fetchToken()
		return
	}
	// 200
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = errors.Wrapf(
			err,
			"Auth.fetchToken: error reading body for user '%s' with method '%s'",
			a.username,
			a.typ,
		)
		log.SetOutput(os.Stderr)
		log.Println(err)
		log.SetOutput(os.Stdout)
		a.fetchToken()
		return
	}

	tokens := &tokenResponse{}
	err = json.Unmarshal(respBody, tokens)
	if err != nil {
		err = errors.Wrapf(
			err,
			"Auth.fetchToken: error unmarshalling body for user '%s' with method '%s'",
			a.username,
			a.typ,
		)
		log.SetOutput(os.Stderr)
		log.Println(err)
		log.SetOutput(os.Stdout)
		a.fetchToken()
		return
	}

	a.tokens = tokens
	a.reqCount = 0
	a.fetching.Store(false)
	a.setValid(true)
	a.distributeTokens()
}

//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
