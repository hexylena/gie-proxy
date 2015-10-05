package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Route struct {
	FrontendPath     string
	BackendAddr      string
	AuthorizedCookie string
}

func (r *Route) IsAuthorized(cookie string) bool {
	return r.AuthorizedCookie == cookie
}

type RouteMappings struct {
	Routes         []Route `xml:"Routes>Route"`
	AuthCookieName string
	Storage        string
}

// Init function, automatically loads from storage
func NewRouteMapping(storage *string) *RouteMappings {
	rm := &RouteMappings{
		Storage: *storage,
	}
	err := rm.RestoreFromFile(storage)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v", rm)
	return rm
}

func (rm *RouteMappings) StoreToFile(path string) error {
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

func (rm *RouteMappings) RestoreFromFile(path *string) error {
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

func (rm *RouteMappings) FindRoute(url string, cookie string) (*Route, error) {
	fmt.Printf("url: %s, cookie: %s\n", url, cookie)
	for _, x := range rm.Routes {
		fmt.Println(x)
		if strings.HasPrefix(url, x.FrontendPath) && x.IsAuthorized(cookie) {
			return &x, nil
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

	fmt.Printf("Adding new route %#v", r)
	rm.Routes = append(rm.Routes, *r)
	// After we add a route, we update the storage map
	rm.StoreToFile(rm.Storage)
}

// Remove a route mapping
// TODO
func (rm *RouteMappings) RemoveRoute(route *Route) {
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
