package main

type Route struct {
	FrontendPath     string
	BackendUrl       string
	AuthorizedCookie string
}

func (r *Route) IsAuthorized(cookie string) bool {
	return true
	//return r.AuthorizedCookie == cookie
}
