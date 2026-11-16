//go:build linux

package researchv2

import "golang.org/x/sys/unix"

// stageThreadCPU reads the CPU clock of the currently locked OS thread.
func stageThreadCPU() (int64, bool) {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_THREAD_CPUTIME_ID, &ts); err != nil {
		return 0, false
	}
	return ts.Sec*1_000_000_000 + ts.Nsec, true
}
