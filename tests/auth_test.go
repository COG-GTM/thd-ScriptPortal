package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/homedepot/ScriptPortal/endpoints/auth"
)

func TestRequireNoCredentials(t *testing.T) {
	config := auth.Config{Users: map[string]string{"admin": "secret"}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	config.RequireFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("WWW-Authenticate"); got != `Basic realm="ScriptPortal"` {
		t.Fatalf("expected WWW-Authenticate header, got %q", got)
	}
	if recorder.Body.String() != "unauthorized" {
		t.Fatalf("expected unauthorized body, got %q", recorder.Body.String())
	}
}

func TestRequireWrongPassword(t *testing.T) {
	config := auth.Config{Users: map[string]string{"admin": "secret"}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.SetBasicAuth("admin", "wrong")

	config.RequireFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", recorder.Code)
	}
}

func TestRequireRightCredentialsStoresUser(t *testing.T) {
	config := auth.Config{Users: map[string]string{"admin": "secret"}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.SetBasicAuth("admin", "secret")

	config.RequireFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFromContext(r.Context())
		if !ok || user != "admin" {
			t.Errorf("expected authenticated user admin, got %q, %t", user, ok)
		}
		w.WriteHeader(http.StatusOK)
	})(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestLoadFromEnvRequiresAuthentication(t *testing.T) {
	t.Setenv("SCRIPTPORTAL_BASIC_AUTH_USERS", "")
	t.Setenv("SCRIPTPORTAL_ALLOW_UNAUTHENTICATED", "")

	_, err := auth.LoadFromEnv()
	if err == nil {
		t.Fatal("expected authentication configuration error")
	}
}

func TestLoadFromEnvAllowsExplicitUnauthenticatedMode(t *testing.T) {
	t.Setenv("SCRIPTPORTAL_BASIC_AUTH_USERS", "")
	t.Setenv("SCRIPTPORTAL_ALLOW_UNAUTHENTICATED", "true")

	config, err := auth.LoadFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if config.Enabled() {
		t.Fatal("expected authentication to be disabled")
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	config.RequireFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected pass-through status 200, got %d", recorder.Code)
	}
}

func TestLoadFromEnvRejectsMalformedEntry(t *testing.T) {
	t.Setenv("SCRIPTPORTAL_BASIC_AUTH_USERS", "nouser")
	t.Setenv("SCRIPTPORTAL_ALLOW_UNAUTHENTICATED", "")

	_, err := auth.LoadFromEnv()
	if err == nil {
		t.Fatal("expected malformed entry error")
	}
}
