package main

import (
	"errors"
	"fmt"
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
	Routes         []*Route
	AuthCookieName string
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
