package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "default status", wantCode: 0, wantStdout: "Phase 02"},
		{name: "help", args: []string{"--help"}, wantCode: 0, wantStdout: "Usage: plutod"},
		{name: "version", args: []string{"version"}, wantCode: 0, wantStdout: "plutod 0.2.0-foundation"},
		{name: "unknown command", args: []string{"bogus"}, wantCode: 2, wantStderr: "unknown command: bogus"},
		{name: "extra arguments", args: []string{"version", "extra"}, wantCode: 2, wantStderr: "version takes no arguments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)
			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d", code, tt.wantCode)
			}
			if !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout = %q, want substring %q", stdout.String(), tt.wantStdout)
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want substring %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}
