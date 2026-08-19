package guac

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type failingDoer struct{}

func (failingDoer) Do(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("simulated network failure")
}

func newTestServerWith(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(func() { server.Close() })
	return server
}

func TestAuthenticate(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"authToken":"test-token","dataSource":"postgresql"}`))
	})
	server := newTestServerWith(t, handler)

	client := NewGuacClient(server.URL)
	if err := client.Authenticate("u", "p"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Token != "test-token" {
		t.Errorf("got token %q, want %q", client.Token, "test-token")
	}
}

func TestAuthenticate_BadURL(t *testing.T) {
	client := NewGuacClient("http://invalid\x00url")
	err := client.Authenticate("u", "p")
	if err == nil {
		t.Fatal("expected error for malformed URL, got nil")
	}
}

func TestAuthenticate_FailSend(t *testing.T) {
	client := NewGuacClient("localhost")
	client.http = failingDoer{}

	err := client.Authenticate("u", "p")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestAuthenticate_BadStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"authToken":"test-token","dataSource":"postgresql"}`))
	})
	server := newTestServerWith(t, handler)

	client := NewGuacClient(server.URL)
	err := client.Authenticate("u", "p")
	if !errors.Is(err, ErrAuthFailed) {
		t.Errorf("expected ErrAuthFailed, got %v", err)
	}

}

func TestAuthenticate_MalformedJSON(t *testing.T) {
	server := newTestServerWith(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`this is not json`))
	})

	client := NewGuacClient(server.URL)
	err := client.Authenticate("u", "p")

	if !errors.Is(err, ErrBadResponse) {
		t.Errorf("expected ErrBadResponse, got %v", err)
	}
}

func TestAuthenticate_UnreadableBody(t *testing.T) {
	server := newTestServerWith(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000") // promise 1000 bytes
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("short")) // send only 5 → truncated
	})

	client := NewGuacClient(server.URL)
	err := client.Authenticate("u", "p")

	if !errors.Is(err, ErrBadResponse) {
		t.Errorf("expected ErrBadResponse, got %v", err)
	}
}
