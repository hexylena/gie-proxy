package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"time"
)

// Route represents connection information to wire up a frontend request to a
// backend
type Route struct {
	FrontendPath     string
	BackendAddr      string
	AuthorizedCookie string
	LastSeen         time.Time
	ContainerIds     []string `xml:ContainerIds`
}

// IsAuthorized checks if a user's cookie is valid for a given route object.
func (r *Route) IsAuthorized(cookie string) bool {
	return r.AuthorizedCookie == cookie
}

// Seen notifies the route object that it was seen recently
func (r *Route) Seen() {
	r.LastSeen = time.Now()
}

// IsExpired checks if a route is experied, given the current time and a
// threshold
func (r *Route) IsExpired(currentTime time.Time, threshold time.Duration) bool {
	return currentTime.Sub(r.LastSeen) > threshold
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
}

// NewRouteMapping automatically loads the RouteMapping object from storage
func NewRouteMapping(storage *string, dockerEndpoint *string) *RouteMapping {
	rm := &RouteMapping{
		Storage: *storage,
	}
	err := rm.restoreFromFile(storage)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v", rm)

	// Set up
	rm.DockerEndpoint = *dockerEndpoint
	client, err := docker.NewClient(rm.DockerEndpoint)
	if err != nil {
		// Panic if we can't kill containers
		panic(err)
	}
	rm.client = client

	return rm
}

// RemoveDeadContainers is intended to be called with the current time and a
// threshold past which a container with no traffic should be killed. The
// function kills that route's containers, removes the route, and saves to
// file.
func (rm *RouteMapping) RemoveDeadContainers(currentTime time.Time, threshold time.Duration) {
	for _, route := range rm.Routes {
		if route.IsExpired(currentTime, threshold) {
			fmt.Printf("Found expired route %s\n", route)
			rm.RemoveRoute(&route)
			rm.Save()
		}
	}
}

// KillContainers kills all containers associated with a route
func (r *Route) KillContainers(rm *RouteMapping) {
	for _, containerID := range r.ContainerIds {
		err := rm.client.KillContainer(docker.KillContainerOptions{
			ID:     containerID,
			Signal: 9,
		})
		if err != nil {
			fmt.Printf("Error killing container: %s\n", err)
		}
	}
}

// RegisterCleaner sets up a goroutine with a ticker every N seconds which
// checks if there are any expired containers to kill
func (rm *RouteMapping) RegisterCleaner() {
	// Register our new
	ticker := time.NewTicker(time.Second * 3)
	go func(routeMapping *RouteMapping) {
		for t := range ticker.C {
			rm.RemoveDeadContainers(t, rm.NoAccessThreshold)
		}
	}(rm)
}

// StoreToFile serializes the routemappings object to an XML file.
func (rm *RouteMapping) StoreToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	output, err := xml.MarshalIndent(rm, "", "    ")
	if err != nil {
		return err
	}

	f.Write(output)

	return nil
}

func (rm *RouteMapping) restoreFromFile(path *string) error {
	// If the file doesn't exist, just return.
	if _, err := os.Stat(*path); os.IsNotExist(err) {
		return nil
	}

	data, err := ioutil.ReadFile(*path)
	if err != nil {
		return err
	}

	if err := xml.Unmarshal(data, &rm); err != nil {
		return err
	}

	return nil
}

// FindRoute locates a given route based on the URL the request is
// requesting, and the user's cookie. This allows us to have multiple
// /ipython routes that map to different backends, based on who is
// requesting.
func (rm *RouteMapping) FindRoute(url string, cookie string) (*Route, error) {
	fmt.Printf("url: %s, cookie: %s\n", url, cookie)
	for _, x := range rm.Routes {
		if strings.HasPrefix(url, x.FrontendPath) && x.IsAuthorized(cookie) {
			return &x, nil
		}
	}
	return &Route{}, errors.New("Could not find route")
}

// Save is a convenience function to automatically serialize to default
// storage location.
func (rm *RouteMapping) Save() {
	rm.StoreToFile(rm.Storage)
}

// AddRoute adds a new route
func (rm *RouteMapping) AddRoute(url string, backend string, cookie string, containers []string) {
	r := &Route{
		FrontendPath:     url,
		BackendAddr:      backend,
		AuthorizedCookie: cookie,
		LastSeen:         time.Now(),
		ContainerIds:     containers,
	}

	fmt.Printf("Adding new route %#v\n", r)
	rm.Routes = append(rm.Routes, *r)
	// After we add a route, we update the storage map
	rm.Save()
}

// RemoveRoute removes a route
func (rm *RouteMapping) RemoveRoute(route *Route) {
	// More generic cleanup method for route?
	route.KillContainers(rm)
	// Then remove the route proper
	for idx, x := range rm.Routes {
		if reflect.DeepEqual(*route, x) {
			rm.Routes = rm.Routes[:idx+copy(rm.Routes[idx:], rm.Routes[idx+1:])]
			return
		}
	}
}
