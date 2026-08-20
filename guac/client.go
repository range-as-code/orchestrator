package guac

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var (
	ErrAuthFailed      = errors.New("authentication failed")
	ErrBadResponse     = errors.New("bad response from server")
	ErrOperationFailed = errors.New("operation failed")
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type GuacClient struct {
	BaseURL string
	Token   string
	http    httpDoer
}

func NewGuacClient(baseURL string) *GuacClient {
	return &GuacClient{
		BaseURL: baseURL,
		http:    &http.Client{},
	}
}

func (c *GuacClient) doRequest(method, urlString, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, urlString, body)
	if err != nil {
		return nil, fmt.Errorf("constructing request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	return resp, nil
}

func readJSONBody(body io.Reader, v any) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("reading body: %w", ErrBadResponse)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parsing response: %w", ErrBadResponse)
	}
	return nil
}
