package load

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"example.com/pluto/internal/experiment/profiles"
)

var ErrPlan = errors.New("invalid isolated workload plan")

var runIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

const (
	EqualTxRate                  = "equal_tx_s"
	EqualByteRate                = "equal_bytes_s"
	FixedLegitimatePlusAdversary = "fixed_legitimate_plus_adversarial"
)

// PlannedTx contains only pre-signed public wire bytes. Signing keys never
// enter the load process. This plan is a private run artifact because it also
// contains scenario labels and account scheduling metadata.
type PlannedTx struct {
	ID            string           `json:"id"`
	OffsetNS      int64            `json:"offset_ns"`
	Traffic       string           `json:"traffic"`
	Scenario      string           `json:"scenario"`
	Capability    string           `json:"capability"`
	Source        string           `json:"source"`
	AccountSlot   int              `json:"account_slot"`
	Nonce         uint64           `json:"nonce"`
	BytesHex      string           `json:"bytes_hex"`
	SHA256        string           `json:"sha256"`
	ExpectedCode  researchv2.Code  `json:"expected_code"`
	ExpectedStage researchv2.Stage `json:"expected_stage"`
}

type Plan struct {
	Version          int                `json:"version"`
	RunID            string             `json:"run_id"`
	ConfigSHA256     string             `json:"config_sha256"`
	GenesisSHA256    string             `json:"genesis_sha256"`
	Profile          researchv2.Profile `json:"profile"`
	Regime           string             `json:"regime"`
	Waveform         string             `json:"waveform"`
	Seed             int64              `json:"seed"`
	WarmupNS         int64              `json:"warmup_ns"`
	SteadyNS         int64              `json:"steady_ns"`
	DrainNS          int64              `json:"drain_ns"`
	WindowNS         int64              `json:"window_ns"`
	TargetTxPS       float64            `json:"target_tx_per_second"`
	TargetBytesPS    float64            `json:"target_bytes_per_second"`
	TargetLegitTPS   float64            `json:"target_legitimate_tx_per_second"`
	TargetInvalidTPS float64            `json:"target_invalid_tx_per_second"`
	ReplayMode       bool               `json:"replay_mode,omitempty"`
	Events           []PlannedTx        `json:"events"`
}

func ReadPlan(path string, c Config, configHash string) (Plan, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Plan{}, "", err
	}
	var p Plan
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return Plan{}, "", fmt.Errorf("%w: %v", ErrPlan, err)
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Plan{}, "", fmt.Errorf("%w: trailing document", ErrPlan)
	}
	if err := p.Validate(c, configHash); err != nil {
		return Plan{}, "", err
	}
	h := sha256.Sum256(raw)
	return p, hex.EncodeToString(h[:]), nil
}

// WritePlan creates one private artifact and refuses to overwrite a prior run.
func WritePlan(path string, p Plan, c Config, configHash string) error {
	if err := p.Validate(c, configHash); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(raw, '\n'))
	return err
}

func (p Plan) Validate(c Config, configHash string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	maxDuration := int64(time.Duration(c.Safety.MaxDurationSeconds) * time.Second)
	if p.Version != 1 || !runIDPattern.MatchString(p.RunID) || !validHash(p.ConfigSHA256) || p.ConfigSHA256 != configHash || !validHash(p.GenesisSHA256) || !p.Profile.Valid() ||
		p.WarmupNS < 0 || p.SteadyNS < int64(time.Second) || p.DrainNS < 0 || p.WindowNS != int64(time.Second) || p.WarmupNS%p.WindowNS != 0 || p.SteadyNS%p.WindowNS != 0 ||
		p.WarmupNS > maxDuration || p.SteadyNS > maxDuration-p.WarmupNS || p.DrainNS > maxDuration-p.WarmupNS-p.SteadyNS || len(p.Events) < 1 || len(p.Events) > 6000 ||
		(p.Waveform != "constant" && p.Waveform != "burst" && p.Waveform != "ramp") || (p.Regime != EqualTxRate && p.Regime != EqualByteRate && p.Regime != FixedLegitimatePlusAdversary) ||
		(p.Waveform == "ramp" && p.Regime != EqualTxRate) {
		return ErrPlan
	}
	if p.ReplayMode && (len(p.RunID) < 4 || p.RunID[:4] != "A06-" || p.Profile != researchv2.ProfileH || p.Regime != EqualTxRate || p.Waveform != "constant") {
		return ErrPlan
	}
	if p.Regime == EqualTxRate && (p.TargetTxPS <= 0 || p.TargetBytesPS != 0 || p.TargetLegitTPS != 0 || p.TargetInvalidTPS != 0) ||
		p.Regime == EqualByteRate && (p.TargetBytesPS <= 0 || p.TargetTxPS != 0 || p.TargetLegitTPS != 0 || p.TargetInvalidTPS != 0) ||
		p.Regime == FixedLegitimatePlusAdversary && (p.TargetLegitTPS <= 0 || p.TargetInvalidTPS <= 0 || p.TargetTxPS != 0 || p.TargetBytesPS != 0 || p.Waveform != "constant") {
		return ErrPlan
	}
	seenID := map[string]bool{}
	seenHash := map[string]bool{}
	firstHashEvent := map[string]PlannedTx{}
	seenLegitSlot := map[int]bool{}
	windowCount := map[int64]int{}
	windowBytes := map[int64]int{}
	var steadyCount, steadyBytes, steadyLegit, steadyInvalid int
	var previous int64 = -1
	endSend := p.WarmupNS + p.SteadyNS
	for _, e := range p.Events {
		first := firstHashEvent[e.SHA256]
		repeated := seenHash[e.SHA256]
		allowedReplay := p.ReplayMode && repeated && first.Traffic == "adversarial" && e.Traffic == "adversarial" &&
			first.Scenario == e.Scenario && first.AccountSlot == e.AccountSlot && first.ExpectedCode == e.ExpectedCode &&
			first.Capability == e.Capability && (e.Scenario == "A06_valid_replay" || e.Scenario == "A06_bad_secp_repeated")
		if !runIDPattern.MatchString(e.ID) || seenID[e.ID] || !validHash(e.SHA256) || repeated && !allowedReplay || e.OffsetNS < 0 || e.OffsetNS < previous || e.OffsetNS >= endSend ||
			e.Source != "rpc" || e.AccountSlot < 0 || (e.Traffic != "legitimate" && e.Traffic != "adversarial") || e.Scenario == "" || e.Capability == "" ||
			(e.Traffic == "legitimate" && (e.ExpectedCode != researchv2.CodeOK || e.Nonce != 0 || seenLegitSlot[e.AccountSlot])) ||
			(e.Traffic == "adversarial" && e.ExpectedCode == researchv2.CodeOK && !(p.ReplayMode && e.Scenario == "A06_valid_replay")) {
			return fmt.Errorf("%w: event metadata", ErrPlan)
		}
		raw, err := hex.DecodeString(e.BytesHex)
		if err != nil || len(raw) == 0 || len(raw) > c.Network.MempoolMaxTxBytes {
			return fmt.Errorf("%w: event wire bytes", ErrPlan)
		}
		h := sha256.Sum256(raw)
		if hex.EncodeToString(h[:]) != e.SHA256 {
			return fmt.Errorf("%w: event hash", ErrPlan)
		}
		if tx, err := researchv2.Parse(raw); err == nil {
			if tx.Profile != p.Profile && e.ExpectedCode != researchv2.CodeWrongProfile {
				return fmt.Errorf("%w: event profile", ErrPlan)
			}
		} else if e.ExpectedCode != researchv2.CodeMalformed {
			return fmt.Errorf("%w: malformed event classification", ErrPlan)
		}
		seenID[e.ID], seenHash[e.SHA256] = true, true
		if !repeated {
			firstHashEvent[e.SHA256] = e
		}
		if e.Traffic == "legitimate" {
			seenLegitSlot[e.AccountSlot] = true
		}
		previous = e.OffsetNS
		window := e.OffsetNS / p.WindowNS
		windowCount[window]++
		windowBytes[window] += len(raw)
		if windowCount[window] > c.Safety.MaxTxPerSecond || windowBytes[window] > c.Safety.MaxBytesPerSecond {
			return fmt.Errorf("%w: planned rate exceeds hard cap", ErrPlan)
		}
		if e.OffsetNS >= p.WarmupNS {
			steadyCount++
			steadyBytes += len(raw)
			if e.Traffic == "legitimate" {
				steadyLegit++
			} else {
				steadyInvalid++
			}
		}
	}
	seconds := float64(p.SteadyNS) / float64(time.Second)
	within := func(actual int, target float64) bool {
		if target <= 0 {
			return false
		}
		ratio := float64(actual) / (target * seconds)
		return ratio >= .95 && ratio <= 1.05
	}
	if p.Waveform == "constant" || p.Waveform == "ramp" {
		switch p.Regime {
		case EqualTxRate:
			if !within(steadyCount, p.TargetTxPS) {
				return fmt.Errorf("%w: planned tx rate miss", ErrPlan)
			}
		case EqualByteRate:
			if !within(steadyBytes, p.TargetBytesPS) {
				return fmt.Errorf("%w: planned byte rate miss", ErrPlan)
			}
		case FixedLegitimatePlusAdversary:
			if !within(steadyLegit, p.TargetLegitTPS) || !within(steadyInvalid, p.TargetInvalidTPS) {
				return fmt.Errorf("%w: planned split rate miss", ErrPlan)
			}
		}
	}
	return nil
}

func validHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// ComparablePlans prevents a paired comparison from silently changing its
// declared matched pressure. It does not assert achieved offered rates.
func ComparablePlans(a, b Plan) error {
	if a.Regime != b.Regime || a.Waveform != b.Waveform || a.SteadyNS != b.SteadyNS || a.WindowNS != b.WindowNS {
		return ErrPlan
	}
	switch a.Regime {
	case EqualTxRate:
		if a.TargetTxPS != b.TargetTxPS {
			return ErrPlan
		}
	case EqualByteRate:
		if a.TargetBytesPS != b.TargetBytesPS {
			return ErrPlan
		}
	case FixedLegitimatePlusAdversary:
		if a.TargetLegitTPS != b.TargetLegitTPS || a.TargetInvalidTPS != b.TargetInvalidTPS {
			return ErrPlan
		}
	default:
		return ErrPlan
	}
	return nil
}
