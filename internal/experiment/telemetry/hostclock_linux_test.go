//go:build linux

package telemetry

import "testing"

func TestHostMonotonicNS(t *testing.T) {
	before, after := HostMonotonicNS(), HostMonotonicNS()
	if before == nil || after == nil || *before <= 0 || *after < *before {
		t.Fatal("Linux host monotonic clock unavailable or regressed")
	}
}
