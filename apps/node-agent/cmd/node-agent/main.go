package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	port := env("PORT", "8090")
	app, err := newAppFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	controlPlaneCtx, stopControlPlane := context.WithCancel(context.Background())
	defer stopControlPlane()
	if app.controlPlaneEnabled() {
		app.startControlPlaneLoop(controlPlaneCtx)
	}

	signalContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	go func() {
		<-signalContext.Done()
		stopControlPlane()
	}()

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
