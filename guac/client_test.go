package guac

import (
	"encoding/json"
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
		_, _ = w.Write([]byte(`{"authToken":"test-token","dataSource":"postgresql"}`))
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
		_, _ = w.Write([]byte(`this is not json`))
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
		_, _ = w.Write([]byte("short")) // send only 5 → truncated
	})

	client := NewGuacClient(server.URL)
	err := client.Authenticate("u", "p")

	if !errors.Is(err, ErrBadResponse) {
		t.Errorf("expected ErrBadResponse, got %v", err)
	}
}

func TestCreateConnection(t *testing.T) {
	server := newTestServerWith(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/session/data/postgresql/connections" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/session/data/postgresql/connections")
		}
		if got := r.URL.Query().Get("token"); got != "test-token" {
			t.Fatalf("token = %q, want %q", got, "test-token")
		}

		var reqBody connectionRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if reqBody.ParentIdentifier != "parent-id" {
			t.Fatalf("parentIdentifier = %q, want %q", reqBody.ParentIdentifier, "parent-id")
		}
		if reqBody.Name != "db" {
			t.Fatalf("name = %q, want %q", reqBody.Name, "db")
		}
		if reqBody.Protocol != "vnc" {
			t.Fatalf("protocol = %q, want %q", reqBody.Protocol, "vnc")
		}
		if reqBody.Parameters["hostname"] != "example.com" {
			t.Fatalf("hostname = %q, want %q", reqBody.Parameters["hostname"], "example.com")
		}
		if reqBody.Parameters["port"] != "443" {
			t.Fatalf("port = %q, want %q", reqBody.Parameters["port"], "443")
		}
		if reqBody.Parameters["username"] != "alice" {
			t.Fatalf("username = %q, want %q", reqBody.Parameters["username"], "alice")
		}
		if reqBody.Parameters["password"] != "secret" {
			t.Fatalf("password = %q, want %q", reqBody.Parameters["password"], "secret")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"identifier":"conn-123"}`))
	})

	client := NewGuacClient(server.URL)
	client.Token = "test-token"

	got, err := client.CreateConnection(ConnectionSpec{
		Name:             "db",
		Protocol:         "vnc",
		Hostname:         "example.com",
		Port:             "443",
		Username:         "alice",
		Password:         "secret",
		ParentIdentifier: "parent-id",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "conn-123" {
		t.Fatalf("identifier = %q, want %q", got, "conn-123")
	}
}

func TestCreateConnection_NotAuthenticated(t *testing.T) {
	client := NewGuacClient("http://example.com")

	_, err := client.CreateConnection(ConnectionSpec{Name: "db"})
	if !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

func TestCreateConnection_FailSend(t *testing.T) {
	client := NewGuacClient("http://example.com")
	client.Token = "test-token"
	client.http = failingDoer{}

	_, err := client.CreateConnection(ConnectionSpec{Name: "db"})
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestCreateConnection_BadStatus(t *testing.T) {
	server := newTestServerWith(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"identifier":"conn-123"}`))
	})

	client := NewGuacClient(server.URL)
	client.Token = "test-token"

	_, err := client.CreateConnection(ConnectionSpec{Name: "db"})
	if !errors.Is(err, ErrOperationFailed) {
		t.Fatalf("expected ErrOperationFailed, got %v", err)
	}
}

func TestCreateConnection_MalformedJSON(t *testing.T) {
	server := newTestServerWith(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`this is not json`))
	})

	client := NewGuacClient(server.URL)
	client.Token = "test-token"

	_, err := client.CreateConnection(ConnectionSpec{Name: "db"})
	if !errors.Is(err, ErrBadResponse) {
		t.Fatalf("expected ErrBadResponse, got %v", err)
	}
}

func TestCreateConnection_UnreadableBody(t *testing.T) {
	server := newTestServerWith(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("short"))
	})

	client := NewGuacClient(server.URL)
	client.Token = "test-token"

	_, err := client.CreateConnection(ConnectionSpec{Name: "db"})
	if !errors.Is(err, ErrBadResponse) {
		t.Fatalf("expected ErrBadResponse, got %v", err)
	}
}
