package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authnz
	recvAPIKey := r.URL.Query().Get("api_key")
	// If it doesn't match what we expect, kick
	if recvAPIKey != h.Frontend.APIKey {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}
	log.Info("Received %s request to the API", r.Method)

	// Request Processing
	if r.Method == "GET" {
		// Get a list of routes
		renderViewData(h, w, r)
	} else if r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		route := new(Route)
		err := decoder.Decode(&route)
		if err != nil {
			log.Error(fmt.Sprintf("Error unmarshalling %s", err))
			http.Error(w, "Invalid Route data", http.StatusBadRequest)
			return
		}

		// Seems like this should automatically be a decode exception?
		if route.FrontendPath == "" || route.BackendAddr == "" || route.AuthorizedCookie == "" {
			log.Info("An invalid route was attempted [%s %s %s %s]", route.FrontendPath, route.BackendAddr, route.AuthorizedCookie, route.ContainerIds)
			http.Error(w, "Invalid Route data", http.StatusBadRequest)
			return
		}

		// Create a new route
		h.RouteMapping.AddRoute(
			route.FrontendPath,
			route.BackendAddr,
			route.AuthorizedCookie,
			route.ContainerIds,
		)

		renderViewData(h, w, r)
	}
}

func renderViewData(h *apiHandler, w http.ResponseWriter, r *http.Request) {
	jsonRoutes, err := json.MarshalIndent(h.RouteMapping.Routes, "", "    ")
	if err != nil {
		http.Error(w, "Data encoding error", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, string(jsonRoutes))
}
