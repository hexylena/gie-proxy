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

    type testcase struct {
        Path string
        ExpectedCode int
        ExpectedMsg string
        ExpectedErr error
    }

    tests := []testcase{
        {"/api", 401, "Invalid API key\n", nil},
        {"/api?api_key=", 401, "Invalid API key\n", nil},
        {"/api?api_key=supersecret", 200, "[]", nil},
    }

    for _, tc := range tests {
        data, code, err := get(ts, tc.Path)
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

    now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
    route := &Route{
        FrontendPath: "/some/path",
        BackendAddr: "1.1.1.1",
        AuthorizedCookie: "gxsesh",
        LastSeen: now,
        ContainerIds: []string{"deadbeef", "cafebabe" },
    }
    routes := []Route{*route}
    tc_data_route, err := json.MarshalIndent(route, "", "    ")
    if err != nil {
        t.Error("Could not serialize test case route", err)
    }
    tc_data_routes, err := json.MarshalIndent(routes, "", "    ")
    if err != nil {
        t.Error("Could not serialize test case route", err)
    }

    type testcase2 struct {
        Path string
        Body []byte
        ExpectedCode int
        ExpectedMsg string
        ExpectedErr error
    }

    tests2 := []testcase2{
        {"/api?api_key=supersecret", nil, 400, "Invalid Route Data\n", nil},
        {"/api?api_key=supersecret", []byte("asdf"), 400, "Invalid Route Data\n", nil},
        {"/api?api_key=supersecret", []byte("{\"FrontendPath\": \"\"}"), 400, "Invalid Route Data\n", nil},
    }

    for _, tc := range tests2 {
        data, code, err := post(ts, tc.Path, tc.Body)
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

    tests3 := []testcase2{
        {"/api?api_key=supersecret", tc_data_route, 200, string(tc_data_routes), nil},
    }
    for _, tc := range tests3 {
        _, code, err := post(ts, tc.Path, tc.Body)
        if len(tsh.RouteMapping.Routes) > 0 {
            for idx := range tsh.RouteMapping.Routes {
                tsh.RouteMapping.Routes[idx].LastSeen = now
            }
        }
        data, code, err := get(ts, tc.Path)
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
}
