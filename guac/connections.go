package guac

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ConnectionSpec struct {
	Name             string
	Protocol         string
	Hostname         string
	Port             string
	Username         string
	Password         string
	ParentIdentifier string
}

type connectionRequest struct {
	ParentIdentifier string            `json:"parentIdentifier"`
	Name             string            `json:"name"`
	Protocol         string            `json:"protocol"`
	Parameters       map[string]string `json:"parameters"`
	Attributes       map[string]string `json:"attributes"`
}

type ConnectionGroupSpec struct {
	Name             string
	ParentIdentifier string
	Type             string
}

type connectionGroupRequest struct {
	ParentIdentifier string            `json:"parentIdentifier"`
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	Attributes       map[string]string `json:"attributes"`
}

type guacCreationResponse struct {
	Identifier string `json:"identifier"`
}

func (c *GuacClient) CreateConnection(spec ConnectionSpec) (string, error) {
	requestBody := connectionRequest{
		ParentIdentifier: spec.ParentIdentifier,
		Name:             spec.Name,
		Protocol:         spec.Protocol,
		Parameters: map[string]string{
			"hostname": spec.Hostname,
			"port":     spec.Port,
			"username": spec.Username,
			"password": spec.Password,
		},
		Attributes: map[string]string{},
	}

	return c.createResource("/session/data/postgresql/connections", requestBody)
}

func (c *GuacClient) CreateConnectionGroup(spec ConnectionGroupSpec) (string, error) {
	requestBody := connectionGroupRequest{
		ParentIdentifier: spec.ParentIdentifier,
		Name:             spec.Name,
		Type:             spec.Type,
		Attributes:       map[string]string{},
	}

	return c.createResource("/session/data/postgresql/connectionGroups", requestBody)
}

func (c *GuacClient) createResource(path string, payload any) (string, error) {
	if c.Token == "" {
		return "", fmt.Errorf("not authenticated: %w", ErrAuthFailed)
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("constructing body: %w", err)
	}

	resp, err := c.doRequest(
		http.MethodPost,
		c.BaseURL+path+"?token="+c.Token,
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %w", resp.StatusCode, ErrOperationFailed)
	}

	var result guacCreationResponse
	if err := readJSONBody(resp.Body, &result); err != nil {
		return "", err
	}

	return result.Identifier, nil
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
