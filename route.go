package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

// Route represents connection information to wire up a frontend request to a
// backend
type Route struct {
	FrontendPath     string
	BackendAddr      string
	AuthorizedCookie string
	LastSeen         time.Time
	ContainerIds     []string `xml:ContainerIds`
	Expired          bool
}

// String representation of Route struct
func (r Route) String() string {
	return fmt.Sprintf("%s->%s (LastSeen @ %s, %d containers associated)", r.FrontendPath, r.BackendAddr, r.LastSeen, len(r.ContainerIds))
}

// IsAuthorized checks if a user's cookie is valid for a given route object.
func (r *Route) IsAuthorized(cookie string) bool {
	return r.AuthorizedCookie == cookie
}

// Seen notifies the route object that it was seen recently
func (r *Route) Seen() {
	r.LastSeen = time.Now()
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

// String representation of RouteMapping struct
func (rm RouteMapping) String() string {
	return fmt.Sprintf("RouteMapping <%d routes under %s>", len(rm.Routes), rm.AuthCookieName)
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

// RemoveDeadContainers finds containers with no traffic which should be
// killed. The function kills that route's containers, removes the route, and
// saves to file.
func (rm *RouteMapping) RemoveDeadContainers() {
	log.Debug("Removing expired containers")
	for _, route := range rm.Routes {
		if route.Expired {
			log.Info("Found expired route %s", route)
			rm.RemoveRoute(&route)
		}
	}
	rm.Save()
}

// KillContainers kills all containers associated with a route
func (r *Route) KillContainers(rm *RouteMapping) {
	for _, containerID := range r.ContainerIds {
		log.Debug("Killing %s", containerID)
		err := rm.client.KillContainer(docker.KillContainerOptions{
			ID:     containerID,
			Signal: 9,
		})
		if err != nil {
			log.Warning("Error killing container: %s", err)
		}
	}
}

// RegisterCleaner sets up a goroutine with a ticker every N seconds which
// checks if there are any expired containers to kill
func (rm *RouteMapping) RegisterCleaner() {
	// Register our new
	// TODO: configurable?
	ticker := time.NewTicker(time.Second * 3)
	go func(routeMapping *RouteMapping) {
		for range ticker.C {
			rm.RemoveDeadContainers()
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
	log.Debug("url: %s, cookie: %s", url, cookie)
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

	log.Debug("Adding new route %s", r)
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
