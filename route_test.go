package dynoapi

import (
	"net/http"
	"testing"

	"github.com/codingconcepts/dynoapi/test"
)

func TestAddRoute(t *testing.T) {
	route := RouteConfig{
		Body:          "body",
		DurationParam: "durationParam",
		Example:       "example",
		Method:        "method",
		StatusCode:    http.StatusOK,
		URI:           "url",
	}

	testCases := []struct {
		name           string
		set            func(RouteConfig) RouteConfig
		expectedResult bool
	}{
		{
			name: "body",
			set: func(r RouteConfig) RouteConfig {
				r.Body = "different"
				return r
			},
			expectedResult: false,
		},
		{
			name: "durationArg",
			set: func(r RouteConfig) RouteConfig {
				r.DurationParam = "different"
				return r
			},
			expectedResult: false,
		},
		{
			name: "example",
			set: func(r RouteConfig) RouteConfig {
				r.Example = "different"
				return r
			},
			expectedResult: false,
		},
		{
			name: "method",
			set: func(r RouteConfig) RouteConfig {
				r.Method = "different"
				return r
			},
			expectedResult: false,
		},
		{
			name: "statusCode",
			set: func(r RouteConfig) RouteConfig {
				r.StatusCode = http.StatusCreated
				return r
			},
			expectedResult: false,
		},
		{
			name: "uri",
			set: func(r RouteConfig) RouteConfig {
				r.URI = "different"
				return r
			},
			expectedResult: false,
		},
		{
			name: "same",
			set: func(r RouteConfig) RouteConfig {
				return r
			},
			expectedResult: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			otherRoute := testCase.set(route)
			test.Assert(t, otherRoute.Equals(route) == testCase.expectedResult)
		})
	}
}

func TestMerge(t *testing.T) {
	left := &RouteConfigs{
		RouteConfig{Body: "a", DurationParam: "b", Example: "c", Method: "d", StatusCode: 1, URI: "e"},
		RouteConfig{Body: "b", DurationParam: "c", Example: "d", Method: "e", StatusCode: 2, URI: "f"},
	}

	right := RouteConfigs{
		RouteConfig{Body: "a", DurationParam: "b", Example: "c", Method: "d", StatusCode: 1, URI: "e"},
		RouteConfig{Body: "b", DurationParam: "c", Example: "d", Method: "e", StatusCode: 2, URI: "f"},
		RouteConfig{Body: "c", DurationParam: "d", Example: "e", Method: "f", StatusCode: 3, URI: "g"},
		RouteConfig{Body: "c", DurationParam: "c", Example: "d", Method: "e", StatusCode: 2, URI: "f"},
		RouteConfig{Body: "b", DurationParam: "d", Example: "d", Method: "e", StatusCode: 2, URI: "f"},
		RouteConfig{Body: "b", DurationParam: "c", Example: "e", Method: "e", StatusCode: 2, URI: "f"},
		RouteConfig{Body: "b", DurationParam: "c", Example: "d", Method: "f", StatusCode: 2, URI: "f"},
		RouteConfig{Body: "b", DurationParam: "c", Example: "d", Method: "e", StatusCode: 3, URI: "f"},
		RouteConfig{Body: "b", DurationParam: "c", Example: "d", Method: "e", StatusCode: 2, URI: "g"},
	}

	left.Merge(right...)

	test.Equals(t, 9, len(*left))
}
