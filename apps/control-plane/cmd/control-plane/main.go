package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type jsonObject map[string]any

func main() {
	port := env("PORT", "8081")
	startedAt := time.Now()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, jsonObject{
			"service":       "control-plane",
			"status":        "ok",
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
			"uptimeSeconds": int(time.Since(startedAt).Seconds()),
			"nextcloud": jsonObject{
				"configured": false,
				"baseUrl":    env("NEXTCLOUD_BASE_URL", ""),
				"provider":   "nextcloud",
			},
		})
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, jsonObject{
			"service":    "control-plane",
			"version":    env("BETTERNAS_VERSION", "0.1.0-dev"),
			"apiVersion": "v1",
		})
	})
	mux.HandleFunc("/api/v1/exports", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []jsonObject{})
	})
	mux.HandleFunc("/api/v1/mount-profiles/issue", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, jsonObject{
			"id":             "dev-profile",
			"exportId":       "dev-export",
			"protocol":       "webdav",
			"displayName":    "Example export",
			"mountUrl":       env("BETTERNAS_EXAMPLE_MOUNT_URL", "http://localhost:8090/dav"),
			"readonly":       false,
			"credentialMode": "session-token",
		})
	})
	mux.HandleFunc("/api/v1/cloud-profiles/issue", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, jsonObject{
			"id":       "dev-cloud",
			"exportId": "dev-export",
			"provider": "nextcloud",
			"baseUrl":  env("NEXTCLOUD_BASE_URL", "http://localhost:8080"),
			"path":     "/apps/files/files",
		})
	})
	mux.HandleFunc("/api/v1/nodes/register", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, jsonObject{
			"id":            "dev-node",
			"machineId":     "dev-machine",
			"displayName":   "Development NAS",
			"agentVersion":  "0.1.0-dev",
			"status":        "online",
			"lastSeenAt":    time.Now().UTC().Format(time.RFC3339),
			"directAddress": env("BETTERNAS_NODE_DIRECT_ADDRESS", "http://localhost:8090"),
			"relayAddress":  nil,
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("betterNAS control plane listening on :%s", port)
	log.Fatal(server.ListenAndServe())
}

func env(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
