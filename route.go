package main

import (
	"bufio"
	"errors"
	"fmt"
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
	Routes         []Route
	AuthCookieName string
	Storage        string
}

func (rm *RouteMappings) StoreToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, route := range rm.Routes {
		fmt.Fprintf(f, "%s\t%s\t%s\n", route.FrontendPath, route.BackendAddr, route.AuthorizedCookie)
	}
	return nil
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
	fmt.Printf("%#v", rm.Routes)

	return rm
}

// TODO: xml/json
func (rm *RouteMappings) RestoreFromFile(path *string) error {
	// If the file doesn't exist, just return.
	if _, err := os.Stat(*path); os.IsNotExist(err) {
		return nil
	}

	// Open file
	f, err := os.Open(*path)
	if err != nil {
		return err
	}
	defer f.Close()

	routes := make([]Route, 0)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		linedata := strings.Split(scanner.Text(), "\t")
		if len(linedata) != 3 {
			return errors.New("Improperly formatted line! " + scanner.Text())
		}
		route := &Route{
			FrontendPath:     linedata[0],
			BackendAddr:      linedata[1],
			AuthorizedCookie: linedata[2],
		}
		routes = append(routes, *route)
	}

	rm.Routes = routes
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

	rm.Routes = append(rm.Routes, *r)
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
