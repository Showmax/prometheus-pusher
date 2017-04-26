package main

import (
	"bufio"
	"os"
	"strings"

	iradix "github.com/hashicorp/go-immutable-radix"
)

// route map type
//
// TODO: store pointers to strings as values in radix tree
//
type routeMap struct {
	*iradix.Tree
	defaultRoute []string
}

// creates routeMap instance from a route_map config file
//
func newRouteMap(file string, dr string) *routeMap {
	m := iradix.New()

	fd, err := os.Open(file)
	if err != nil {
		logger.Fatalf("Failed to parse route map %s - %s", file, err.Error())
		return nil
	}

	sc := bufio.NewScanner(fd)
	for sc.Scan() {
		if sc.Text() == "" {
			continue
		}
		if strings.HasPrefix(sc.Text(), "#") {
			continue
		}

		elem := strings.Fields(sc.Text())
		rt := strings.Split(elem[1], ",")
		m, _, _ = m.Insert([]byte(elem[0]), rt)
	}
	r := &routeMap{m, strings.Split(dr, ",")}
	return r
}

// calculates route for given metric name
//
func (r *routeMap) route(name []byte) []string {
	_, value, _ := r.Root().LongestPrefix(name)
	rt, _ := value.([]string)
	if len(rt) == 0 {
		return r.defaultRoute
	}
	return rt
}
