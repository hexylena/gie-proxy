package main

import (
	"net/http"
)

func (f *frontend) Start(rm *RouteMapping) {
	mux := http.NewServeMux()

	// Main request handler, processes every incoming request
	var requestHandler http.Handler = &requestHandler{
		Transport: &http.Transport{
			DisableKeepAlives:  false,
			DisableCompression: false,
		},
		RouteMapping: rm,
		Frontend:     f,
	}
	rm.RegisterCleaner()

	var apiHandler http.Handler = &apiHandler{
		Transport: &http.Transport{
			DisableKeepAlives:  false,
			DisableCompression: false,
		},
		RouteMapping: rm,
		Frontend:     f,
	}

	// The slash route handles ALL requests by passing to the request_handler
	// object
	mux.Handle("/api", apiHandler)
	mux.Handle("/", requestHandler)
	// Here we then launch the server from mux
	srv := &http.Server{Handler: mux, Addr: f.Addr}

	// Start
	log.Info("Listening on %s %s", f.Addr, f.Path)
	if err := srv.ListenAndServe(); err != nil {
		log.Critical("Starting frontend failed: %v", err)
	}
}
