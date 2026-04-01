package nodeagent

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegistrationRequestUsesEmptyTagsArray(t *testing.T) {
	t.Parallel()

	loop := newRegistrationLoop(Config{
		MachineID:    "nas-1",
		DisplayName:  "NAS 1",
		AgentVersion: "test-version",
		ExportPath:   t.TempDir(),
		ExportLabel:  "archive",
	}, log.New(io.Discard, "", 0))

	request := loop.registrationRequest()
	if request.Exports[0].Tags == nil {
		t.Fatal("tags slice = nil, want empty slice")
	}

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal registration request: %v", err)
	}

	if !bytes.Contains(body, []byte(`"tags":[]`)) {
		t.Fatalf("registration json = %s, want empty tags array", string(body))
	}
}

func TestHeartbeatRouteEscapesOpaqueNodeID(t *testing.T) {
	t.Parallel()

	got := heartbeatRoute("node/123")
	want := "/api/v1/nodes/node%2F123/heartbeat"
	if got != want {
		t.Fatalf("heartbeatRoute returned %q, want %q", got, want)
	}
}

func TestHeartbeatRouteUnsupportedDetectsDefinitiveUnsupportedRoute(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		err             error
		wantUnsupported bool
	}{
		{
			name: "not found",
			err: &responseStatusError{
				route:      heartbeatRoute("node/123"),
				statusCode: http.StatusNotFound,
				message:    "missing",
			},
			wantUnsupported: false,
		},
		{
			name: "method not allowed",
			err: &responseStatusError{
				route:      heartbeatRoute("node/123"),
				statusCode: http.StatusMethodNotAllowed,
				message:    "method not allowed",
			},
			wantUnsupported: true,
		},
		{
			name: "not implemented",
			err: &responseStatusError{
				route:      heartbeatRoute("node/123"),
				statusCode: http.StatusNotImplemented,
				message:    "not implemented",
			},
			wantUnsupported: true,
		},
		{
			name: "temporary failure",
			err: &responseStatusError{
				route:      heartbeatRoute("node/123"),
				statusCode: http.StatusBadGateway,
				message:    "bad gateway",
			},
			wantUnsupported: false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := heartbeatRouteUnsupported(testCase.err)
			if got != testCase.wantUnsupported {
				t.Fatalf("heartbeatRouteUnsupported(%v) = %t, want %t", testCase.err, got, testCase.wantUnsupported)
			}
		})
	}
}

func TestHeartbeatRequiresRegistrationRefreshDetectsRejectedNode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		err         error
		wantRefresh bool
	}{
		{
			name: "not found",
			err: &responseStatusError{
				route:      heartbeatRoute("node/123"),
				statusCode: http.StatusNotFound,
				message:    "missing",
			},
			wantRefresh: true,
		},
		{
			name: "gone",
			err: &responseStatusError{
				route:      heartbeatRoute("node/123"),
				statusCode: http.StatusGone,
				message:    "gone",
			},
			wantRefresh: true,
		},
		{
			name: "temporary failure",
			err: &responseStatusError{
				route:      heartbeatRoute("node/123"),
				statusCode: http.StatusBadGateway,
				message:    "bad gateway",
			},
			wantRefresh: false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := heartbeatRequiresRegistrationRefresh(testCase.err)
			if got != testCase.wantRefresh {
				t.Fatalf("heartbeatRequiresRegistrationRefresh(%v) = %t, want %t", testCase.err, got, testCase.wantRefresh)
			}
		})
	}
}

func TestPostJSONAddsBearerAuthorization(t *testing.T) {
	t.Parallel()

	requestHeaders := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestHeaders <- r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"node-1"}`)
	}))
	defer server.Close()

	loop := newRegistrationLoop(Config{
		ControlPlaneURL:   server.URL,
		ControlPlaneToken: "node-auth-token",
	}, log.New(io.Discard, "", 0))

	var response nodeRegistrationResponse
	if err := loop.postJSON(context.Background(), registerNodeRoute, nodeRegistrationRequest{}, http.StatusOK, &response); err != nil {
		t.Fatalf("post json: %v", err)
	}

	headers := <-requestHeaders
	if got := headers.Get("Authorization"); got != "Bearer node-auth-token" {
		t.Fatalf("authorization header = %q, want Bearer token", got)
	}
}

func TestPostJSONOmitsBearerAuthorizationWhenTokenUnset(t *testing.T) {
	t.Parallel()

	requestHeaders := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestHeaders <- r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"node-1"}`)
	}))
	defer server.Close()

	loop := newRegistrationLoop(Config{
		ControlPlaneURL: server.URL,
	}, log.New(io.Discard, "", 0))

	var response nodeRegistrationResponse
	if err := loop.postJSON(context.Background(), registerNodeRoute, nodeRegistrationRequest{}, http.StatusOK, &response); err != nil {
		t.Fatalf("post json: %v", err)
	}

	headers := <-requestHeaders
	if got := headers.Get("Authorization"); got != "" {
		t.Fatalf("authorization header = %q, want empty", got)
	}
}
