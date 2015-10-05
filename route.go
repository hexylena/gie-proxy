package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
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
	Routes         []Route `xml:"Routes>Route"`
	AuthCookieName string
	Storage        string
}

// NewRouteMapping automatically loads the RouteMapping object from storage
func NewRouteMapping(storage *string) *RouteMapping {
	rm := &RouteMapping{
		Storage: *storage,
	}
	err := rm.restoreFromFile(storage)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v", rm)
	return rm
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
		fmt.Println(x)
		if strings.HasPrefix(url, x.FrontendPath) && x.IsAuthorized(cookie) {
			return &x, nil
		}
	}
	return &Route{}, errors.New("Could not find route")
}

// AddRoute adds a new route
func (rm *RouteMapping) AddRoute(url string, backend string, cookie string) {
	r := &Route{
		FrontendPath:     url,
		BackendAddr:      backend,
		AuthorizedCookie: cookie,
		LastSeen:         time.Now(),
	}

	fmt.Printf("Adding new route %#v", r)
	rm.Routes = append(rm.Routes, *r)
	// After we add a route, we update the storage map
	rm.StoreToFile(rm.Storage)
}

// RemoveRoute removes a route
func (rm *RouteMapping) RemoveRoute(route *Route) {
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
}
