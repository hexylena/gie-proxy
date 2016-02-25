package main

import (
	"net/http"
)

type frontend struct {
	Addr string
	Path string
}

type requestHandler struct {
	Transport    *http.Transport
	RouteMapping *RouteMapping
	Frontend     *frontend
	CookieName   string
}

type apiHandler struct {
	Transport    *http.Transport
	RouteMapping *RouteMapping
	APIKey       string
}
