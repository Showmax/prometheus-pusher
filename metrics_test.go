package main

import (
	"bufio"
	"bytes"
	// "fmt"
	"regexp"
	"testing"
)

func TestMetrics(t *testing.T) {
	var m *metrics
	var mapped map[string][]byte
	rm := newRouteMap("test/routes", "test")
	testRe := regexp.MustCompile(`^(?:\w+(?:{.*?})?)\s(?:-?\d+(?:\.\d+(?:e(\+|-)\d+)?)?)\s(?:\d{8,14})$`)
	c, _ := parseConfig(cfgTest)

	t.Run("new", func(t *testing.T) {
		m = newMetrics(mbTest, c)
		if len(m.dBrd) != 889 {
			t.Fatalf("Expected to read 889 metrics, got %d", len(m.dBrd))
		}
	})

	t.Run("imux", func(t *testing.T) {
		mapped = m.imux(rm, c)
		if len(mapped) != 5 {
			t.Fatalf("Test expected to result in 5 destination buckets, but got %d", len(mapped))
		}
		for dst, data := range mapped {
			t.Run(dst, func(t *testing.T) {
				scn := bufio.NewScanner(bytes.NewBuffer(data))
				for scn.Scan() {
					if scn.Bytes()[0] != '#' {
						// t.Logf("%s - %s\n", dst, scn.Text())
						if !testRe.Match(scn.Bytes()) {
							t.Fatalf("Metrics line `%s` doesn't correspond to the expected format (name[{fields}] value timestamp)", scn.Text())
						}
					}
				}
			})
		}
	})

}

func BenchmarkMetrics(b *testing.B) {
	rm := newRouteMap("test/routes", "test")
	c, _ := parseConfig(cfgTest)
	for i := 0; i < b.N; i++ {
		newMetrics(mbTest, c).imux(rm, c)
	}
}
