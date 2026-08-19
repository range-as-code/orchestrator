package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type GuacClient struct {
	BaseURL string
	Token   string
	http    *http.Client
}
type authResponse struct {
	AuthToken  string `json:"authToken"`
	DataSource string `json:"dataSource"`
}

func (c *GuacClient) Authenticate(user, pass string) error {
	data := url.Values{}
	data.Set("username", user)
	data.Set("password", pass)

	urlStr := c.BaseURL + "/tokens"

	req, err := http.NewRequest(
		http.MethodPost,
		urlStr,
		strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to construct request: %w", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth failed: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}
	var authresp authResponse
	if err := json.Unmarshal(body, &authresp); err != nil {
		return fmt.Errorf("cannot unmarshal JSON: %w", err)
	}

	c.Token = authresp.AuthToken

	return nil
}

func NewGuacClient(baseURL string) *GuacClient {
	return &GuacClient{
		BaseURL: baseURL,
		http:    &http.Client{},
	}
}

func GetEnvString(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("required env var %s is not set", key)
	}
	return v, nil
}

func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}

	return val
}

func main() {

	baseurl := Must(GetEnvString("GUAC_BASE_URL"))
	username := Must(GetEnvString("GUAC_USERNAME"))
	password := Must(GetEnvString("GUAC_PASSWORD"))

	client := NewGuacClient(baseurl)
	if err := client.Authenticate(username, password); err != nil {
		log.Fatalf("authentication failed: %v", err)
	}
	fmt.Println("token: ", client.Token)

}
