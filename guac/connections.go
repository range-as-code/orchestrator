package guac

import (
	"bytes"
	"encoding/json"
	"fmt"
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
