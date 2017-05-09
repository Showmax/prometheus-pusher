package main

import "testing"

func TestConfigParse(t *testing.T) {
	if _, err := parseConfig(cfgTest); err != nil {
		t.Fatalf("Failed to parse config - %s", err.Error())
	}
}
