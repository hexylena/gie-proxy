package main

import (
	"encoding/json"
	"flag"
	"fmt"
	//"github.com/kr/pretty"
	"log"
	"net/http"
	"strings"
)

var addr *string = flag.String("listen", "0.0.0.0:8080", "address to listen on")
var path *string = flag.String("listen_path", "/galaxy/gie_proxy", "path to listen on (for cookies)")
var cookie_name *string = flag.String("cookie_name", "galaxysession", "cookie name")
var session_map *string = flag.String("storage", "./session_map.xml", "Session map file. Used to (re)store route lists across restarts")
var api_key *string = flag.String("api_key", "THE_DEFAULT_IS_NOT_SECURE", "Key to access the API")

type Frontend struct {
	Addr string
	Path string
}

type RequestHandler struct {
	Transport    *http.Transport
	RouteMapping *RouteMappings
	Frontend     *Frontend
}

type ApiHandler struct {
	Transport    *http.Transport
	RouteMapping *RouteMappings
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.URL.Scheme = "http"
	// Add x-forwarded-for header
	addForwardedFor(r)

	// Their requested URL must agree with our prefix
	if !strings.HasPrefix(r.RequestURI, h.Frontend.Path) {
		http.Error(w, "unknown backend", http.StatusBadRequest)
		return
	}

	// Get their cookie
	cookie, err := r.Cookie(*cookie_name)
	if err != nil {
		http.Error(w, "unknown auth cookie", http.StatusUnauthorized)
		return
	}

	log.Printf("Request for %s", r.RequestURI)

	// Find our route
	route, err := h.RouteMapping.FindRoute(
		r.RequestURI[len(h.Frontend.Path):], // Strip proxy prefix from path
		cookie.Value,
	)
	if err.Error() == "Could not find route" {
		http.Error(w, "unknown backend", http.StatusBadRequest)
	}

	// Reset request URI
	r.RequestURI = ""
	r.URL.Host = route.BackendAddr
	// Strip frontend's path out
	//r.URL.Path = r.URL.Path[len(h.Frontend.Path):]

	connect_err := connectRoute(h, w, r)

	// If the backend is dead, remove it.
	// The next request from the user will be better behaved.
	if connect_err.Error() == "dead-backend" {
		h.RouteMapping.RemoveRoute(route)
	}
}

func connectRoute(h *RequestHandler, w http.ResponseWriter, r *http.Request) error {
	var err error
	if shouldUpgradeWebsocket(r) {
		err = plumbWebsocket(w, r)
	} else {
		err = plumbHttp(h, w, r)
	}
	return err
}

func main() {
	flag.Parse()
	// Load up route mapping
	rm := NewRouteMapping(session_map)
	// Build the frontend
	f := &Frontend{
		Addr: *addr,
		Path: *path,
	}
	// Start our proxy
	log.Printf("Starting frontend ...")
	f.Start(rm)
}

func (h *ApiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authnz
	recv_api_key := r.URL.Query().Get("api_key")
	// If it doesn't match what we expect, kick
	if recv_api_key != *api_key {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	// Request Processing
	if r.Method == "GET" {
		// Get a list of routes
		jsonRoutes, err := json.MarshalIndent(h.RouteMapping.Routes, "", "    ")
		if err != nil {
			http.Error(w, "Data encoding error", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, string(jsonRoutes))
	} else if r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		route := new(Route)
		err := decoder.Decode(&route)
		if err != nil {
			http.Error(w, "Invalid Route data", http.StatusBadRequest)
			return
		}

		// Seems like this should automatically be a decode exception?
		if route.FrontendPath == "" || route.BackendAddr == "" || route.AuthorizedCookie == "" {
			http.Error(w, "Invalid Route data", http.StatusBadRequest)
			return
		}

		// Create a new route
		h.RouteMapping.AddRoute(
			route.FrontendPath,
			route.BackendAddr,
			route.AuthorizedCookie,
		)

		// Then display all of them. TODO: refactor this?
		jsonRoutes, err := json.MarshalIndent(h.RouteMapping.Routes, "", "    ")
		if err != nil {
			http.Error(w, "Data encoding error", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, string(jsonRoutes))
	}
}

func (f *Frontend) Start(rm *RouteMappings) {
	mux := http.NewServeMux()

	// Main request handler, processes every incoming request
	var request_handler http.Handler = &RequestHandler{
		Transport: &http.Transport{
			DisableKeepAlives:  false,
			DisableCompression: false,
		},
		RouteMapping: rm,
		Frontend:     f,
	}

	var api_handler http.Handler = &ApiHandler{
		Transport: &http.Transport{
			DisableKeepAlives:  false,
			DisableCompression: false,
		},
		RouteMapping: rm,
	}

	// The slash route handles ALL requests by passing to the request_handler
	// object
	mux.Handle("/api", api_handler)
	mux.Handle("/", request_handler)
	// Here we then launch the server from mux
	srv := &http.Server{Handler: mux, Addr: f.Addr}

	// Start
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("Starting frontend failed: %v", err)
	}
}
