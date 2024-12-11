package jsonapi

import (
	"net/http"
)

func WithRequestHeader(key, value string) Opt {
	return func(c *Config) error {
		c.Middleware = append(c.Middleware, &requestHeaderMiddleware{key: key, value: value})
		return nil
	}
}

type requestHeaderMiddleware struct {
	key   string
	value string
}

func (m *requestHeaderMiddleware) Request(req *http.Request) error {
	req.Header.Set(m.key, m.value)
	return nil
}

func (m *requestHeaderMiddleware) Response(res *http.Response) error {
	return nil
}

func WithAuthorization(authorization string) Opt {
	return WithRequestHeader("Authorization", authorization)
}

func WithContentType(contentType string) Opt {
	return WithRequestHeader("Content-Type", contentType)
}
