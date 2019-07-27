package auth

import (
	"context"
	"github.com/pkg/errors"
	"net/http"
)

type (
	ClientCredentialsOptions struct {
		ClientID string `env:"AUTH_CLIENT_ID" long:"auth-client-id"`
		Secret   string `env:"AUTH_CLIENT_SECRET" long:"auth-client-secret"`
		Scope    string `env:"AUTH_CLIENT_SCOPE" long:"auth-client-scope"`

		Url string `env:"AUTH_URL" long:"auth-url"`
	}
	ResourceOwnerOptions struct {
		ClientID string `env:"AUTH_CLIENT_ID" long:"auth-client-id"`
		Secret   string `env:"AUTH_CLIENT_SECRET" long:"auth-client-secret"`
		Scope    string `env:"AUTH_CLIENT_SCOPE" long:"auth-client-scope"`

		Username string `env:"AUTH_USERNAME" long:"auth-username"`
		Password string `env:"AUTH_PASSWORD" long:"auth-password"`

		Url string `env:"AUTH_URL" long:"auth-url"`
	}
)

const TypeResourceOwner = "password"
const TypeClientCredentials = "client_credentials"

func WatchGlobalReady(callback func(bool)) {
	readyCallbacks = append(readyCallbacks, callback)
	callback(globalReadyStatus())
}

// Resource owner grant type
func ResourceOwner(ctx context.Context, options ResourceOwnerOptions) (*Auth, error) {
	a := authBase()
	a.typ = TypeResourceOwner
	if options.Url != "" {
		a.url = options.Url
	}

	a.ctx = ctx

	a.clientID = options.ClientID
	a.secret = options.Secret
	a.scope = options.Scope

	a.username = options.Username
	a.password = options.Password

	if a.url == "" {
		return nil, errors.New("Auth.ResourceOwner: Missing authorization url")
	}
	if a.secret == "" {
		return nil, errors.New("Auth.ResourceOwner: Missing client secret")
	}
	if a.clientID == "" {
		return nil, errors.New("Auth.ResourceOwner: Missing client ID")
	}
	if a.username == "" {
		return nil, errors.New("Auth.ResourceOwner: Missing username")
	}
	if a.password == "" {
		return nil, errors.New("Auth.ResourceOwner: Missing password")
	}

	return a, nil
}

func ClientCredentials(ctx context.Context, options ClientCredentialsOptions) (*Auth, error) {
	a := authBase()
	a.typ = TypeClientCredentials
	if options.Url != "" {
		a.url = options.Url
	}

	a.ctx = ctx
	a.clientID = options.ClientID
	a.secret = options.Secret
	a.scope = options.Scope

	if a.url == "" {
		return nil, errors.New("Auth.ClientCredentials: Missing authorization url")
	}
	if a.secret == "" {
		return nil, errors.New("Auth.ClientCredentials: Missing client secret")
	}
	if a.clientID == "" {
		return nil, errors.New("Auth.ClientCredentials: Missing client ID")
	}
	if a.clientID == "" {
		return nil, errors.New("Auth.ClientCredentials: Missing requested scope")
	}

	return a, nil
}

// Interaction
func (a *Auth) GetToken() <-chan string {
	ch := make(chan string, 1)
	if a.valid.Load().(bool) {
		a.m.RLock()
		ch <- a.tokens.AccessToken
		a.m.RUnlock()
		return ch
	}
	a.m.Lock()
	a.tokenRequests = append(a.tokenRequests, ch)
	a.m.Unlock()
	go a.distributeTokens()
	return ch
}

func (a *Auth) AppendHeader(h *http.Header) {
	token := <-a.GetToken()
	h.Set("authorization", "Bearer "+token)
}

func (a *Auth) Refresh() {
	a.reqCount = 0
	a.setValid(false)
	a.fetchToken()
}
