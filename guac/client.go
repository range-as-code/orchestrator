package guac

import (
	"net/http"
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
