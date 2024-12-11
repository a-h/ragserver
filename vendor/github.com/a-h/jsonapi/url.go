package jsonapi

import (
	"errors"
	"net/url"
)

type URLBuilder struct {
	u   *url.URL
	err error
}

var ErrEmptyURL = errors.New("empty url")
var ErrMissingScheme = errors.New("missing scheme")

func URL(baseURL string) *URLBuilder {
	if baseURL == "" {
		return &URLBuilder{err: ErrEmptyURL}
	}
	ub := &URLBuilder{}
	ub.u, ub.err = url.Parse(baseURL)
	if ub.err == nil && ub.u.Scheme == "" {
		ub.err = ErrMissingScheme
	}
	return ub
}

func (ub *URLBuilder) Path(segments ...string) *URLBuilder {
	if ub.err != nil {
		return ub
	}
	for _, s := range segments {
		ub.u.Path += "/" + s
	}
	return ub
}

func (ub *URLBuilder) Query(query map[string]string) *URLBuilder {
	if ub.err != nil {
		return ub
	}
	q := ub.u.Query()
	for k, v := range query {
		q.Add(k, v)
	}
	ub.u.RawQuery = q.Encode()
	return ub
}

func (ub *URLBuilder) Fragment(fragment string) *URLBuilder {
	if ub.err != nil {
		return ub
	}
	ub.u.Fragment = fragment
	return ub
}

func (ub *URLBuilder) String() (string, error) {
	if ub.err != nil {
		return "", ub.err
	}
	return ub.u.String(), nil
}
