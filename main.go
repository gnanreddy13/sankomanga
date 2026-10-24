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

// shutdownTimeout is how long we let in-flight requests finish after being
// asked to stop. A request can be waiting on AniList for up to
// anilistTimeout (10s), so we allow that plus a few seconds of margin.
const shutdownTimeout = 15 * time.Second

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
		// than anilistTimeout, or slow AniList answers would be cut off
		// before we could send them.
		WriteTimeout: 30 * time.Second,
		// Limit how long an idle keep-alive connection stays open.
		IdleTimeout: 60 * time.Second,
	}

	// stopSignal is cancelled when the process is asked to stop: Ctrl-C in
	// a terminal sends SIGINT, and hosting platforms send SIGTERM.
	stopSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run the server in the background (a goroutine), so that main is free
	// to wait for either a stop signal or the server failing.
	serverFailed := make(chan error, 1)
	go func() {
		log.Printf("%s listening on http://localhost%s", appName, addr)
		serverFailed <- server.ListenAndServe()
	}()

	select {
	case err := <-serverFailed:
		// The server couldn't start or crashed, e.g. the port is already in use.
		log.Fatalf("server failed: %v", err)
	case <-stopSignal.Done():
		// Go back to normal signal handling, so pressing Ctrl-C a second
		// time stops the program immediately instead of waiting.
		stop()
	}

	// Shutdown stops accepting new requests, lets in-flight ones finish,
	// and then returns. If they take longer than shutdownTimeout, it gives
	// up and we exit anyway.
	log.Printf("shutting down, waiting up to %s for requests to finish", shutdownTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown did not finish cleanly: %v", err)
		return
	}
	log.Printf("stopped")
}
