package guac

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type authResponse struct {
	AuthToken  string `json:"authToken"`
	DataSource string `json:"dataSource"`
}

func (c *GuacClient) Authenticate(user, pass string) error {
	data := url.Values{}
	data.Set("username", user)
	data.Set("password", pass)

	resp, err := c.doRequest(
		http.MethodPost,
		c.BaseURL+"/tokens",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d: %w", resp.StatusCode, ErrAuthFailed)
	}

	var authResp authResponse
	if err := readJSONBody(resp.Body, &authResp); err != nil {
		return err
	}

	c.Token = authResp.AuthToken
	return nil
}
