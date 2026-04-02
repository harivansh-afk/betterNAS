package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRunNodeHeartbeatLoopSendsRecurringHeartbeats(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var heartbeats []nodeHeartbeatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/nodes/dev-node/heartbeat" {
			http.NotFound(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read heartbeat body: %v", err)
		}
		_ = r.Body.Close()

		var heartbeat nodeHeartbeatRequest
		if err := json.Unmarshal(body, &heartbeat); err != nil {
			t.Fatalf("decode heartbeat body: %v", err)
		}

		mu.Lock()
		heartbeats = append(heartbeats, heartbeat)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		runNodeHeartbeatLoop(
			ctx,
			server.Client(),
			server.URL,
			"session-token",
			"dev-node",
			10*time.Millisecond,
			func() time.Time { return time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC) },
			log.New(io.Discard, "", 0),
		)
		close(done)
	}()

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		mu.Lock()
		count := len(heartbeats)
		mu.Unlock()
		if count >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected recurring heartbeats, got %d", count)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("heartbeat loop did not stop after context cancellation")
	}

	mu.Lock()
	defer mu.Unlock()
	for _, heartbeat := range heartbeats {
		if heartbeat.NodeID != "dev-node" {
			t.Fatalf("expected node ID dev-node, got %q", heartbeat.NodeID)
		}
		if heartbeat.Status != "online" {
			t.Fatalf("expected status online, got %q", heartbeat.Status)
		}
		if heartbeat.LastSeenAt != "2025-01-01T12:00:00Z" {
			t.Fatalf("expected fixed lastSeenAt, got %q", heartbeat.LastSeenAt)
		}
	}
}
