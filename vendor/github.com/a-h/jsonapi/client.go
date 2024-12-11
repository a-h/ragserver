package jsonapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Config struct {
	Client     Doer
	Middleware []Middleware
}

type Middleware interface {
	Request(req *http.Request) error
	Response(res *http.Response) error
}

// WithTimeout attempts to set the timeout on the HTTP client.
// It is a no-op if the underlying Doer is not an *http.Client.
func WithTimeout(timeout time.Duration) Opt {
	return func(c *Config) error {
		if c.Client == nil {
			c.Client = http.DefaultClient
		}
		if httpc, ok := c.Client.(*http.Client); ok {
			httpc.Timeout = timeout
		}
		return nil
	}
}

// WithClient uses a custom Doer for the HTTP requests.
// Typically, this is a *http.Client.
func WithClient(client Doer) Opt {
	return func(c *Config) error {
		c.Client = client
		return nil
	}
}

// WithMiddleware adds middleware to the HTTP request.
// See the github.com/a-h/jsonapi/middleware package for middleware.
func WithMiddleware(middleware ...Middleware) Opt {
	return func(c *Config) error {
		c.Middleware = append(c.Middleware, middleware...)
		return nil
	}
}

// Opt is an option for the JSON API client.
// See WithTimeout, WithClient, and WithMiddleware.
type Opt func(*Config) (err error)

func newConfig(opts ...Opt) (*Config, error) {
	c := &Config{
		Client: http.DefaultClient,
		Middleware: []Middleware{
			&requestHeaderMiddleware{"Content-Type", "application/json"},
		},
	}
	for _, o := range opts {
		if err := o(c); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}
	return c, nil
}

// Put a HTTP request to the given URL with the given request body.
func Put[TReq, TResp any](ctx context.Context, url string, request TReq, opts ...Opt) (response TResp, err error) {
	return doRequestResponse[TReq, TResp](ctx, http.MethodPut, url, request, opts...)
}

// Post a HTTP request to the given URL with the given request body.
func Post[TReq, TResp any](ctx context.Context, url string, request TReq, opts ...Opt) (response TResp, err error) {
	return doRequestResponse[TReq, TResp](ctx, http.MethodPost, url, request, opts...)
}

func Raw(req *http.Request, opts ...Opt) (res *http.Response, err error) {
	config, err := newConfig(opts...)
	if err != nil {
		return res, fmt.Errorf("failed to create config: %w", err)
	}
	for _, m := range config.Middleware {
		if err := m.Request(req); err != nil {
			return res, fmt.Errorf("middleware failed to modify request: %w", err)
		}
	}
	res, err = config.Client.Do(req)
	if err != nil {
		return res, fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	for _, m := range config.Middleware {
		if err := m.Response(res); err != nil {
			return res, fmt.Errorf("middleware failed to modify response: %w", err)
		}
	}
	return res, nil
}

func doRequestResponse[TReq, TResp any](ctx context.Context, method, url string, request TReq, opts ...Opt) (response TResp, err error) {
	buf, err := json.Marshal(request)
	if err != nil {
		return response, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(buf))
	if err != nil {
		return response, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := Raw(req, opts...)
	if err != nil {
		return response, err
	}
	return decodeResponse[TResp](resp)
}

// Get a HTTP response from the given URL.
// Returns ok=false if the response was a 404.
func Get[TResp any](ctx context.Context, url string, opts ...Opt) (response TResp, ok bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return response, false, fmt.Errorf("failed to create request: %w", err)
	}
	res, err := Raw(req, opts...)
	if err != nil {
		return response, false, err
	}
	if res.StatusCode == http.StatusNotFound {
		return response, false, nil
	}
	response, err = decodeResponse[TResp](res)
	if err != nil {
		return response, false, err
	}
	return response, true, err
}

func decodeResponse[TResp any](res *http.Response) (response TResp, err error) {
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		body, _ := io.ReadAll(res.Body)
		return response, InvalidStatusError{
			Status: res.StatusCode,
			Body:   string(body),
		}
	}
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %w", err)
	}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return response, InvalidJSONError{
			Status: res.StatusCode,
			Body:   string(bodyBytes),
			Err:    err,
		}
	}
	return response, nil
}

type InvalidStatusError struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

func (e InvalidStatusError) Error() string {
	return fmt.Sprintf("api responded with non-success status %d: message: %s", e.Status, e.Body)
}

type InvalidJSONError struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
	Err    error  `json:"error"`
}

func (e InvalidJSONError) Error() string {
	return fmt.Sprintf("api responded with 2xx status code %d, but the response could not be decoded with error: %v: %q", e.Status, e.Err, e.Body)
}
