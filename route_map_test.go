package main

import (
	"strings"
	"testing"
)

func TestRouteMap(t *testing.T) {
	loadTestConfig()
	var rm *routeMap
	routeCases := []struct {
		try    []byte
		expect []string
	}{
		{[]byte("http_requests"), []string{"test2", "test-bck"}},
		{[]byte("go_goroutines"), []string{"test1", "test-bck"}},
		{[]byte("node_memory"), []string{"test4", "test-bck"}},
		{[]byte("node_exporter_count"), []string{"test1", "test-bck"}},
		{[]byte("node_netstat_open"), []string{"test3", "test-bck"}},
		{[]byte("something_unknown"), []string{"test0", "test-bck"}},
	}

	t.Run("new", func(t *testing.T) {
		rm = newRouteMap("test/routes", "test0,test-bck")
		if rm.Len() != 26 {
			t.Fatalf("Route map should contain 26 elements, but has %d", rm.Len())
		}
	})

	t.Run("route", func(t *testing.T) {
		for _, c := range routeCases {
			t.Run(string(c.try), func(t *testing.T) {
				route := rm.route(c.try)
			loop:
				for _, r := range route {
					for _, e := range c.expect {
						if r == e {
							continue loop
						}
					}
					t.Fatalf("Metric `%s` expected to be routed to `%s`, but got `%s`", c.try, strings.Join(c.expect, ","), strings.Join(route, ","))
				}
			})
		}
	})
}
