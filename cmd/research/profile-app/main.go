// profile-app runs one controlled-profile ABCI application in its own process.
// It is a measurement-only experiment entry point, never a public node.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cometbft/cometbft/abci/server"

	researchv2 "example.com/pluto/internal/experiment/profiles"
	researchv2app "example.com/pluto/internal/experiment/profiles/app"
	"example.com/pluto/internal/experiment/telemetry"
)

type options struct {
	dbDir, genesisPath, chainID, profile, listen, resourcesPath, eventsPath string
	recorder, stageCPU                                                      bool
}

func parse(args []string) (options, error) {
	var o options
	f := flag.NewFlagSet("profile-app", flag.ContinueOnError)
	f.StringVar(&o.dbDir, "db", "", "existing private application database")
	f.StringVar(&o.genesisPath, "genesis", "", "private genesis binary path")
	f.StringVar(&o.chainID, "chain-id", "", "chain ID")
	f.StringVar(&o.profile, "profile", "", "C, CP, P or H")
	f.StringVar(&o.listen, "listen", "", "loopback tcp ABCI address")
	f.StringVar(&o.resourcesPath, "resources", "", "exclusive private resource JSONL")
	f.StringVar(&o.eventsPath, "events", "", "exclusive private CheckTx JSONL")
	f.BoolVar(&o.recorder, "recorder", false, "enable bounded CheckTx recorder")
	f.BoolVar(&o.stageCPU, "stage-cpu", false, "measure instrumented per-stage thread CPU")
	if err := f.Parse(args); err != nil {
		return o, err
	}
	if f.NArg() != 0 || o.dbDir == "" || o.genesisPath == "" || o.chainID == "" || o.resourcesPath == "" || o.eventsPath == "" {
		return o, errors.New("missing required argument")
	}
	if o.stageCPU && !o.recorder {
		return o, errors.New("stage CPU profiling requires recorder")
	}
	if _, err := profileCode(o.profile); err != nil {
		return o, err
	}
	if !strings.HasPrefix(o.listen, "tcp://127.0.0.1:") {
		return o, errors.New("ABCI listener must use 127.0.0.1 loopback TCP")
	}
	port, err := strconv.Atoi(strings.TrimPrefix(o.listen, "tcp://127.0.0.1:"))
	if err != nil || port < 1 || port > 65535 {
		return o, errors.New("invalid loopback TCP port")
	}
	return o, nil
}

func profileCode(name string) (researchv2.Profile, error) {
	switch name {
	case "C":
		return researchv2.ProfileC, nil
	case "CP":
		return researchv2.ProfileCP, nil
	case "P":
		return researchv2.ProfileP, nil
	case "H":
		return researchv2.ProfileH, nil
	default:
		return 0, errors.New("unknown controlled profile")
	}
}

type resourceSample struct {
	Schema         string `json:"schema"`
	UTC            string `json:"utc"`
	PID            int    `json:"pid"`
	UserCPUNS      int64  `json:"user_cpu_ns"`
	SystemCPUNS    int64  `json:"system_cpu_ns"`
	RSSPeakBytes   int64  `json:"rss_peak_bytes"`
	HeapAllocBytes uint64 `json:"heap_alloc_bytes"`
	GCCount        uint32 `json:"gc_count"`
	GCPauseNS      uint64 `json:"gc_pause_total_ns"`
	Goroutines     int    `json:"goroutines"`
	Threads        int    `json:"threads"`
	ThreadStatus   string `json:"thread_status"`
	RecorderDrops  uint64 `json:"recorder_drops"`
}

func threadCount() (int, string) {
	raw, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0, "UNAVAILABLE_PROC_STATUS"
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "Threads:") {
			count, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Threads:")))
			if err == nil {
				return count, "AVAILABLE"
			}
		}
	}
	return 0, "UNAVAILABLE_PROC_STATUS"
}

func sample(drops uint64) resourceSample {
	var use syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &use)
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	threads, threadStatus := threadCount()
	return resourceSample{
		Schema: "pluto.file02.app_process_resources.v1", UTC: time.Now().UTC().Format(time.RFC3339Nano),
		PID: os.Getpid(), UserCPUNS: use.Utime.Nano(), SystemCPUNS: use.Stime.Nano(),
		RSSPeakBytes: use.Maxrss * 1024, HeapAllocBytes: mem.HeapAlloc,
		GCCount: mem.NumGC, GCPauseNS: mem.PauseTotalNs,
		Goroutines: runtime.NumGoroutine(), Threads: threads, ThreadStatus: threadStatus,
		RecorderDrops: drops,
	}
}

func run(o options) error {
	profile, _ := profileCode(o.profile)
	genesis, err := os.ReadFile(o.genesisPath)
	if err != nil {
		return err
	}
	app, err := researchv2app.Open(o.dbDir, genesis, o.chainID, profile)
	if err != nil {
		return err
	}
	defer app.Close()
	resourceFile, err := os.OpenFile(o.resourcesPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer resourceFile.Close()
	eventFile, err := os.OpenFile(o.eventsPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer eventFile.Close()
	var recorder *telemetry.Recorder
	if o.recorder {
		recorder = telemetry.NewRecorder(8192)
		app.SetRecorder(recorder)
	}
	app.SetStageCPUProfiler(o.stageCPU)
	srv, err := server.NewServer(o.listen, "socket", app)
	if err != nil {
		return err
	}
	if err := srv.Start(); err != nil {
		return err
	}
	defer srv.Stop()
	resources := json.NewEncoder(resourceFile)
	flush := func() error {
		var drops uint64
		if recorder != nil {
			events, missed := recorder.Drain()
			drops = missed
			if err := telemetry.WriteJSONL(eventFile, events); err != nil {
				return err
			}
		}
		return resources.Encode(sample(drops))
	}
	if err := flush(); err != nil {
		return err
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := flush(); err != nil {
				return err
			}
		case <-stop:
			if err := flush(); err != nil {
				return err
			}
			if err := resourceFile.Sync(); err != nil {
				return err
			}
			return eventFile.Sync()
		}
	}
}

func main() {
	o, err := parse(os.Args[1:])
	if err == nil {
		err = run(o)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
