package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/webdav"
)

func main() {
	port := env("PORT", "8090")
	exportPath := env("BETTERNAS_EXPORT_PATH", ".")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})

	dav := &webdav.Handler{
		Prefix:     "/dav",
		FileSystem: webdav.Dir(exportPath),
		LockSystem: webdav.NewMemLS(),
	}
	mux.Handle("/dav/", dav)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("betterNAS node agent serving %s on :%s", exportPath, port)
	log.Fatal(server.ListenAndServe())
}

func env(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}
