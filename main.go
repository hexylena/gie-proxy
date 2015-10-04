package main

import (
	"flag"
	"log"
	"net/http"
)

var addr *string = flag.String("listen", "0.0.0.0:8080", "address to listen on")
var path *string = flag.String("listen_path", "/galaxy/gie_proxy", "path to listen on (for cookies)")
var cookie_name *string = flag.String("cookie_name", "galaxysession", "cookie name")
var session_map *string = flag.String("storage", "./session_map.json", "Session map file. Used to (re)store route lists across restarts")

type Frontend struct {
	Addr string
	Path string
}

type RequestHandler struct {
	Transport    *http.Transport
	RouteMapping *RouteMappings
	Frontend     *Frontend
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Printf("incoming request: %# v", pretty.Formatter(*r))
	r.URL.Scheme = "http"

	// Add x-forwarded-for header
	addForwardedFor(r)

	// If they request a url shorter than the request URI, we know it's bad.
	// E.g. requesting /x when the proxy prefix is /blah is bad
	if len(h.Frontend.Path) > len(r.RequestURI) {
		http.Error(w, "unknown backend", http.StatusBadRequest)
		return
	}

	// Get their cookie
	cookie, err := r.Cookie(*cookie_name)
	if err != nil {
		http.Error(w, "unknown auth cookie", http.StatusUnauthorized)
		return
	}
	//log.Printf("%#v", pretty.Formatter(cookie))

	// Pick out route
	route, err := h.RouteMapping.FindRoute(
		r.RequestURI[len(h.Frontend.Path):], // Strip proxy prefix from path
		cookie.Value,
	)
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "unknown backend", http.StatusServiceUnavailable)
		return
	}

	// Reset request URI
	r.RequestURI = ""
	r.URL.Host = route.BackendAddr
	// Strip frontend's path out
	r.URL.Path = r.URL.Path[len(h.Frontend.Path):]

	// Upgrade the websocket if need be
	upgrade_websocket := shouldUpgradeWebsocket(r)

	//log.Printf("proxied request: %# v", pretty.Formatter(*r))
	if upgrade_websocket {
		plumbWebsocket(w, r)
	} else {
		plumbHttp(h, w, r)
	}
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

	// The slash route handles ALL requests by passing to the request_handler
	// object
	mux.Handle("/", request_handler)
	// Here we then launch the server from mux
	srv := &http.Server{Handler: mux, Addr: f.Addr}

	// Start
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("Starting frontend failed: %v", err)
	}
}
