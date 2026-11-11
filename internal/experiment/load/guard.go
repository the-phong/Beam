package load

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"example.com/pluto/internal/experiment/telemetry"
)

var ErrSafety = errors.New("isolated load safety stop")

type WatchedNode struct {
	RPC          string `json:"rpc"`
	ObserverFile string `json:"observer_file"`
	ObserverPID  int    `json:"observer_pid"`
	CometPID     int    `json:"comet_pid"`
	AppPID       int    `json:"app_pid"`
}

type HostGuard struct {
	config         Config
	nodes          [4]WatchedNode
	dataRoot       string
	killFile       string
	allocatedCores int
	client         *http.Client
	lastCPUAt      time.Time
	lastCPUNS      uint64
	highCPUFrom    time.Time
	processStarts  map[int]uint64
}

func NewHostGuard(c Config, nodes [4]WatchedNode, dataRoot, killFile string, allocatedCores int) (*HostGuard, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if dataRoot == "" || killFile == "" || allocatedCores < 1 || allocatedCores > runtime.NumCPU() {
		return nil, fmt.Errorf("%w: missing safety controls", ErrSafety)
	}
	if info, err := os.Stat(dataRoot); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%w: data root unavailable", ErrSafety)
	}
	seenPID := map[int]bool{}
	appPIDs := map[int]bool{}
	seenFile := map[string]bool{}
	for _, n := range nodes {
		if _, err := NewHTTPSender(n.RPC, time.Second); err != nil || n.ObserverFile == "" || n.ObserverPID <= 0 || n.CometPID <= 0 || n.AppPID <= 0 || n.CometPID == os.Getpid() || n.ObserverPID == os.Getpid() || n.ObserverPID == n.CometPID || n.ObserverPID == n.AppPID || n.CometPID == n.AppPID || seenPID[n.CometPID] || seenPID[n.ObserverPID] || seenFile[n.ObserverFile] {
			return nil, fmt.Errorf("%w: invalid validator/collector identity", ErrSafety)
		}
		seenPID[n.CometPID], seenPID[n.ObserverPID], seenFile[n.ObserverFile] = true, true, true
		appPIDs[n.AppPID] = true
	}
	for appPID := range appPIDs {
		if seenPID[appPID] || appPID == os.Getpid() {
			return nil, fmt.Errorf("%w: application process overlaps load or observer", ErrSafety)
		}
	}
	return &HostGuard{config: c, nodes: nodes, dataRoot: dataRoot, killFile: killFile, allocatedCores: allocatedCores,
		client: &http.Client{Timeout: 1500 * time.Millisecond, Transport: &http.Transport{Proxy: nil}}, processStarts: map[int]uint64{}}, nil
}

func (g *HostGuard) Check(ctx context.Context) error {
	if _, err := os.Stat(g.killFile); err == nil {
		return fmt.Errorf("%w: manual kill switch", ErrSafety)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: kill switch inaccessible", ErrSafety)
	}
	if err := checkDisk(g.dataRoot, g.config.Safety.MinFreeDiskFraction); err != nil {
		return err
	}
	memTotal, memAvailable, err := readMemory()
	if err != nil || float64(memTotal-memAvailable)/float64(memTotal) >= g.config.Safety.MaxRSSFraction {
		return fmt.Errorf("%w: host memory cap or sampler failure", ErrSafety)
	}
	now := time.Now()
	var totalRSS, totalCPU uint64
	countedApps := map[int]bool{}
	var unavailable int
	statusHashes := map[string]string{}
	for _, n := range g.nodes {
		if _, err := os.Stat(fmt.Sprintf("/proc/%d", n.ObserverPID)); err != nil {
			return fmt.Errorf("%w: observer process exited", ErrSafety)
		}
		sample, err := latestSample(n.ObserverFile)
		if err != nil || sample.Comet == nil || sample.Application == nil || sample.Comet.PID != n.CometPID || sample.Application.PID != n.AppPID || sample.Schema != telemetry.Schema || sample.Kind != "node_sample" || sample.DroppedSamples != 0 || len(sample.Unavailable) != 0 {
			return fmt.Errorf("%w: observer unavailable or incomplete", ErrSafety)
		}
		observed, err := time.Parse(time.RFC3339Nano, sample.ObservedUTC)
		if err != nil || now.Sub(observed) < -time.Second || now.Sub(observed) > time.Duration(g.config.Safety.MaxObserverStaleMS)*time.Millisecond {
			return fmt.Errorf("%w: stale observer sample", ErrSafety)
		}
		for _, process := range []*telemetry.ProcessSample{sample.Comet, sample.Application} {
			if first, ok := g.processStarts[process.PID]; ok && first != process.StartTicks {
				return fmt.Errorf("%w: watched process restarted", ErrSafety)
			}
			g.processStarts[process.PID] = process.StartTicks
		}
		totalRSS += sample.Comet.RSSBytes
		totalCPU += sample.Comet.CPU_NS
		if !countedApps[n.AppPID] {
			totalRSS += sample.Application.RSSBytes
			totalCPU += sample.Application.CPU_NS
			countedApps[n.AppPID] = true
		}
		height, hash, err := g.nodeStatus(ctx, n.RPC)
		if err != nil {
			unavailable++
			continue
		}
		if prior, ok := statusHashes[height]; ok && hash != prior {
			return fmt.Errorf("%w: same-height AppHash divergence", ErrSafety)
		}
		statusHashes[height] = hash
	}
	if unavailable >= 2 {
		return fmt.Errorf("%w: two validators unavailable", ErrSafety)
	}
	if float64(totalRSS)/float64(memTotal) >= g.config.Safety.MaxRSSFraction {
		return fmt.Errorf("%w: watched-process RSS cap", ErrSafety)
	}
	if !g.lastCPUAt.IsZero() && totalCPU >= g.lastCPUNS {
		elapsed := now.Sub(g.lastCPUAt).Seconds()
		if elapsed > 0 {
			fraction := float64(totalCPU-g.lastCPUNS) / (elapsed * 1e9 * float64(g.allocatedCores))
			if fraction >= g.config.Safety.MaxCPUFraction {
				if g.highCPUFrom.IsZero() {
					g.highCPUFrom = now
				}
				if now.Sub(g.highCPUFrom) >= time.Duration(g.config.Safety.CPUBreachSeconds)*time.Second {
					return fmt.Errorf("%w: sustained watched-process CPU cap", ErrSafety)
				}
			} else {
				g.highCPUFrom = time.Time{}
			}
		}
	} else if !g.lastCPUAt.IsZero() {
		return fmt.Errorf("%w: process CPU counter reset", ErrSafety)
	}
	g.lastCPUAt, g.lastCPUNS = now, totalCPU
	return nil
}

func latestSample(path string) (telemetry.NodeSample, error) {
	info, err := os.Stat(path)
	if err != nil || info.Size() > 1<<20 {
		return telemetry.NodeSample{}, ErrSafety
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return telemetry.NodeSample{}, err
	}
	end := bytes.LastIndexByte(raw, '\n')
	if end < 0 {
		return telemetry.NodeSample{}, ErrSafety
	}
	start := bytes.LastIndexByte(raw[:end], '\n') + 1
	var sample telemetry.NodeSample
	if err := json.Unmarshal(raw[start:end], &sample); err != nil {
		return telemetry.NodeSample{}, err
	}
	return sample, nil
}

func checkDisk(root string, minimum float64) error {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(root, &fs); err != nil || fs.Blocks == 0 || float64(fs.Bavail)/float64(fs.Blocks) < minimum {
		return fmt.Errorf("%w: free disk below cap or unavailable", ErrSafety)
	}
	return nil
}

func readMemory() (uint64, uint64, error) {
	raw, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	var total, available uint64
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = value * 1024
		case "MemAvailable:":
			available = value * 1024
		}
	}
	if total == 0 || available > total {
		return 0, 0, ErrSafety
	}
	return total, available, nil
}

func (g *HostGuard) nodeStatus(ctx context.Context, origin string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+"/status", nil)
	if err != nil {
		return "", "", err
	}
	response, err := g.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", "", ErrSafety
	}
	var wire struct {
		Result struct {
			SyncInfo struct {
				Height  string `json:"latest_block_height"`
				AppHash string `json:"latest_app_hash"`
			} `json:"sync_info"`
		} `json:"result"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&wire); err != nil || wire.Result.SyncInfo.Height == "" || wire.Result.SyncInfo.AppHash == "" {
		return "", "", ErrSafety
	}
	return wire.Result.SyncInfo.Height, wire.Result.SyncInfo.AppHash, nil
}
