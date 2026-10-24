package main

import (
	"errors"
	"log"
	"net/http"
	"time"
)

func main() {
	addr := ":" + port()

	// The handlers get their manga data from the real AniList client.
	a := &app{source: newAniListClient()}

	// We build an http.Server ourselves instead of calling http.ListenAndServe
	// directly, because the shortcut has no timeouts. Without them, a slow or
	// malicious client can hold a connection open forever.
	server := &http.Server{
		Addr:    addr,
		Handler: newRouter(a),

		// Limit how long a client may take to send its request headers.
		ReadHeaderTimeout: 5 * time.Second,
		// Limit how long it may take to send the whole request. Ours are tiny GETs.
		ReadTimeout: 10 * time.Second,
		// Limit how long we spend writing a response. This must stay longer
		// than the timeout we will put on AniList calls (step 4), or slow
		// AniList answers would be cut off before we could send them.
		WriteTimeout: 30 * time.Second,
		// Limit how long an idle keep-alive connection stays open.
		IdleTimeout: 60 * time.Second,
	}

	log.Printf("%s listening on http://localhost%s", appName, addr)

	// ListenAndServe only returns when something goes wrong (or on shutdown,
	// which we'll add in step 7, signalled by http.ErrServerClosed).
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}
