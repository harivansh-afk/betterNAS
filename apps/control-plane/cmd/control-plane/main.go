package main

import (
	"log"
	"net/http"
	"strings"
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
	nodeBootstrapToken, err := requiredEnv("BETTERNAS_CONTROL_PLANE_NODE_BOOTSTRAP_TOKEN")
	if err != nil {
		return nil, err
	}

	davAuthSecret, err := requiredEnv("BETTERNAS_DAV_AUTH_SECRET")
	if err != nil {
		return nil, err
	}

	davCredentialTTL, err := parseRequiredDurationEnv("BETTERNAS_DAV_CREDENTIAL_TTL")
	if err != nil {
		return nil, err
	}

	var sessionTTL time.Duration
	rawSessionTTL := strings.TrimSpace(env("BETTERNAS_SESSION_TTL", "720h"))
	if rawSessionTTL != "" {
		sessionTTL, err = time.ParseDuration(rawSessionTTL)
		if err != nil {
			return nil, err
		}
	}

	return newApp(
		appConfig{
			version:             env("BETTERNAS_VERSION", "0.1.0-dev"),
			nextcloudBaseURL:    env("NEXTCLOUD_BASE_URL", ""),
			statePath:           env("BETTERNAS_CONTROL_PLANE_STATE_PATH", ".state/control-plane/state.json"),
			dbPath:              env("BETTERNAS_CONTROL_PLANE_DB_PATH", ""),
			clientToken:         env("BETTERNAS_CONTROL_PLANE_CLIENT_TOKEN", ""),
			nodeBootstrapToken:  nodeBootstrapToken,
			davAuthSecret:       davAuthSecret,
			davCredentialTTL:    davCredentialTTL,
			sessionTTL:          sessionTTL,
			registrationEnabled: env("BETTERNAS_REGISTRATION_ENABLED", "true") == "true",
			corsOrigin:          env("BETTERNAS_CORS_ORIGIN", ""),
		},
		startedAt,
	)
}
