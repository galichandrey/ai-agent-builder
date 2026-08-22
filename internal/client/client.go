package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ag/ai-agent-builder/internal/config"
)

// LangflowClient is an HTTP client for the LangFlow API.
type LangflowClient struct {
	baseURL       string
	apiKey        string
	httpClient    *http.Client
	customHeaders map[string]string
}

// NewClient creates a LangflowClient from the given config.
func NewClient(cfg *config.Config) *LangflowClient {
	timeout := time.Duration(cfg.RequestTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &LangflowClient{
		baseURL:       strings.TrimRight(cfg.LangflowURL, "/"),
		apiKey:        cfg.APIKey,
		httpClient:    &http.Client{Timeout: timeout},
		customHeaders: cfg.CustomHeaders,
	}
}

// doRequest executes an HTTP request and returns the response body bytes.
func (c *LangflowClient) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	url := c.baseURL + "/api/v1" + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.applyHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(data),
			Message:    http.StatusText(resp.StatusCode),
		}
	}

	return data, nil
}

// applyHeaders adds auth and custom headers to the request.
func (c *LangflowClient) applyHeaders(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}
	for k, v := range c.customHeaders {
		req.Header.Set(k, v)
	}
}

// doGet performs a GET request and returns the response body bytes.
func (c *LangflowClient) doGet(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// doPost performs a POST request with a JSON body and returns the response body bytes.
func (c *LangflowClient) doPost(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

// doPatch performs a PATCH request with a JSON body and returns the response body bytes.
func (c *LangflowClient) doPatch(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPatch, path, body)
}

// doDelete performs a DELETE request and returns an error if the request fails.
func (c *LangflowClient) doDelete(ctx context.Context, path string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// doGetStream performs a GET request and returns a streaming reader for NDJSON responses.
func (c *LangflowClient) doGetStream(ctx context.Context, path string) (io.ReadCloser, error) {
	url := c.baseURL + "/api/v1" + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.applyHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(data),
			Message:    http.StatusText(resp.StatusCode),
		}
	}

	return resp.Body, nil
}

// doPostStream performs a POST request with a JSON body and returns a streaming
// reader for NDJSON responses (used by the build endpoint).
func (c *LangflowClient) doPostStream(ctx context.Context, path string, body any) (io.ReadCloser, error) {
	url := c.baseURL + "/api/v1" + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.applyHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(data),
			Message:    http.StatusText(resp.StatusCode),
		}
	}

	return resp.Body, nil
}

// HTTPError represents a non-2xx HTTP response.
type HTTPError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *HTTPError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("HTTP %d (%s): %s", e.StatusCode, e.Message, e.Body)
	}
	return fmt.Sprintf("HTTP %d (%s)", e.StatusCode, e.Message)
}
