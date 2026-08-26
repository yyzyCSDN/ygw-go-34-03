package main

import "testing"

func TestProbeRunsEndToEnd(t *testing.T) {
	cfg := DefaultConfig()
	if err := Probe(cfg); err != nil {
		t.Fatalf("probe failed: %v", err)
	}
}
