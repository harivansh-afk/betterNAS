package main

import (
	"errors"
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
	var sessionTTL time.Duration
	rawSessionTTL := strings.TrimSpace(env("BETTERNAS_SESSION_TTL", "720h"))
	if rawSessionTTL != "" {
		parsedSessionTTL, err := time.ParseDuration(rawSessionTTL)
		if err != nil {
			return nil, err
		}
		sessionTTL = parsedSessionTTL
	}

	app, err := newApp(
		appConfig{
			version:             env("BETTERNAS_VERSION", "0.1.0-dev"),
			nextcloudBaseURL:    env("NEXTCLOUD_BASE_URL", ""),
			statePath:           env("BETTERNAS_CONTROL_PLANE_STATE_PATH", ".state/control-plane/state.json"),
			dbPath:              env("BETTERNAS_CONTROL_PLANE_DB_PATH", ".state/control-plane/betternas.db"),
			sessionTTL:          sessionTTL,
			registrationEnabled: env("BETTERNAS_REGISTRATION_ENABLED", "true") == "true",
			corsOrigin:          env("BETTERNAS_CORS_ORIGIN", ""),
		},
		startedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := seedDefaultUserFromEnv(app); err != nil {
		return nil, err
	}

	return app, nil
}

func seedDefaultUserFromEnv(app *app) error {
	username := strings.TrimSpace(env("BETTERNAS_USERNAME", ""))
	password := env("BETTERNAS_PASSWORD", "")
	if username == "" || password == "" {
		return nil
	}

	if _, err := app.store.createUser(username, password); err != nil {
		if errors.Is(err, errUsernameTaken) {
			_, authErr := app.store.authenticateUser(username, password)
			return authErr
		}
		return err
	}

	return nil
}
