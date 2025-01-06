package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/a-h/jsonapi"
	"github.com/a-h/ragserver/models"
)

func New(baseURL, apiKey string) Client {
	return Client{
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

type Client struct {
	baseURL string
	apiKey  string
}

func (c Client) DocumentsPut(ctx context.Context, req models.DocumentsPostRequest) (resp models.DocumentsPostResponse, err error) {
	url, err := jsonapi.URL(c.baseURL).Path("documents").String()
	if err != nil {
		return resp, err
	}
	return jsonapi.Post[models.DocumentsPostRequest, models.DocumentsPostResponse](ctx, url, req, jsonapi.WithRequestHeader("Authorization", c.apiKey))
}

func (c Client) QueryPost(ctx context.Context, request models.QueryPostRequest, f func(ctx context.Context, chunk []byte) error) (err error) {
	url, err := jsonapi.URL(c.baseURL).Path("query").String()
	if err != nil {
		return err
	}
	buf, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	res, err := jsonapi.Raw(httpReq, jsonapi.WithRequestHeader("Authorization", c.apiKey))
	if err != nil {
		return fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		body, _ := io.ReadAll(res.Body)
		return jsonapi.InvalidStatusError{
			Status: res.StatusCode,
			Body:   string(body),
		}
	}
	for {
		chunk := make([]byte, 1024)
		n, err := res.Body.Read(chunk)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read response body: %w", err)
		}
		if err := f(ctx, chunk[:n]); err != nil {
			return fmt.Errorf("failed to process chunk: %w", err)
		}
	}
	return nil
}
