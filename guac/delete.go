package guac

import (
	"fmt"
	"net/http"
)

// Return true if delete was successful, return false if not
// Set error if error occurs
func (c *GuacClient) doDelete(path string) (bool, error) {
	if c.Token == "" {
		return false, fmt.Errorf("not authenticated: %w", ErrAuthFailed)
	}
	u := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return false, fmt.Errorf("constructing request: %w", err)
	}
	q := req.URL.Query()
	q.Set("token", c.Token)
	req.URL.RawQuery = q.Encode()

	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("status %d: %w", resp.StatusCode, ErrOperationFailed)
	}
}

func (c *GuacClient) DeleteConnection(id string) (bool, error) {
	return c.doDelete("/session/data/postgresql/connections/" + id)
}
func (c *GuacClient) DeleteGroup(id string) (bool, error) {
	return c.doDelete("/session/data/postgresql/connectionGroups/" + id)
}
