package main

import (
	"os"
	"strings"
)

// appName is the app's name. It is the only place the backend spells it
// out (the frontend has its own, APP_NAME in web/static/js/config.js).
const appName = "sankomanga"

// defaultPort is used when the PORT environment variable is not set.
const defaultPort = "8080"

// trustedProxyHeader returns the name of the header that holds each
// visitor's real address, set with the TRUSTED_PROXY_HEADER environment
// variable (for example "X-Real-IP" on Railway). Empty when not set.
//
// Behind a hosting provider's proxy, every request reaches us from the
// proxy, so the connection's own address is the proxy's, not the
// visitor's; the proxy passes the visitor's address in a header instead.
// Only set this when the app really runs behind a proxy that writes that
// header itself: anyone can send a made-up header straight to the app,
// which would let them pose as many different visitors.
func trustedProxyHeader() string {
	return os.Getenv("TRUSTED_PROXY_HEADER")
}

// siteURL returns the site's public address, like
// "https://sankomanga.up.railway.app", with no "/" at the end. Link
// previews need it: the apps that show them want full addresses, not
// ones relative to the page.
//
// It's the SITE_URL environment variable when set (use that with your
// own domain). Otherwise, on Railway, it's the address Railway gave the
// app (RAILWAY_PUBLIC_DOMAIN, which Railway sets by itself). Otherwise
// (running locally) it's empty, and those addresses simply start with
// "/", which is fine on your own computer.
func siteURL() string {
	if url := os.Getenv("SITE_URL"); url != "" {
		return strings.TrimSuffix(url, "/")
	}
	if domain := os.Getenv("RAILWAY_PUBLIC_DOMAIN"); domain != "" {
		return "https://" + domain
	}
	return ""
}

// port returns the port the server should listen on.
// Hosting platforms usually tell an app which port to use through the PORT
// environment variable, so we read that first and fall back to a default.
func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return defaultPort
}
