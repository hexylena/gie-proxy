package main

import (
	"net/http"
	"strings"
)

func connectRoute(h *requestHandler, w http.ResponseWriter, r *http.Request, route **Route) error {
	var err error
	if shouldUpgradeWebsocket(r) {
		err = plumbWebsocket(w, r, route)
	} else {
		err = plumbHTTP(h, w, r, route)
	}
	return err
}

func (h *requestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.URL.Scheme = "http"
	// Add x-forwarded-for header
	addForwardedFor(r)

	// Their requested URL must agree with our prefix
	if !strings.HasPrefix(r.RequestURI, h.Frontend.Path) {
		log.Warning("Bad request %s", r.RequestURI)
		http.Error(w, "unknown backend", http.StatusBadRequest)
		return
	}

	// Get their cookie
	cookie, err := r.Cookie(h.RouteMapping.AuthCookieName)
	if err != nil {
		log.Warning("Request lacked cookie")
		http.Error(w, "unknown auth cookie", http.StatusUnauthorized)
		return
	}

	// Find our route
	route, err := h.RouteMapping.FindRoute(
		r.RequestURI[len(h.Frontend.Path):], // Strip proxy prefix from path
		cookie.Value,
	)
	if err != nil && err.Error() == "Could not find route" {
		log.Warning("Could not find route")
		http.Error(w, "unknown backend", http.StatusBadRequest)
	}
	// Reset request URI
	r.RequestURI = ""
	r.URL.Host = route.BackendAddr
	// Strip frontend's path out
	//r.URL.Path = r.URL.Path[len(h.Frontend.Path):]

	// Here we do the plumbing and connect up goroutines to automatically
	// copy between two endpoints
	connectErr := connectRoute(h, w, r, &route)

	// If the backend is dead, remove it.
	// The next request from the user will be better behaved.
	if connectErr != nil && connectErr.Error() == "dead-backend" {
		h.RouteMapping.RemoveRoute(route)
	}
}
