package client

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/hop-city/common/wait"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type (
	Client struct {
		ctx                  context.Context
		log                  *zerolog.Logger
		httpClient           *http.Client
		auth                 Auth
		maxRetries           uint
		backoff              *wait.Wait
		closeConnection      bool
		favourContentHeaders bool
		retryStatusCodes     []int
	}

	Auth interface {
		GetToken() <-chan string
		Refresh()
		AppendHeader(*http.Header)
	}

	FetchOptions struct {
		Ctx     context.Context
		Method  string
		Url     string
		Send    interface{}
		Expect  interface{}
		Headers map[string]string
		no      int
	}
)

var requestCount = 0

// for now let's try
// go-resty/resty
// github.com/go-resty/resty/v2 v2.0.0

// TODO(mb) add headers forwarding from context

// TODO(mb) retrying adds each new retry to stack because of direct return
//  Shall we make it flat?

// TODO(mb) getting header is potentially forever blocking
//  How to best approach this?
func New(ctx context.Context, auth Auth) *Client {
	hc := &http.Client{
		Timeout: 30 * time.Second,
	}

	backoff := wait.New().SetJitter(0.3).SetMultiplier(1)
	c := &Client{
		ctx: ctx,
		log: zerolog.Ctx(ctx),

		httpClient:           hc,
		auth:                 auth,
		maxRetries:           0,
		backoff:              backoff,
		closeConnection:      false,
		favourContentHeaders: false,
		// https://httpstatuses.com
		retryStatusCodes: []int{408, 500, 503, 504},
	}

	return c
}
func (c *Client) SetMaxRetries(retries uint) *Client {
	c.maxRetries = retries
	return c
}
func (c *Client) SetTimeout(seconds time.Duration) *Client {
	c.httpClient.Timeout = seconds * time.Second
	return c
}
func (c *Client) SetCloseConnection(shouldClose bool) *Client {
	c.closeConnection = shouldClose
	return c
}
func (c *Client) FavourContentHeaders(do bool) *Client {
	c.favourContentHeaders = do
	return c
}

func (c *Client) Fetch(opt FetchOptions) (*http.Response, error) {
	requestCount++
	opt.no = requestCount
	return c.fetch(opt, nil, nil, 0)
}

func (c *Client) fetch(opt FetchOptions, lastResponse *http.Response, lastError error, retry uint) (*http.Response, error) {
	var log *zerolog.Logger
	if opt.Ctx != nil {
		log = zerolog.Ctx(opt.Ctx)
	} else {
		log = c.log
	}

	if retry > c.maxRetries {
		// no more retries
		if lastResponse != nil {
			log.Debug().Msgf(
				"[%d/%d] rest/client.Fetch: %d No more retries. skipping %s",
				opt.no,
				retry,
				lastResponse.StatusCode,
				opt.Url,
			)
		} else {
			log.Debug().Msgf(
				"[%d/%d] rest/client.Fetch: No more retries. No response. skipping %s",
				opt.no,
				retry,
				opt.Url,
			)
		}
		return lastResponse, lastError
	} else if retry > 0 && lastResponse != nil {
		// still got retries, shall continue?
		if !c.shouldRetry(lastResponse) {
			log.Debug().Msgf(
				"[%d/%d] rest/client.Fetch: Code %d will not be retried, skipping %s",
				opt.no,
				retry,
				lastResponse.StatusCode,
				opt.Url,
			)
			return lastResponse, lastError
		}
		log.Debug().Msgf(
			"[%d/%d] rest/client.Fetch: %d Retrying: %s",
			opt.no,
			retry,
			lastResponse.StatusCode,
			opt.Url,
		)
	}
	<-c.backoff.Backoff(c.ctx, retry)

	bodyReader := readPayload(opt.Send)

	req, err := http.NewRequest(opt.Method, opt.Url, bodyReader)
	if err != nil {
		return nil, errors.Wrapf(err, "[%d] rest/client.Fetch: error creating request:", retry)
	}
	if c.closeConnection {
		req.Close = true
	}
	c.setHeaders(req, opt)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		err = errors.Wrapf(err, "[%d] rest/client.Fetch: error sending request:", retry)
		return c.fetch(opt, nil, err, retry+1)
	}

	data, err := ioutil.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(data))

	d := string(data)
	if len(d) > 100 {
		d = d[0:100] + "..."
	}
	log.Debug().Msgf(
		"[%d/%d] rest/client.Fetch: %d %s %s %s",
		opt.no,
		retry,
		resp.StatusCode,
		resp.Header.Get("content-type"),
		opt.Url,
		d,
	)
	if resp.StatusCode == 401 {
		if c.auth != nil {
			c.auth.Refresh()
			return c.fetch(opt, resp, err, retry+1)
		}
		return resp, errors.Errorf("[%d] rest/client.Fetch: unauthorised - code %d - %s", retry, resp.StatusCode, data)
	}
	if resp.StatusCode >= 400 {
		err = errors.Errorf("[%d] rest/client.Fetch: Error %d %s %s", retry, resp.StatusCode, opt.Url, data)
		return c.fetch(opt, resp, err, retry+1)
	}

	// resolve body type based on content-type and opt.Expected
	return c.readBody(&opt, resp, data)
}

func readPayload(payload interface{}) io.Reader {
	var bodyReader io.Reader
	switch payload.(type) {
	case nil:
		bodyReader = nil
	case map[string]string:
		data, _ := payload.(map[string]string)
		query := url.Values{}
		for k, v := range data {
			query.Set(k, v)
		}
		bodyReader = bytes.NewReader([]byte(query.Encode()))
	case []byte:
		b, _ := payload.([]byte)
		bodyReader = bytes.NewReader(b)
	case string:
		str, _ := payload.(string)
		bodyReader = bytes.NewReader([]byte(str))
	default:
		data, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(data)
	}
	return bodyReader
}

func (c *Client) setHeaders(req *http.Request, opt FetchOptions) {
	// Auth
	if c.auth != nil {
		// blocking
		c.auth.AppendHeader(&req.Header)
	}

	// User headers
	if opt.Headers != nil {
		for k, v := range opt.Headers {
			req.Header.Set(k, v)
		}
	}

	// Data type
	if req.Header.Get("content-type") != "" {
		return
	}
	switch opt.Send.(type) {
	case map[string]string:
		req.Header.Set("content-type", "application/x-www-form-urlencoded")
	case string:
		req.Header.Set("content-type", "text/plain")
	default: // []byte too
		req.Header.Set("content-type", "application/json")
	}

}

func (c *Client) readBody(opt *FetchOptions, resp *http.Response, data []byte) (*http.Response, error) {
	if c.favourContentHeaders {
		switch resp.Header.Get("content-type") {
		case "text/plain":
			if out, ok := opt.Expect.(*string); ok {
				*out = string(data)
			}
			return resp, nil

		case "application/x-www-form-urlencoded":
			if out, ok := opt.Expect.(map[string]string); ok {
				params, err := url.ParseQuery(string(data))
				if err != nil {
					return resp, errors.Errorf("rest/client.Fetch: error parsing urlencoded data - '%s'", data)
				}
				for k, v := range params {
					out[k] = strings.Join(v, ",")
				}
			} else {
				if opt.Expect != nil {
					return resp, errors.Errorf(
						"rest/client.Fetch: received urlencoded data but expected data is of type %T", opt.Expect)
				}
			}
			return resp, nil

		case "application/json":
			if opt.Expect != nil {
				err := json.Unmarshal(data, opt.Expect)
				if err != nil {
					return resp, errors.Errorf("rest/client.Fetch: error unmarshalling JSON data - '%s'", data)
				}
			}
			return resp, nil
		}
	}
	// no content type header provided, guessing based on user input
	if opt.Expect == nil {
		return resp, nil
	}
	if out, ok := opt.Expect.(*string); ok {
		*out = string(data)
	} else if out, ok := opt.Expect.(map[string]string); ok {
		params, err := url.ParseQuery(string(data))
		if err != nil {
			return resp, errors.Errorf("rest/client.Fetch: error parsing urlencoded data - '%s'", data)
		}
		for k, v := range params {
			out[k] = strings.Join(v, ",")
		}
	} else {
		err := json.Unmarshal(data, opt.Expect)
		if err != nil {
			return resp, errors.Errorf("rest/client.Fetch: error unmarshalling JSON data - '%s'", data)
		}
	}
	return resp, nil
}

func (c *Client) shouldRetry(resp *http.Response) bool {
	hit := false
	current := resp.StatusCode
	for _, v := range c.retryStatusCodes {
		if v == current {
			hit = true
			break
		}
	}
	return hit
}
