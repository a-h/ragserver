package jsonapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// WithAuthMiddleware returns an Opt that adds authentication middleware to the client.
// The tokenFetcher should return the access token to be used in the Authorization header.
func WithAuthMiddleware(tokenFetcher func() (string, error)) Opt {
	return func(c *Config) error {
		c.Middleware = append(c.Middleware, newAuthMiddleware(tokenFetcher))
		return nil
	}
}

func newAuthMiddleware(tokenFetcher func() (string, error)) *AuthMiddleware {
	return &AuthMiddleware{
		TokenFetcher: tokenFetcher,
		MinRemaining: time.Minute * 10,
		now:          time.Now,
		m:            &sync.Mutex{},
	}
}

type AuthMiddleware struct {
	// TokenFetcher is a function that returns a new access token.
	// The access token should be fetched from the authentication server, and is expected
	// to be a base64 encoded JWT, i.e. a base64 encoded string of the form
	// "header.payload.signature".
	TokenFetcher func() (string, error)
	MinRemaining time.Duration
	token        string
	expires      time.Time
	now          func() time.Time
	m            *sync.Mutex
}

func (m *AuthMiddleware) Request(req *http.Request) (err error) {
	if m.TokenFetcher == nil {
		return nil
	}
	m.m.Lock()
	defer m.m.Unlock()
	if m.token == "" || m.expires.IsZero() || m.expires.Before(m.now().Add(-m.MinRemaining)) {
		m.token, err = m.TokenFetcher()
		if err != nil {
			m.token = ""
			return fmt.Errorf("failed to fetch token: %w", err)
		}
		if strings.HasPrefix(m.token, "Bearer ") {
			m.token = strings.TrimPrefix(m.token, "Bearer ")
		}
		m.expires, err = getExpiry(m.token)
		if err != nil {
			m.expires = time.Time{}
			return fmt.Errorf("failed to get expiry: %w", err)
		}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.token))
	return nil
}

func (m *AuthMiddleware) Response(res *http.Response) error {
	return nil
}

type jwtClaims struct {
	Exp int `json:"exp"`
}

func getExpiry(accessToken string) (expires time.Time, err error) {
	base64Claims := strings.Split(accessToken, ".")
	if len(base64Claims) != 3 {
		return expires, fmt.Errorf("unexpected token format")
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(base64Claims[1])
	if err != nil {
		return expires, fmt.Errorf("failed to decode claims: %w", err)
	}
	var claims jwtClaims
	err = json.Unmarshal(claimsBytes, &claims)
	if err != nil {
		return expires, fmt.Errorf("failed to unmarshal claims: %w", err)
	}
	return time.Unix(int64(claims.Exp), 0), nil
}
