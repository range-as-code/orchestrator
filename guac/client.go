package guac

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type GuacClient struct {
	BaseURL string
	Token   string
	http    httpDoer
}
type authResponse struct {
	AuthToken  string `json:"authToken"`
	DataSource string `json:"dataSource"`
}

var (
	ErrAuthFailed  = errors.New("authentication failed")
	ErrBadResponse = errors.New("bad response from server")
)

func (c *GuacClient) Authenticate(user, pass string) error {
	data := url.Values{}
	data.Set("username", user)
	data.Set("password", pass)

	req, err := http.NewRequest(
		http.MethodPost,
		c.BaseURL+"/tokens",
		strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("constructing request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d: %w", resp.StatusCode, ErrAuthFailed)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading body: %w", ErrBadResponse)
	}

	var authResp authResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return fmt.Errorf("parsing response: %w", ErrBadResponse)
	}

	c.Token = authResp.AuthToken
	return nil
}

func NewGuacClient(baseURL string) *GuacClient {
	return &GuacClient{
		BaseURL: baseURL,
		http:    &http.Client{},
	}
}
