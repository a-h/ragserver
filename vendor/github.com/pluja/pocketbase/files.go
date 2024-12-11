package pocketbase

import (
	"encoding/json"
	"fmt"
)

type (
	Files struct {
		*Client
	}

	ResponseGetToken struct {
		Token string `json:"token"`
	}
)

// GetToken requests a new private file access token for the current auth model (admin or record).
func (f Files) GetToken() (string, error) {
	if err := f.Authorize(); err != nil {
		return "", err
	}

	request := f.client.R().
		SetHeader("Content-Type", "application/json")

	resp, err := request.Post(f.url + "/api/files/token")
	if err != nil {
		return "", fmt.Errorf("[files] can't send token request to pocketbase, err %w", err)
	}

	if resp.IsError() {
		return "", fmt.Errorf("[files] pocketbase returned status at getting a new token: %d, msg: %s, err %w",
			resp.StatusCode(),
			resp.String(),
			ErrInvalidResponse,
		)
	}

	response := ResponseGetToken{}
	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return "", fmt.Errorf("[files] can't unmarshal response, err %w", err)
	}
	return response.Token, nil
}
