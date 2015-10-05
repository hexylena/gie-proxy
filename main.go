package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"
	//"github.com/kr/pretty"
	"log"
	"net/http"
	"strings"
)

var addr = flag.String("listen", "0.0.0.0:8080", "address to listen on")
var path = flag.String("listen_path", "/galaxy/gie_proxy", "path to listen on (for cookies)")
var cookieName = flag.String("cookie_name", "galaxysession", "cookie name")
var sessionMap = flag.String("storage", "./sessionMap.xml", "Session map file. Used to (re)store route lists across restarts")
var apiKey = flag.String("api_key", "THE_DEFAULT_IS_NOT_SECURE", "Key to access the API")
var noAccessThreshold = flag.Int("noaccess", 60, "Length of time a proxy route must be unused before automatically being removed")
var dockerEndpoint = flag.String("docker", "unix:///var/run/docker.sock", "Endpoint at which we can access docker. No TLS Support yet")

type frontend struct {
	Addr string
	Path string
}

type requestHandler struct {
	Transport    *http.Transport
	RouteMapping *RouteMapping
	Frontend     *frontend
}

type apiHandler struct {
	Transport    *http.Transport
	RouteMapping *RouteMapping
}

func (h *requestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.URL.Scheme = "http"
	// Add x-forwarded-for header
	addForwardedFor(r)

	// Their requested URL must agree with our prefix
	if !strings.HasPrefix(r.RequestURI, h.Frontend.Path) {
		http.Error(w, "unknown backend", http.StatusBadRequest)
		return
	}

	// Get their cookie
	cookie, err := r.Cookie(*cookieName)
	if err != nil {
		http.Error(w, "unknown auth cookie", http.StatusUnauthorized)
		return
	}

	// Find our route
	route, err := h.RouteMapping.FindRoute(
		r.RequestURI[len(h.Frontend.Path):], // Strip proxy prefix from path
		cookie.Value,
	)
	if err != nil && err.Error() == "Could not find route" {
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
}

func connectRoute(h *requestHandler, w http.ResponseWriter, r *http.Request) error {
	var err error
	if shouldUpgradeWebsocket(r) {
		err = plumbWebsocket(w, r)
	} else {
		err = plumbHTTP(h, w, r)
	}
	return err
}

func main() {
	flag.Parse()
	// Load up route mapping
	rm := NewRouteMapping(sessionMap, dockerEndpoint)
	rm.AuthCookieName = *cookieName
	rm.NoAccessThreshold = time.Second * time.Duration(*noAccessThreshold)
	rm.Save()
	// Build the frontend
	f := &frontend{
		Addr: *addr,
		Path: *path,
	}
	// Start our proxy
	log.Printf("Starting frontend ...")
	f.Start(rm)
}

func renderViewData(h *apiHandler, w http.ResponseWriter, r *http.Request) {
	jsonRoutes, err := json.MarshalIndent(h.RouteMapping.Routes, "", "    ")
	if err != nil {
		http.Error(w, "Data encoding error", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, string(jsonRoutes))
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authnz
	recvAPIKey := r.URL.Query().Get("api_key")
	// If it doesn't match what we expect, kick
	if recvAPIKey != *apiKey {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	// Request Processing
	if r.Method == "GET" {
		// Get a list of routes
		renderViewData(h, w, r)
	} else if r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		route := new(Route)
		err := decoder.Decode(&route)
		if err != nil {
			http.Error(w, "Invalid Route data", http.StatusBadRequest)
			return
		}

		// Seems like this should automatically be a decode exception?
		if route.FrontendPath == "" || route.BackendAddr == "" || route.AuthorizedCookie == "" || len(route.ContainerIds) == 0 {
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
	}

	// The slash route handles ALL requests by passing to the request_handler
	// object
	mux.Handle("/api", apiHandler)
	mux.Handle("/", requestHandler)
	// Here we then launch the server from mux
	srv := &http.Server{Handler: mux, Addr: f.Addr}

	// Start
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("Starting frontend failed: %v", err)
	}
}
