package main

import (
	"io/ioutil"
    "bytes"
	"testing"
    "time"
	"net/http"
    "encoding/json"
	"net/http/httptest"
)

func get(ts *httptest.Server, path string) (string, int, error) {
	res, err := http.Get(ts.URL + path)
	if err != nil {
        return "", 0, err
	}
	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
        return "", 0, err
	}
    return string(data), res.StatusCode, nil
}
type testcase struct {
    Path string
    Body []byte
    ExpectedCode int
    ExpectedMsg string
    ExpectedErr error
}

func post(ts *httptest.Server, path string, jsonStr []byte) (string, int, error) {
    req, err := http.NewRequest("POST", ts.URL + path, bytes.NewBuffer(jsonStr))
    client := &http.Client{}
    res, err := client.Do(req)
    if err != nil {
        return "", 0, err
    }
	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
        return "", 0, err
	}
    return string(data), res.StatusCode, nil
}

func apiTest(data string, code int, err error, tc testcase, t *testing.T) {
    if err != tc.ExpectedErr {
        t.Error("Request to", tc.Path, "had error", err, "expected error", tc.ExpectedErr)
    }

    if code != tc.ExpectedCode {
        t.Error("Request to", tc.Path, "had code", code, "expected code", tc.ExpectedCode)
    }

    if data != tc.ExpectedMsg {
        t.Error("Request to", tc.Path, "had body", data, "expected body", tc.ExpectedMsg)
    }
}

func TestApiServeHTTP_get(t *testing.T) {
    frontend := &frontend{
        Addr:   "127.0.0.1",
        Path:   "/gxproxy",
        APIKey: "supersecret",
    }
	var tsh = &apiHandler{
		Transport: &http.Transport{
			DisableKeepAlives:  false,
			DisableCompression: false,
		},
		RouteMapping: &RouteMapping{
			AuthCookieName: "sid",
			Routes:         make([]Route, 0),
			Storage:        "/dev/null",
		},
		Frontend: frontend,
	}
	ts := httptest.NewServer(tsh)
    defer ts.Close()

    tests := []testcase{
        {"/api", nil, 401, "Invalid API key\n", nil},
        {"/api?api_key=", nil, 401, "Invalid API key\n", nil},
        {"/api?api_key=supersecret", nil, 200, "[]", nil},
    }

    for _, tc := range tests {
        data, code, err := get(ts, tc.Path)

        apiTest(data, code, err, tc, t)
    }

    now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
    route := &Route{
        FrontendPath: "/some/path",
        BackendAddr: "1.1.1.1",
        AuthorizedCookie: "gxsesh",
        LastSeen: now,
        ContainerIds: []string{"deadbeef", "cafebabe" },
    }
    routes := []Route{*route}
    tcDataRoute, err := json.MarshalIndent(route, "", "    ")
    if err != nil {
        t.Error("Could not serialize test case route", err)
    }
    tcDataRoutes, err := json.MarshalIndent(routes, "", "    ")
    if err != nil {
        t.Error("Could not serialize test case route", err)
    }


    tests2 := []testcase{
        {"/api?api_key=supersecret", nil, 400, "Invalid Route Data\n", nil},
        {"/api?api_key=supersecret", []byte("asdf"), 400, "Invalid Route Data\n", nil},
        {"/api?api_key=supersecret", []byte("{\"FrontendPath\": \"\"}"), 400, "Invalid Route Data\n", nil},
    }

    for _, tc := range tests2 {
        data, code, err := post(ts, tc.Path, tc.Body)
        apiTest(data, code, err, tc, t)
    }

    tests3 := []testcase{
        {"/api?api_key=supersecret", tcDataRoute, 200, string(tcDataRoutes), nil},
    }
    for _, tc := range tests3 {
        _, code, err := post(ts, tc.Path, tc.Body)
        if len(tsh.RouteMapping.Routes) > 0 {
            for idx := range tsh.RouteMapping.Routes {
                tsh.RouteMapping.Routes[idx].LastSeen = now
            }
        }
        data, code, err := get(ts, tc.Path)
        apiTest(data, code, err, tc, t)
    }
}
