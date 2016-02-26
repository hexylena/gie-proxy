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

	var apiHandler http.Handler = &apiHandler{
		Transport: &http.Transport{
			DisableKeepAlives:  false,
			DisableCompression: false,
		},
		RouteMapping: rm,
		Frontend:     f,
	}

	// The API route only handles requests with an absolute path of /api. Given
	// that the frontend.Path is usually set, this means it will be completely
	// unavailable to external access, and only available from localhost. This
	// is a win for security, as only Galaxy should be talking to the API
	mux.Handle("/api", apiHandler)
	// The slash route handles ALL requests by passing to the request_handler
	// object
	mux.Handle("/", requestHandler)
	// Here we then launch the server from mux
	srv := &http.Server{Handler: mux, Addr: f.Addr}
	// Start
	log.Info("Listening on %s %s", f.Addr, f.Path)
	if err := srv.ListenAndServe(); err != nil {
		log.Critical("Starting frontend failed: %v", err)
	}
}
