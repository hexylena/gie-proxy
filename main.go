package main

import (
	"flag"
	//goconf "github.com/akrennmair/goconf"
	"log"
	"net/http"
	//"os"
)

type Frontend struct {
	Addr string
}

type RequestHandler struct {
	Transport    *http.Transport
	RouteMapping *RouteMappings
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("incoming request: %#v", *r)
	r.RequestURI = ""
	r.URL.Scheme = "http"

	r = addForwardedFor(r)

	// Pick out route
	route, err := h.RouteMapping.FindRoute("", "")
	if err != nil {
		log.Printf("Error: %s", err)
	}
	r.URL.Host = route.BackendUrl

	upgrade_websocket := shouldUpgradeWebsocket(r)

	if upgrade_websocket {
		plumbWebsocket(w, r)
	} else {
		plumbHttp(h, w, r)
	}
}

func main() {
	var addr *string = flag.String("listen", "0.0.0.0:8080", "address to listen on")
	//var cfgfile *string = flag.String("config", "", "configuration file")
	flag.Parse()

	r1 := &Route{
		FrontendPath:     "/hello",
		BackendUrl:       "http://localhost:8080",
		AuthorizedCookie: "test",
	}

	rm1 := &RouteMappings{
		Routes: []*Route{r1},
	}

	f := &Frontend{
		Addr: *addr,
	}

	//cfg, err := goconf.ReadConfigFile(*cfgfile)
	//if err != nil {
	//log.Printf("opening %s failed: %v", *cfgfile, err)
	//os.Exit(1)
	//}

	log.Printf("Starting frontend ...")
	f.Start(rm1)
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
	}
	//if logger != nil {
	//request_handler = NewRequestLogger(request_handler, *logger)
	//}

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
