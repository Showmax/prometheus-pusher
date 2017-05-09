package main

import (
	"bufio"
	"bytes"
	"regexp"
	"testing"
)

func TestMetrics(t *testing.T) {
	var m *metrics
	var mapped map[string][]byte
	rm := newRouteMap("test/routes", "test")
	testRe := regexp.MustCompile("^[a-zA-Z0-9_]+({.+})? [\\-0-9e+\\.]")

	t.Run("new", func(t *testing.T) {
		m = newMetrics(mbTest)
		if len(m.brd) != 889 {
			t.Fatalf("Expected to read 889 metrics, got %d", len(m.brd))
		}
	})

	t.Run("imux", func(t *testing.T) {
		mapped = m.imux(rm)
		if len(mapped) != 5 {
			t.Fatalf("Test expected to result in 5 destination buckets, but got %d", len(mapped))
		}
		for dst, data := range mapped {
			t.Run(dst, func(t *testing.T) {
				scn := bufio.NewScanner(bytes.NewBuffer(data))
				for scn.Scan() {
					if !testRe.Match(scn.Bytes()) {
						t.Fatalf("Metrics line `%s` doesn't correspond to the expected format (name[{fields}] value timestamp)", scn.Text())
					}
				}
			})
		}
	})

}

func BenchmarkMetrics(b *testing.B) {
	rm := newRouteMap("test/routes", "test")
	for i := 0; i < b.N; i++ {
		newMetrics(mbTest).imux(rm)
	}
}
