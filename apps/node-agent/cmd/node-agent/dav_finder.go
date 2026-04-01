package main

import (
	"context"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/webdav"
)

// finderCompatible wraps a webdav.Handler to work around macOS Finder quirks.
//
// Finder sends GET requests to WebDAV collection (directory) URLs and expects
// a 200 response. The standard Go webdav.Handler returns 405 Method Not Allowed
// for GET on directories, which causes Finder to refuse the mount.
//
// This wrapper intercepts GET and HEAD requests on directories and returns a
// minimal 200 response so Finder proceeds with the WebDAV protocol.
func finderCompatible(dav *webdav.Handler, fs webdav.FileSystem, prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			dav.ServeHTTP(w, r)
			return
		}

		// Strip the prefix to get the filesystem-relative path.
		fsPath := strings.TrimPrefix(r.URL.Path, prefix)
		if fsPath == "" {
			fsPath = "/"
		}

		f, err := fs.OpenFile(context.Background(), fsPath, os.O_RDONLY, 0)
		if err != nil {
			// Not found or other error: let the regular handler deal with it.
			dav.ServeHTTP(w, r)
			return
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			dav.ServeHTTP(w, r)
			return
		}

		if !info.IsDir() {
			// Regular file: let the standard handler serve it.
			dav.ServeHTTP(w, r)
			return
		}

		// Directory GET: return a minimal 200 so Finder is satisfied.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte("<!DOCTYPE html><html><body><p>betterNAS WebDAV</p></body></html>\n"))
		}
	})
}
