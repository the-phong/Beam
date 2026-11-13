// R02 is a research-only direct ABCI CheckTx measurement. It publishes no
// transaction, key, signature, or private runtime home.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
	researchv2app "example.com/pluto/internal/experiment/profiles/app"
	researchv2test "example.com/pluto/internal/experiment/profiles/testfixture"
	"example.com/pluto/internal/experiment/telemetry"
)

const chainID = "pluto-file02-r02"
const steadyCount = 240

type sample struct {
	Slot       int              `json:"slot"`
	ServiceNS  int64            `json:"service_ns"`
	LockWaitNS int64            `json:"app_lock_wait_ns"`
	WireBytes  int              `json:"wire_bytes"`
	Spans      []telemetry.Span `json:"spans"`
}

type profileResult struct {
	Name          string   `json:"profile"`
	Code          int      `json:"code"`
	Samples       []sample `json:"samples"`
	RecorderDrops uint64   `json:"recorder_drops"`
	TotalAllocB   uint64   `json:"total_alloc_bytes"`
	TotalMallocs  uint64   `json:"total_mallocs"`
	ProcessCPUNS  int64    `json:"process_cpu_ns"`
	ElapsedNS     int64    `json:"elapsed_ns"`
	ExpectedCount int      `json:"expected_count"`
	GenesisSHA256 string   `json:"genesis_sha256"`
}

type output struct {
	Schema        string          `json:"schema"`
	GoVersion     string          `json:"go_version"`
	GOOS          string          `json:"goos"`
	GOARCH        string          `json:"goarch"`
	Order         []string        `json:"order"`
	ProfileResult []profileResult `json:"profiles"`
}

func cpuNS(r syscall.Rusage) int64 {
	return (r.Utime.Sec+r.Stime.Sec)*int64(time.Second) + (r.Utime.Usec+r.Stime.Usec)*int64(time.Microsecond)
}

func runProfile(f *load.PilotFixture, name string, code researchv2.Profile) (profileResult, error) {
	app, err := researchv2app.New(f.Genesis, f.ChainID, code)
	if err != nil {
		return profileResult{}, err
	}
	recorder := telemetry.NewRecorder(steadyCount + 2)
	app.SetRecorder(recorder)
	fixture := &researchv2test.Fixture{ChainID: f.ChainID}
	makeTx := func(slot int) ([]byte, error) {
		identity := f.Senders[slot]
		return fixture.SignedBy(code, identity, identity.Binding.Account,
			f.Receiver.Binding.Account, 1, 0)
	}
	warmup, err := makeTx(0)
	if err != nil {
		return profileResult{}, err
	}
	ctx := context.Background()
	warmResult, err := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: warmup})
	if err != nil || warmResult == nil || warmResult.Code != 0 {
		return profileResult{}, fmt.Errorf("%s warm-up CheckTx: %v", name, err)
	}
	_, dropped := recorder.Drain()
	if dropped != 0 {
		return profileResult{}, errors.New("warm-up recorder drop")
	}
	transactions := make([][]byte, steadyCount)
	for i := range transactions {
		transactions[i], err = makeTx(i + 1)
		if err != nil {
			return profileResult{}, err
		}
	}
	runtime.GC()
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	var cpuBefore, cpuAfter syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &cpuBefore); err != nil {
		return profileResult{}, err
	}
	started := time.Now()
	for _, raw := range transactions {
		result, err := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: raw})
		if err != nil || result == nil {
			return profileResult{}, fmt.Errorf("%s valid CheckTx failed: %v", name, err)
		}
		if result.Code != 0 || result.Log != string(researchv2.StageAccepted) {
			return profileResult{}, fmt.Errorf("%s valid CheckTx failed: code=%d", name, result.Code)
		}
	}
	elapsed := time.Since(started)
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &cpuAfter); err != nil {
		return profileResult{}, err
	}
	runtime.ReadMemStats(&memAfter)
	events, dropped := recorder.Drain()
	if len(events) != steadyCount || dropped != 0 {
		return profileResult{}, fmt.Errorf("%s events/drops %d/%d", name, len(events), dropped)
	}
	samples := make([]sample, 0, steadyCount)
	for i, event := range events {
		if event.Code != 0 || event.LastStage != string(researchv2.StageAccepted) ||
			event.CheckType != "new" || event.ServiceNS <= 0 || event.Profile != uint8(code) {
			return profileResult{}, fmt.Errorf("%s event %d invalid", name, i)
		}
		samples = append(samples, sample{Slot: i + 1, ServiceNS: event.ServiceNS,
			LockWaitNS: event.AppLockWaitNS, WireBytes: event.WireBytes, Spans: event.Spans})
	}
	h := sha256.Sum256(f.Genesis)
	return profileResult{Name: name, Code: int(code), Samples: samples, RecorderDrops: dropped,
		TotalAllocB:  memAfter.TotalAlloc - memBefore.TotalAlloc,
		TotalMallocs: memAfter.Mallocs - memBefore.Mallocs,
		ProcessCPUNS: cpuNS(cpuAfter) - cpuNS(cpuBefore), ElapsedNS: elapsed.Nanoseconds(),
		ExpectedCount: steadyCount, GenesisSHA256: hex.EncodeToString(h[:])}, nil
}

func run() error {
	wanted := strings.Split(os.Getenv("FILE02_R02_ORDER"), ",")
	if len(wanted) != 4 {
		return errors.New("FILE02_R02_ORDER must list four profiles")
	}
	profiles := map[string]researchv2.Profile{"C": researchv2.ProfileC, "CP": researchv2.ProfileCP,
		"P": researchv2.ProfileP, "H": researchv2.ProfileH}
	seen := map[string]bool{}
	for _, name := range wanted {
		if !profiles[name].Valid() || seen[name] {
			return errors.New("invalid profile permutation")
		}
		seen[name] = true
	}
	f, err := load.NewPilotFixture(chainID, steadyCount+1)
	if err != nil {
		return err
	}
	out := output{Schema: "pluto.file02.r02.raw.v1", GoVersion: runtime.Version(),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Order: wanted}
	for _, name := range wanted {
		result, err := runProfile(f, name, profiles[name])
		if err != nil {
			return err
		}
		out.ProfileResult = append(out.ProfileResult, result)
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
