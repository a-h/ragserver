package pocketbase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
)

type (
	Backup struct {
		*Client
	}

	ResponseBackupFullList struct {
		Key      string `json:"key"`
		Size     int    `json:"size"`
		Modified string `json:"modified"`
	}
)

// FullList returns list with all available backup files.
func (b Backup) FullList() ([]ResponseBackupFullList, error) {
	var response []ResponseBackupFullList
	if err := b.Authorize(); err != nil {
		return response, err
	}

	request := b.client.R().
		SetHeader("Content-Type", "application/json")

	resp, err := request.Get(b.url + "/api/backups")
	if err != nil {
		return response, fmt.Errorf("[backup] can't send fulllist request to pocketbase, err %w", err)
	}

	if resp.IsError() {
		return response, fmt.Errorf("[backup] pocketbase returned status: %d, msg: %s, err %w",
			resp.StatusCode(),
			resp.String(),
			ErrInvalidResponse,
		)
	}

	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return response, fmt.Errorf("[backup] can't unmarshal response, err %w", err)
	}

	return response, nil
}

type CreateRequest struct {
	Name string `json:"name"`
}

// Create initializes a new backup.
func (b Backup) Create(key ...string) error {
	if err := b.Authorize(); err != nil {
		return err
	}

	request := b.client.R().
		SetHeader("Content-Type", "application/json")
	if len(key) > 0 {
		request = request.SetMultipartFormData(map[string]string{
			"name": getZIPName(key[0]),
		})
	}

	resp, err := request.Post(b.url + "/api/backups")
	if err != nil {
		return fmt.Errorf("[backup] can't send create request to pocketbase, err %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("[backup] pocketbase returned status at creating a new backup: %d, msg: %s, err %w",
			resp.StatusCode(),
			resp.String(),
			ErrInvalidResponse,
		)
	}

	return nil
}

// Upload uploads an existing backup file.
func (b Backup) Upload(key string, reader io.Reader) error {
	if err := b.Authorize(); err != nil {
		return err
	}

	request := b.client.R().
		SetHeader("Content-Type", "application/json").
		SetMultipartFormData(map[string]string{
			"name": key,
		}).
		SetFileReader("file", key, reader)

	resp, err := request.Post(b.url + "/api/backups/upload")
	if err != nil {
		return fmt.Errorf("[backup] can't send upload request to pocketbase, err %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("[backup] pocketbase returned status at uploading a new backup: %d, msg: %s, err %w",
			resp.StatusCode(),
			resp.String(),
			ErrInvalidResponse,
		)
	}

	return nil
}

// Delete deletes a single backup file.
//
// Example:
//
//	file, _ := os.Open("./backups/pb_backup.zip")
//	defer file.Close()
//	_ = defaultClient.Backup().Upload("mybackup.zip", file)
func (b Backup) Delete(key string) error {
	if err := b.Authorize(); err != nil {
		return err
	}

	request := b.client.R().
		SetHeader("Content-Type", "application/json")

	resp, err := request.Delete(b.url + "/api/backups/" + key)
	if err != nil {
		return fmt.Errorf("[backup] can't send delete request to pocketbase, err %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("[backup] pocketbase returned status at deleting a backup: %d, msg: %s, err %w",
			resp.StatusCode(),
			resp.String(),
			ErrInvalidResponse,
		)
	}

	return nil
}

// Restore initializes an app data restore from an existing backup.
func (b Backup) Restore(key string) error {
	if err := b.Authorize(); err != nil {
		return err
	}

	request := b.client.R().
		SetHeader("Content-Type", "application/json")

	u, err := url.Parse(b.url + "/api/backups/" + strings.ToLower(key) + "/restore")
	if err != nil {
		return fmt.Errorf("[backup] pocketbase returned restoring a new backup, because of an invalid URL: err %w", err)
	}
	resp, err := request.Post(u.String())
	if err != nil {
		return fmt.Errorf("[backup] can't send create request to pocketbase, err %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("[backup] pocketbase returned status at creating a new backup: %d, msg: %s, err %w",
			resp.StatusCode(),
			resp.String(),
			ErrInvalidResponse,
		)
	}

	return nil
}

// GetDownloadToken builds a download url for a single existing backup using an
// admin file token and the backup file key.
//
// The file token can be generated via `client.Files().GetToken()`.
func (b Backup) GetDownloadURL(token string, key string) (string, error) {
	if strings.TrimSpace(token) == "" || strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("[backup] pocketbase cannot get donwload-URL because of a missing token and/or key")
	}

	if err := b.Authorize(); err != nil {
		return "", err
	}

	params := url.Values{}
	params.Add("token", token)
	encodedParams := params.Encode()
	u, err := url.Parse(
		b.url + "/api/backups/" + key + "?" + encodedParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func getZIPName(backupName string) string {
	zipName := strings.ToLower(backupName)
	if !strings.Contains(zipName, ".zip") {
		zipName = zipName + ".zip"
	}
	return zipName
}
