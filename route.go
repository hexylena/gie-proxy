package main

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

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

// String representation of RouteMapping struct
func (rm RouteMapping) String() string {
	return fmt.Sprintf("RouteMapping <%d routes under %s>", len(rm.Routes), rm.AuthCookieName)
}

// InitializeRouteMapper automatically loads the RouteMapping object from storage
func InitializeRouteMapper(rm *RouteMapping) {
	err := rm.restoreFromFile(rm.Storage)
	if err != nil {
		panic(err)
	}
	log.Info("Restored %d RouteMapper routes from storage", len(rm.Routes))

	client, err := docker.NewClient(rm.DockerEndpoint)
	if err != nil {
		// Panic if we can't kill containers
		panic(err)
	}
	rm.client = client
	log.Info("Connected RouteMapper to Docker")

	rm.RegisterCleaner()
}

// RemoveDeadContainers finds containers with no traffic which should be
// killed. The function kills that route's containers, removes the route, and
// saves to file.
func (rm *RouteMapping) RemoveDeadContainers() {
	for _, route := range rm.Routes {
		if route.Expired || time.Since(route.LastSeen) > rm.NoAccessThreshold {
			log.Info("Found expired route %s", route)
			rm.RemoveRoute(&route)
		}
	}
	rm.Save()
}

// KillContainers kills all containers associated with a route
func (r *Route) KillContainers(rm *RouteMapping) {
	for _, containerID := range r.ContainerIds {
		log.Info("Killing %s", containerID)
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
	ticker := time.NewTicker(rm.CleanInterval)
	go func(routeMapping *RouteMapping) {
		for range ticker.C {
			log.Info("Running goroutines: %d", runtime.NumGoroutine())
			rm.RemoveDeadContainers()
		}
	}(rm)
}

// FindRoute locates a given route based on the URL the request is
// requesting, and the user's cookie. This allows us to have multiple
// /ipython routes that map to different backends, based on who is
// requesting.
func (rm *RouteMapping) FindRoute(url string, cookie string) (*Route, error) {
	for _, x := range rm.Routes {
		if strings.HasPrefix(url, x.FrontendPath) && x.IsAuthorized(cookie) {
			return &x, nil
		}
	}
	return &Route{}, errors.New("Could not find route")
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

	log.Info("Adding new route %s", r)
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
		// TODO
		if route.FrontendPath == x.FrontendPath && route.BackendAddr == x.BackendAddr && route.AuthorizedCookie == x.AuthorizedCookie {
			rm.Routes = rm.Routes[:idx+copy(rm.Routes[idx:], rm.Routes[idx+1:])]
			return
		}
	}
}
