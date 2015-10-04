package main

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
	Routes         []Route
	AuthCookieName string
}

func (rm *RouteMappings) FindRoute(url string, cookie string) Route {
	for _, x := range rm.Routes {
		if x.FrontendPath == url && x.IsAuthorized(rm.AuthCookieName, cookie) {
			return x
		}
	}
	return nil
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
func (rm *RouteMappings) RemoveRoute(url string, backend string, cookie string) {
	tmpr := &Route{
		FrontendPath:     url,
		BackendUrl:       backend,
		AuthorizedCookie: cookie,
	}
	for idx, x := range rm.Routes {
		if tmpr == x {
			rm.Routes = append(rm.Routes[:i], rm.Routes[i+1:])
			return
		}
	}
}
