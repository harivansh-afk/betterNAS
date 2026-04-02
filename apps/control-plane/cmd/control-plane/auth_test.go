package main

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func postJSONAuthCreated[T any](t *testing.T, client *http.Client, token string, endpoint string, payload any) T {
	t.Helper()

	response := postJSONAuthResponse(t, client, token, endpoint, payload)
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(response.Body)
		t.Fatalf("post %s: expected status 201, got %d: %s", endpoint, response.StatusCode, responseBody)
	}

	var decoded T
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode %s response: %v", endpoint, err)
	}

	return decoded
}

func TestAuthRegisterLoginLogoutMe(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{
		version:             "test-version",
		registrationEnabled: true,
		sessionTTL:          time.Hour,
	})
	defer server.Close()

	// Register.
	reg := postJSONAuthCreated[authLoginResponse](t, server.Client(), "", server.URL+"/api/v1/auth/register", authRegisterRequest{
		Username: "testuser",
		Password: "password123",
	})
	if reg.Token == "" {
		t.Fatal("expected session token from registration")
	}
	if reg.User.Username != "testuser" {
		t.Fatalf("expected username %q, got %q", "testuser", reg.User.Username)
	}
	if reg.User.ID == "" {
		t.Fatal("expected user ID")
	}

	// /me with the registration token.
	me := getJSONAuth[user](t, server.Client(), reg.Token, server.URL+"/api/v1/auth/me")
	if me.Username != "testuser" {
		t.Fatalf("expected username %q from /me, got %q", "testuser", me.Username)
	}

	// Use session to list exports (client auth).
	exports := getJSONAuth[[]storageExport](t, server.Client(), reg.Token, server.URL+"/api/v1/exports")
	if len(exports) != 0 {
		t.Fatalf("expected 0 exports, got %d", len(exports))
	}

	// Login with same credentials.
	login := postJSONAuth[authLoginResponse](t, server.Client(), "", server.URL+"/api/v1/auth/login", authLoginRequest{
		Username: "testuser",
		Password: "password123",
	})
	if login.Token == "" {
		t.Fatal("expected session token from login")
	}
	if login.Token == reg.Token {
		t.Fatal("expected login to issue a different token than registration")
	}

	// Logout the registration token.
	postJSONAuthStatus(t, server.Client(), reg.Token, server.URL+"/api/v1/auth/logout", nil, http.StatusNoContent)

	// Old token should be invalid now.
	getStatusWithAuth(t, server.Client(), reg.Token, server.URL+"/api/v1/auth/me", http.StatusUnauthorized)

	// Login token still works.
	me = getJSONAuth[user](t, server.Client(), login.Token, server.URL+"/api/v1/auth/me")
	if me.Username != "testuser" {
		t.Fatalf("expected username %q, got %q", "testuser", me.Username)
	}
}

func TestAuthDuplicateUsername(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{
		version:             "test-version",
		registrationEnabled: true,
	})
	defer server.Close()

	postJSONAuthCreated[authLoginResponse](t, server.Client(), "", server.URL+"/api/v1/auth/register", authRegisterRequest{
		Username: "taken",
		Password: "password123",
	})

	postJSONAuthStatus(t, server.Client(), "", server.URL+"/api/v1/auth/register", authRegisterRequest{
		Username: "taken",
		Password: "different456",
	}, http.StatusConflict)
}

func TestAuthBadCredentials(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{
		version:             "test-version",
		registrationEnabled: true,
	})
	defer server.Close()

	postJSONAuthCreated[authLoginResponse](t, server.Client(), "", server.URL+"/api/v1/auth/register", authRegisterRequest{
		Username: "realuser",
		Password: "correctpassword",
	})

	postJSONAuthStatus(t, server.Client(), "", server.URL+"/api/v1/auth/login", authLoginRequest{
		Username: "realuser",
		Password: "wrongpassword",
	}, http.StatusUnauthorized)

	postJSONAuthStatus(t, server.Client(), "", server.URL+"/api/v1/auth/login", authLoginRequest{
		Username: "nosuchuser",
		Password: "anything",
	}, http.StatusUnauthorized)
}

func TestAuthRegistrationDisabled(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{
		version:             "test-version",
		registrationEnabled: false,
	})
	defer server.Close()

	postJSONAuthStatus(t, server.Client(), "", server.URL+"/api/v1/auth/register", authRegisterRequest{
		Username: "blocked",
		Password: "password123",
	}, http.StatusForbidden)
}

func TestAuthValidation(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{
		version:             "test-version",
		registrationEnabled: true,
	})
	defer server.Close()

	// Username too short.
	postJSONAuthStatus(t, server.Client(), "", server.URL+"/api/v1/auth/register", authRegisterRequest{
		Username: "ab",
		Password: "password123",
	}, http.StatusBadRequest)

	// Password too short.
	postJSONAuthStatus(t, server.Client(), "", server.URL+"/api/v1/auth/register", authRegisterRequest{
		Username: "validuser",
		Password: "short",
	}, http.StatusBadRequest)
}

func TestAuthSessionUsedForClientEndpoints(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{
		version:             "test-version",
		registrationEnabled: true,
	})
	defer server.Close()

	// Without auth, exports should fail.
	getStatusWithAuth(t, server.Client(), "", server.URL+"/api/v1/exports", http.StatusUnauthorized)

	// Register and get session.
	reg := postJSONAuthCreated[authLoginResponse](t, server.Client(), "", server.URL+"/api/v1/auth/register", authRegisterRequest{
		Username: "admin",
		Password: "password123",
	})

	// Session should work for client endpoints.
	exports := getJSONAuth[[]storageExport](t, server.Client(), reg.Token, server.URL+"/api/v1/exports")
	if exports == nil {
		t.Fatal("expected exports list, got nil")
	}
}

func TestAuthSessionIsTheOnlyClientAuthPath(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{
		version:             "test-version",
		registrationEnabled: true,
	})
	defer server.Close()

	reg := postJSONAuthCreated[authLoginResponse](t, server.Client(), "", server.URL+"/api/v1/auth/register", authRegisterRequest{
		Username: "sessiononly",
		Password: "password123",
	})

	exports := getJSONAuth[[]storageExport](t, server.Client(), reg.Token, server.URL+"/api/v1/exports")
	if exports == nil {
		t.Fatal("expected exports list, got nil")
	}

	getStatusWithAuth(t, server.Client(), "static-fallback-token", server.URL+"/api/v1/exports", http.StatusUnauthorized)
	getStatusWithAuth(t, server.Client(), "wrong", server.URL+"/api/v1/exports", http.StatusUnauthorized)
}
