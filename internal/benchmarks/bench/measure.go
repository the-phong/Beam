// Package bench contains research-only callback timers. It is never imported
// by the Pluto production application or consensus-critical code.
package bench

import "time"

type Stage string

const (
	S1  Stage = "S1_structural_parse"
	S2  Stage = "S2_profile_chain"
	S3  Stage = "S3_key_binding"
	S4  Stage = "S4_secp_verify"
	S5  Stage = "S5_ml_verify"
	S6  Stage = "S6_state_rules"
	S7  Stage = "S7_check_tx"
	S8  Stage = "S8_proposal"
	S9  Stage = "S9_finalize"
	S10 Stage = "S10_durable_commit"
)

type Sample struct {
	Stage      Stage `json:"stage"`
	DurationNS int64 `json:"duration_ns"`
}

// Recorder measures the wall time around explicitly selected operations.
// Disabled Do calls the operation directly. A full buffer drops only samples,
// never transaction work or an error; thus measurement cannot affect validity.
type Recorder struct {
	Enabled  bool
	Capacity int
	Samples  []Sample
	Dropped  uint64
}

func (r *Recorder) Do(stage Stage, operation func() error) error {
	if r == nil || !r.Enabled {
		return operation()
	}
	start := time.Now()
	err := operation()
	duration := time.Since(start).Nanoseconds()
	if r.Capacity > 0 && len(r.Samples) < r.Capacity {
		r.Samples = append(r.Samples, Sample{Stage: stage, DurationNS: duration})
	} else {
		r.Dropped++
	}
	return err
}
