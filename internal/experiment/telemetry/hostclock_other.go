//go:build !linux

package telemetry

// HostMonotonicNS is unavailable outside the Linux measurement testbed.
func HostMonotonicNS() *int64 { return nil }
