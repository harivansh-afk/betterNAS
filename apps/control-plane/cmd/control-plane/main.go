package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	port := env("PORT", "8081")
	app, err := newAppFromEnv(time.Now())
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           app.handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("betterNAS control plane listening on :%s", port)
	log.Fatal(server.ListenAndServe())
}

func newAppFromEnv(startedAt time.Time) (*app, error) {
	clientToken, err := requiredEnv("BETTERNAS_CONTROL_PLANE_CLIENT_TOKEN")
	if err != nil {
		return nil, err
	}

	nodeBootstrapToken, err := requiredEnv("BETTERNAS_CONTROL_PLANE_NODE_BOOTSTRAP_TOKEN")
	if err != nil {
		return nil, err
	}

	return newApp(
		appConfig{
			version:            env("BETTERNAS_VERSION", "0.1.0-dev"),
			nextcloudBaseURL:   env("NEXTCLOUD_BASE_URL", ""),
			statePath:          env("BETTERNAS_CONTROL_PLANE_STATE_PATH", ".state/control-plane/state.json"),
			clientToken:        clientToken,
			nodeBootstrapToken: nodeBootstrapToken,
		},
		startedAt,
	)
}
