//go:build !linux

package researchv2

func stageThreadCPU() (int64, bool) { return 0, false }
