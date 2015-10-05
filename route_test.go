package main

import (
	//"errors"
	"testing"
)

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
		Url       string
		Cookie    string
		Result    *Route
		RouteList []Route
		Err       string
		Msg       string
	}
	tests := []testcase{
		{
			Url:       "/galaxy/gie_proxy/ipython",
			Cookie:    "valid",
			Result:    &Route{},
			Err:       "Could not find route",
			Msg:       "Do not excpect to return from empty route list",
			RouteList: []Route{},
		},
		{
			Url:       "/galaxy/gie_proxy/ipython",
			Cookie:    "invalid",
			Result:    &Route{},
			Err:       "Could not find route",
			Msg:       "Do not excpect to return from empty route list",
			RouteList: []Route{},
		},
	}

	for _, tc := range tests {
		rm := &RouteMappings{
			Routes: tc.RouteList,
		}

		result, err := rm.FindRoute(tc.Url, tc.Cookie)

		if err != nil {
			if err.Error() != tc.Err {
				t.Error("Expected", tc.Err, "found", err)
			}
		}

		if result.FrontendPath != tc.Result.FrontendPath || result.BackendAddr != result.BackendAddr {
			t.Error(
				"For", tc.Url, "and",
				tc.Cookie, "expected", tc.Result,
				"found", result,
			)
		}
	}
}
