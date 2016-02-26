package main

import (
	//"errors"
	"testing"
	"time"
)

func TestSeen(t *testing.T) {
	type testcase struct {
		FakeNow time.Time
		Result  bool
		Msg     string
	}

	tests := []testcase{
		{FakeNow: time.Now(), Result: true, Msg: "Up-to-date"},
		{FakeNow: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC), Result: false, Msg: "Out of date"},
	}

	for _, tc := range tests {
		tmpRoute := &Route{
			LastSeen: tc.FakeNow,
		}
		if tmpRoute.LastSeen != tc.FakeNow {
			t.Error("Time wasn't set")
		}
		// See it
		tmpRoute.Seen()
		// Check if recent
		if time.Now().Sub(tmpRoute.LastSeen).Seconds() > 10 {
			t.Error("Time wasn't updated saw", tmpRoute.LastSeen, "expected", time.Now())
		}
	}
}

func TestIsAuthorized(t *testing.T) {
	type testcase struct {
		Cookie string
		Result bool
		Msg    string
	}
	tests := []testcase{
		{"C is for Cookie", true, ""},
		{"That's good enough for me", false, "Incorrect cookie"},
	}

	for _, tc := range tests {
		tmpRoute := &Route{
			AuthorizedCookie: "C is for Cookie",
		}

		if tmpRoute.IsAuthorized(tc.Cookie) != tc.Result {
			t.Error(
				"For", tc.Cookie,
				"expected", tc.Result,
				tc.Msg,
			)
		}
	}
}

func TestFindRoute(t *testing.T) {
	type testcase struct {
		URL       string
		Cookie    string
		Result    *Route
		RouteList []Route
		Err       string
		Msg       string
	}
	tests := []testcase{
		{
			URL:       "/galaxy/gie_proxy/ipython",
			Cookie:    "valid",
			Result:    &Route{},
			Err:       "Could not find route",
			Msg:       "Do not excpect to return from empty route list",
			RouteList: []Route{},
		},
		{
			URL:       "/galaxy/gie_proxy/ipython",
			Cookie:    "invalid",
			Result:    &Route{},
			Err:       "Could not find route",
			Msg:       "Do not excpect to return from empty route list",
			RouteList: []Route{},
		},
	}

	for _, tc := range tests {
		rm := &RouteMapping{
			Routes: tc.RouteList,
		}

		result, err := rm.FindRoute(tc.URL, tc.Cookie)

		if err != nil {
			if err.Error() != tc.Err {
				t.Error("Expected", tc.Err, "found", err)
			}
		}

		if result.FrontendPath != tc.Result.FrontendPath || result.BackendAddr != result.BackendAddr {
			t.Error(
				"For", tc.URL, "and",
				tc.Cookie, "expected", tc.Result,
				"found", result,
			)
		}
	}
}
