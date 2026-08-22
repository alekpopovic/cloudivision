package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type apiClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func (c apiClient) do(ctx context.Context, method, path string, body, output any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.baseURL, "/")+path, reader)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var apiErr struct{ Code, Message string }
		_ = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&apiErr)
		if apiErr.Message == "" {
			apiErr.Message = response.Status
		}
		return fmt.Errorf("%s: %s", first(apiErr.Code, "http_error"), apiErr.Message)
	}
	if output == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(output); err != nil {
		return fmt.Errorf("decode API response: %w", err)
	}
	return nil
}

func query(path string, values url.Values) string {
	if len(values) == 0 {
		return path
	}
	return path + "?" + values.Encode()
}
