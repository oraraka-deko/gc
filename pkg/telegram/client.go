package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Ptr returns a pointer to any value.
func Ptr[T any](v T) *T {
	return &v
}

// IntPtr returns an int pointer.
func IntPtr(v int) *int {
	return &v
}

// StringPtr returns a string pointer.
func StringPtr(v string) *string {
	return &v
}

// BoolPtr returns a bool pointer.
func BoolPtr(v bool) *bool {
	return &v
}

// DefaultBaseURL is the official Telegram Bot API endpoint.
const DefaultBaseURL = "https://api.telegram.org"

// APIError is returned when Telegram Bot API returns ok: false.
type APIError struct {
	StatusCode  int
	ErrorCode   int
	Description string
	Parameters  *ResponseParameters
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram api error: code=%d, description=%s", e.ErrorCode, e.Description)
}

// InputFile represents a file to be uploaded via multipart/form-data.
type InputFile struct {
	Name     string
	Reader   io.Reader
	FilePath string
}

// Client is the Telegram Bot API client.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// Option allows configuring the Client.
type Option func(*Client)

// WithBaseURL sets a custom API base URL (e.g. for local Bot API server).
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(url, "/")
	}
}

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithTimeout sets a timeout on the default http.Client.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// NewClient initializes a new Telegram Bot API client.
func NewClient(token string, opts ...Option) *Client {
	c := &Client{
		token:   token,
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Token returns the current bot token.
func (c *Client) Token() string {
	return c.token
}

// SetToken updates the bot token.
func (c *Client) SetToken(token string) {
	c.token = token
}

// apiEndpoint constructs the endpoint URL for a given method.
func (c *Client) apiEndpoint(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.token, method)
}

// Do executes a generic JSON POST or GET request against the Telegram Bot API.
func Do[T any](ctx context.Context, c *Client, method string, payload any) (*ApiResponse[T], error) {
	endpoint := c.apiEndpoint(method)

	var reqBody io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp ApiResponse[T]
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response (status %d): %w (raw: %s)", resp.StatusCode, err, string(respBytes))
	}

	if !apiResp.Ok {
		return &apiResp, &APIError{
			StatusCode:  resp.StatusCode,
			ErrorCode:   apiResp.ErrorCode,
			Description: apiResp.Description,
			Parameters:  apiResp.Parameters,
		}
	}

	return &apiResp, nil
}

// DoMultipart sends fields and uploaded files using multipart/form-data.
func DoMultipart[T any](ctx context.Context, c *Client, method string, fields map[string]string, files map[string]InputFile) (*ApiResponse[T], error) {
	endpoint := c.apiEndpoint(method)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("failed to write multipart field %q: %w", k, err)
		}
	}

	for fieldName, file := range files {
		var reader io.Reader
		fileName := file.Name

		if file.FilePath != "" {
			f, err := os.Open(file.FilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to open file %q: %w", file.FilePath, err)
			}
			defer f.Close()
			reader = f
			if fileName == "" {
				fileName = filepath.Base(file.FilePath)
			}
		} else if file.Reader != nil {
			reader = file.Reader
			if fileName == "" {
				fileName = "file.dat"
			}
		} else {
			continue
		}

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName))
		h.Set("Content-Type", "application/octet-stream")
		part, err := writer.CreatePart(h)
		if err != nil {
			return nil, fmt.Errorf("failed to create multipart for %q: %w", fieldName, err)
		}
		if _, err := io.Copy(part, reader); err != nil {
			return nil, fmt.Errorf("failed to stream file %q into part: %w", fieldName, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("multipart request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp ApiResponse[T]
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response (status %d): %w", resp.StatusCode, err)
	}

	if !apiResp.Ok {
		return &apiResp, &APIError{
			StatusCode:  resp.StatusCode,
			ErrorCode:   apiResp.ErrorCode,
			Description: apiResp.Description,
			Parameters:  apiResp.Parameters,
		}
	}

	return &apiResp, nil
}
