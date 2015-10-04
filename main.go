package main

import (
	"flag"
	"fmt"
	goconf "github.com/akrennmair/goconf"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var addr *string = flag.String("listen", "0.0.0.0:8080", "address to listen on")

type Backend struct {
	Name          string
	ConnectString string
}

type Frontend struct {
	Name         string
	BindString   string
	HTTPS        bool
	AddForwarded bool
	Hosts        []string
	Backends     []string
	//AddHeader    struct { Key string; Value string }
	KeyFile  string
	CertFile string
}

type RequestHandler struct {
	Transport    *http.Transport
	Frontend     *Frontend
	HostBackends map[string]chan *Backend
	Backends     chan *Backend
}

type RequestHandler2 struct {
	Transport *http.Transport
	Frontend  *Frontend
	Backend   *Backend
	Routes    []Route
}

func (h *RequestHandler2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Printf("incoming request: %#v", *r)
	r.RequestURI = ""
	r.URL.Scheme = "http"

	// Add x-forwarded-for header
	if h.Frontend.AddForwarded {
		remote_addr := r.RemoteAddr
		idx := strings.LastIndex(remote_addr, ":")
		if idx != -1 {
			remote_addr = remote_addr[0:idx]
			if remote_addr[0] == '[' && remote_addr[len(remote_addr)-1] == ']' {
				remote_addr = remote_addr[1 : len(remote_addr)-1]
			}
		}
		r.Header.Add("X-Forwarded-For", remote_addr)
	}

	// Configure routing
	if len(h.Frontend.Hosts) == 0 {
		backend := <-h.Backends
		r.URL.Host = backend.ConnectString
		h.Backends <- backend
	} else {
		backend_list := h.HostBackends[r.Host]
		if backend_list == nil {
			if len(h.Frontend.Backends) == 0 {
				http.Error(w, "no suitable backend found for request", http.StatusServiceUnavailable)
				return
			} else {
				backend := <-h.Backends
				r.URL.Host = backend.ConnectString
				h.Backends <- backend
			}
		} else {
			backend := <-backend_list
			r.URL.Host = backend.ConnectString
			backend_list <- backend
		}
	}

	upgrade_websocket := shouldUpgradeWebsocket(r)

	if upgrade_websocket {
		plumbWebsocket(w, r)
	} else {
		plumbHttp(h, w, r)
	}
}

func main() {
	var cfgfile *string = flag.String("config", "", "configuration file")
	backends := make(map[string]*Backend)
	hosts := make(map[string][]*Backend)
	frontends := make(map[string]*Frontend)
	flag.Parse()
	cfg, err := goconf.ReadConfigFile(*cfgfile)
	if err != nil {
		log.Printf("opening %s failed: %v", *cfgfile, err)
		os.Exit(1)
	}
	var access_f io.WriteCloser
	accesslog_file, err := cfg.GetString("global", "accesslog")
	if err == nil {
		access_f, err = os.OpenFile(accesslog_file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
		if err == nil {
			defer access_f.Close()
		} else {
			log.Printf("Opening access log %s failed: %v", accesslog_file, err)
		}
	}
	// first, extract backends
	for _, section := range cfg.GetSections() {
		if strings.HasPrefix(section, "backend ") {
			tokens := strings.Split(section, " ")
			if len(tokens) < 2 {
				log.Printf("backend section has no name, ignoring.")
				continue
			}
			connect_str, _ := cfg.GetString(section, "connect")
			if connect_str == "" {
				log.Printf("empty connect string for backend %s, ignoring.", tokens[1])
				continue
			}
			b := &Backend{Name: tokens[1], ConnectString: connect_str}
			backends[b.Name] = b
		}
	}
	// then extract hosts
	for _, section := range cfg.GetSections() {
		if strings.HasPrefix(section, "host ") {
			tokens := strings.Split(section, " ")
			if len(tokens) < 2 {
				log.Printf("host section has no name, ignoring.")
				continue
			}
			backends_str, _ := cfg.GetString(section, "backends")
			backends_list := strings.Split(backends_str, " ")
			if len(backends_list) == 0 {
				log.Printf("host %s has no backends, ignoring.", tokens[1])
				continue
			}
			for _, host := range tokens[1:] {
				backends_for_host := []*Backend{}
				for _, backend := range backends_list {
					b := backends[backend]
					if b == nil {
						log.Printf("backend %s doesn't exist, ignoring.", backend)
					}
					backends_for_host = append(backends_for_host, b)
				}
				hosts[host] = backends_for_host
			}
		}
	}
	// and finally, extract frontends
	for _, section := range cfg.GetSections() {
		if strings.HasPrefix(section, "frontend ") {
			tokens := strings.Split(section, " ")
			if len(tokens) < 2 {
				log.Printf("frontend section has no name, ignoring.")
				continue
			}
			frontend_name := tokens[1]
			frontend := &Frontend{}
			frontend.Name = frontend_name
			frontend.BindString, err = cfg.GetString(section, "bind")
			if err != nil {
				log.Printf("error while getting [%s]bind: %v, ignoring.", section, err)
				continue
			}
			if frontend.BindString == "" {
				log.Printf("frontend %s has no bind argument, ignoring.", frontend_name)
				continue
			}
			frontend.HTTPS, err = cfg.GetBool(section, "https")
			if err != nil {
				frontend.HTTPS = false
			}
			if frontend.HTTPS {
				frontend.KeyFile, err = cfg.GetString(section, "keyfile")
				if err != nil {
					log.Printf("error while getting[%s]keyfile: %v, ignoring.", section, err)
					continue
				}
				if frontend.KeyFile == "" {
					log.Printf("frontend %s has HTTPS enabled but no keyfile, ignoring.", frontend_name)
					continue
				}
				frontend.CertFile, err = cfg.GetString(section, "certfile")
				if err != nil {
					log.Printf("error while getting[%s]certfile: %v, ignoring.", section, err)
					continue
				}
				if frontend.CertFile == "" {
					log.Printf("frontend %s has HTTPS enabled but no certfile, ignoring.", frontend_name)
					continue
				}
			}
			frontend_hosts, err := cfg.GetString(section, "hosts")
			if err == nil && frontend_hosts != "" {
				frontend.Hosts = strings.Split(frontend_hosts, " ")
			}
			frontend_backends, err := cfg.GetString(section, "backends")
			if err == nil && frontend_backends != "" {
				frontend.Backends = strings.Split(frontend_backends, " ")
			}
			frontend.AddForwarded, _ = cfg.GetBool(section, "add-x-forwarded-for")
			if len(frontend.Backends) == 0 && len(frontend.Hosts) == 0 {
				log.Printf("frontend %s has neither backends nor hosts configured, ignoring.", frontend_name)
				continue
			}
			frontends[frontend_name] = frontend
		}
	}
	count := 0
	exit_chan := make(chan int)
	for name, frontend := range frontends {
		log.Printf("Starting frontend %s...", name)
		go func(fe *Frontend, name string) {
			var accesslogger *log.Logger
			if access_f != nil {
				accesslogger = log.New(access_f, "frontend:"+name+" ", log.Ldate|log.Ltime|log.Lmicroseconds)
			} else {
				log.Printf("Not creating logger for frontend %s", name)
			}
			fe.Start(hosts, backends, accesslogger)
			exit_chan <- 1
		}(frontend, name)
		count++
	}
	// this shouldn't return
	for i := 0; i < count; i++ {
		<-exit_chan
	}
}

func (f *Frontend) Start(hosts map[string][]*Backend, backends map[string]*Backend, logger *log.Logger) {
	mux := http.NewServeMux()

	// Main request handler, processes every incoming request
	var request_handler http.Handler = &RequestHandler2{
		Transport: &http.Transport{
			DisableKeepAlives:  false,
			DisableCompression: false,
		},
		Routes: route_list,
	}
	if logger != nil {
		request_handler = NewRequestLogger(request_handler, *logger)
	}

	// The slash route handles ALL requests by passing to the request_handler
	// object
	mux.Handle("/", request_handler)
	// Here we then launch the server from mux
	srv := &http.Server{Handler: mux, Addr: *addr}

	// Start
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("Starting frontend %s failed: %v", f.Name, err)
	}
}
