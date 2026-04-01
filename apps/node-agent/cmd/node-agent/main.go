package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := env("PORT", "8090")
	app, err := newAppFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           app.handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("betterNAS node agent listening on :%s", port)
	log.Fatal(server.ListenAndServe())
}

func env(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}
