//go:build linux

package telemetry

import "golang.org/x/sys/unix"

// HostMonotonicNS reads the Linux host-wide monotonic clock. It is used only
// in optional measurement paths and never affects transaction validity.
func HostMonotonicNS() *int64 {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return nil
	}
	value := ts.Sec*1_000_000_000 + ts.Nsec
	return &value
}
