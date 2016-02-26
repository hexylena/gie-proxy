package main

import (
	"net/http"
	"strings"
)

func connectRoute(h *requestHandler, w http.ResponseWriter, r *http.Request) error {
	var err error
	if shouldUpgradeWebsocket(r) {
		err = plumbWebsocket(w, r)
	} else {
		err = plumbHTTP(h, w, r)
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
	cookie, err := r.Cookie(h.Frontend.CookieName)
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
	// Now that we have a route, update when we last saw it.
	route.Seen()

	// Reset request URI
	r.RequestURI = ""
	r.URL.Host = route.BackendAddr
	// Strip frontend's path out
	//r.URL.Path = r.URL.Path[len(h.Frontend.Path):]

	connectErr := connectRoute(h, w, r)

	// If the backend is dead, remove it.
	// The next request from the user will be better behaved.
	if connectErr != nil && connectErr.Error() == "dead-backend" {
		h.RouteMapping.RemoveRoute(route)
	}

	notify := w.(http.CloseNotifier)
	go func(notify http.CloseNotifier, route *Route) {
		// TODO: add a timer?
		select {
		// This signals that the HTTP conncetion closed
		case <-notify.CloseNotify():
			// TODO: Many HTTP connections come through, need to make sure ALL
			// are closed.
			route.Expired = true
			return
		}
	}(notify, route)
}
