package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Route struct {
	FrontendPath     string `json:"frontend"`
	BackendAddr      string `json:"backendAddr"`
	AuthorizedCookie string `json:"cookie"`
}

func (r *Route) IsAuthorized(cookie string) bool {
	return r.AuthorizedCookie == cookie
}

type RouteMappings struct {
	Routes         []*Route
	AuthCookieName string
	Storage        string
}

func (rm *RouteMappings) StoreToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	jsonRoutes, err := json.Marshal(rm.Routes)
	if err != nil {
		return err
	}
	f.Write(jsonRoutes)
	return nil
}

func NewRouteMapping(storage *string) *RouteMappings {
	rm := &RouteMappings{
		Storage: *storage,
	}
	rm.RestoreFromFile(storage)

	return rm
}

func (rm *RouteMappings) RestoreFromFile(path *string) error {
	data, err := ioutil.ReadFile(*path)
	if err != nil {
		return err
	}
	routes := make([]*Route, 0)

	if err := json.Unmarshal(data, routes); err != nil {
		return err
	}

	rm.Routes = routes
	return nil
}

func (rm *RouteMappings) FindRoute(url string, cookie string) (*Route, error) {
	fmt.Printf("url: %s, cookie: %s\n", url, cookie)
	for _, x := range rm.Routes {
		fmt.Println(x)
		if strings.HasPrefix(url, x.FrontendPath) && x.IsAuthorized(cookie) {
			return x, nil
		}
	}
	return &Route{}, errors.New("Could not find route")
}

// Register a new route
func (rm *RouteMappings) AddRoute(url string, backend string, cookie string) {
	r := &Route{
		FrontendPath:     url,
		BackendAddr:      backend,
		AuthorizedCookie: cookie,
	}

	rm.Routes = append(rm.Routes, r)
	// After we add a route, we update the storage map
	rm.StoreToFile(rm.Storage)
}

// Remove a route mapping
// TODO
//func (rm *RouteMappings) RemoveRoute(url string, backend string, cookie string) {
//tmpr := &Route{
//FrontendPath:     url,
//BackendAddr:       backend,
//AuthorizedCookie: cookie,
//}
//for idx, x := range rm.Routes {
//if tmpr == x {
//sliceA := rm.Routes[:idx]
//sliceB := rm.Routes[idx+1:]
////rm.Routes = append(sliceA, sliceB)
//return
//}
//}
//}
