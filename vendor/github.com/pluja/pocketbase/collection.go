package pocketbase

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type Collection[T any] struct {
	*Client
	Name               string
	BaseCollectionPath string
}

func CollectionSet[T any](client *Client, collection string) *Collection[T] {
	return &Collection[T]{
		Client:             client,
		Name:               collection,
		BaseCollectionPath: client.url + "/api/collections/" + url.QueryEscape(collection),
	}
}

func (c *Collection[T]) Update(id string, body T) error {
	return c.Client.Update(c.Name, id, body)
}

func (c *Collection[T]) Create(body T) (ResponseCreate, error) {
	return c.Client.Create(c.Name, body)
}

func (c *Collection[T]) Delete(id string) error {
	return c.Client.Delete(c.Name, id)
}

func (c *Collection[T]) List(params ParamsList) (ResponseList[T], error) {
	var response ResponseList[T]
	params.hackResponseRef = &response

	_, err := c.Client.List(c.Name, params)
	return response, err
}

func (c *Collection[T]) FullList(params ParamsList) (ResponseList[T], error) {
	var response ResponseList[T]
	params.hackResponseRef = &response

	_, err := c.Client.FullList(c.Name, params)
	return response, err
}

func (c *Collection[T]) One(id string) (T, error) {
	var response T

	if err := c.Authorize(); err != nil {
		return response, err
	}

	request := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetPathParam("collection", c.Name).
		SetPathParam("id", id)

	resp, err := request.Get(c.url + "/api/collections/{collection}/records/{id}")
	if err != nil {
		return response, fmt.Errorf("[one] can't send update request to pocketbase, err %w", err)
	}

	if resp.IsError() {
		return response, fmt.Errorf("[one] pocketbase returned status: %d, msg: %s, err %w",
			resp.StatusCode(),
			resp.String(),
			ErrInvalidResponse,
		)
	}

	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return response, fmt.Errorf("[one] can't unmarshal response, err %w", err)
	}
	return response, nil
}

// Get one record with params (only fields and expand supported)
func (c *Collection[T]) OneWithParams(id string, params ParamsList) (T, error) {
	var response T

	if err := c.Authorize(); err != nil {
		return response, err
	}

	request := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetPathParam("collection", c.Name).
		SetPathParam("id", id).
		SetQueryParam("fields", params.Fields).
		SetQueryParam("expand", params.Expand)

	resp, err := request.Get(c.url + "/api/collections/{collection}/records/{id}")
	if err != nil {
		return response, fmt.Errorf("[one] can't send update request to pocketbase, err %w", err)
	}

	if resp.IsError() {
		return response, fmt.Errorf("[one] pocketbase returned status: %d, msg: %s, err %w",
			resp.StatusCode(),
			resp.String(),
			ErrInvalidResponse,
		)
	}

	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return response, fmt.Errorf("[one] can't unmarshal response, err %w", err)
	}
	return response, nil
}
