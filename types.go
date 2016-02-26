package main

import (
	docker "github.com/fsouza/go-dockerclient"
	"net/http"
	"time"
)

type frontend struct {
	Addr       string
	Path       string
	APIKey     string
	CookieName string
}

type requestHandler struct {
	Transport    *http.Transport
	RouteMapping *RouteMapping
	Frontend     *frontend
}

type apiHandler struct {
	Transport    *http.Transport
	RouteMapping *RouteMapping
	Frontend     *frontend
}

// Route represents connection information to wire up a frontend request to a
// backend
type Route struct {
	FrontendPath     string
	BackendAddr      string
	AuthorizedCookie string
	LastSeen         time.Time
	ContainerIds     []string `xml:"ContainerIds"`
	ID               int64
}

// RouteMapping represents essentially the server state, including all
// routes and metadata necessary to re-launch in an identical state.
type RouteMapping struct {
	Routes            []Route `xml:"Routes>Route"`
	AuthCookieName    string
	Storage           string
	NoAccessThreshold time.Duration
	DockerEndpoint    string
	client            *docker.Client
	CleanInterval     time.Duration
}
