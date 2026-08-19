package guac

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ConnectionSpec struct {
	Name     string
	Protocol string
	Hostname string
	Port     string
	Username string
	Password string
}

type connectionRequest struct {
	ParentIdentifier string            `json:"parentIdentifier"`
	Name             string            `json:"name"`
	Protocol         string            `json:"protocol"`
	Parameters       map[string]string `json:"parameters"`
	Attributes       map[string]string `json:"attributes"`
}

type connectionResponse struct {
	Identifier string `json:"identifier"`
}

func (c *GuacClient) CreateConnection(spec ConnectionSpec) (string, error) {
	if c.Token == "" {
		return "", fmt.Errorf("not authenticated: %w", ErrAuthFailed)
	}

	var requestBody connectionRequest
	requestBody.ParentIdentifier = "ROOT"
	requestBody.Name = spec.Name
	requestBody.Protocol = spec.Protocol
	requestBody.Parameters = map[string]string{
		"hostname": spec.Hostname,
		"port":     spec.Port,
		"username": spec.Username,
		"password": spec.Password,
	}
	requestBody.Attributes = map[string]string{}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("constructing body: %w", err)
	}

	urlString := c.BaseURL + "/session/data/postgresql/connections?token=" + c.Token

	req, err := http.NewRequest(
		http.MethodPost,
		urlString,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return "", fmt.Errorf("constructing request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %w", resp.StatusCode, ErrOperationFailed)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading body: %w", ErrBadResponse)
	}

	var connectionResponse connectionResponse
	if err := json.Unmarshal(body, &connectionResponse); err != nil {
		return "", fmt.Errorf("parsing response: %w", ErrBadResponse)
	}

	return connectionResponse.Identifier, nil
}
