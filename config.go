package main

import "os"

// appName is a placeholder until the real name is decided.
// It is the only place the backend spells out the app's name.
const appName = "manga-app"

// defaultPort is used when the PORT environment variable is not set.
const defaultPort = "8080"

// port returns the port the server should listen on.
// Hosting platforms usually tell an app which port to use through the PORT
// environment variable, so we read that first and fall back to a default.
func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return defaultPort
}
