package guac

import (
	"fmt"
	"net/http"
)

func (c *GuacClient) doDelete(path string) error {
	if c.Token == "" {
		return fmt.Errorf("not authenticated: %w", ErrAuthFailed)
	}

	u := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return fmt.Errorf("constructing request: %w", err)
	}
	q := req.URL.Query()
	q.Set("token", c.Token)
	req.URL.RawQuery = q.Encode()

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("status %d: %w", resp.StatusCode, ErrOperationFailed)
	}
	return nil
}

func (c *GuacClient) DeleteConnection(id string) error {
	return c.doDelete("/session/data/postgresql/connections/" + id)
}

func (c *GuacClient) DeleteGroup(id string) error {
	return c.doDelete("/session/data/postgresql/connectionGroups/" + id)
}
