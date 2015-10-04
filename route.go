package main

import (
	"errors"
)

type Route struct {
	FrontendPath     string
	BackendUrl       string
	AuthorizedCookie string
}

func (r *Route) IsAuthorized(cookie_key string, cookie_value string) bool {
	return true
	//return r.AuthorizedCookie == cookie
}

type RouteMappings struct {
	Routes         []*Route
	AuthCookieName string
}

func (rm *RouteMappings) FindRoute(url string, cookie string) (*Route, error) {
	for _, x := range rm.Routes {
		if x.FrontendPath == url && x.IsAuthorized(rm.AuthCookieName, cookie) {
			return x, nil
		}
	}
	return &Route{}, errors.New("Could not find route")
}

// Register a new route
func (rm *RouteMappings) AddRoute(url string, backend string, cookie string) {
	r := &Route{
		FrontendPath:     url,
		BackendUrl:       backend,
		AuthorizedCookie: cookie,
	}

	rm.Routes = append(rm.Routes, r)
}

// Remove a route mapping
// TODO
//func (rm *RouteMappings) RemoveRoute(url string, backend string, cookie string) {
//tmpr := &Route{
//FrontendPath:     url,
//BackendUrl:       backend,
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
