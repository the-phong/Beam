//go:build integration && fourval

package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/node"
	"example.com/pluto/internal/phase05/corpus"
)

type phase05Config struct {
	Profile        string  `json:"profile"`
	InvalidPercent int     `json:"invalid_percent"`
	Rate           float64 `json:"rate_tps"`
	Count          int     `json:"count"`
	Seed           int64   `json:"seed"`
	Class          string  `json:"class"`
	RunID          string  `json:"run_id"`
}
type phase05Event struct {
	Index              int    `json:"index"`
	CaseID             string `json:"case_id"`
	Size               int    `json:"bytes"`
	Code               uint32 `json:"code"`
	Expected           uint32 `json:"expected_code"`
	LatencyNS          int64  `json:"admission_latency_ns"`
	SubmitUnixNano     int64  `json:"submit_unix_nano"`
	Hash               string `json:"comet_hash"`
	IncludedHeight     int64  `json:"included_height,omitempty"`
	InclusionLatencyNS int64  `json:"inclusion_latency_ns,omitempty"`
	IncludedUnixNano   int64  `json:"included_unix_nano,omitempty"`
	Error              string `json:"error,omitempty"`
}
type phase05ProcessSample struct {
	UnixNano  int64  `json:"unix_nano"`
	Validator int    `json:"validator"`
	Role      string `json:"role"`
	PID       int    `json:"pid"`
	CPUTicks  uint64 `json:"cpu_ticks"`
	UserTicks uint64 `json:"user_cpu_ticks"`
	SysTicks  uint64 `json:"system_cpu_ticks"`
	RSSBytes  uint64 `json:"rss_bytes"`
}
type phase05Result struct {
	Config                   phase05Config          `json:"config"`
	ValidatorCount           int                    `json:"validator_count"`
	AccountCount             int                    `json:"account_count"`
	SourceCommit             string                 `json:"source_commit"`
	SourceTree               string                 `json:"source_tree"`
	GoVersion                string                 `json:"go_version"`
	GOOS                     string                 `json:"goos"`
	GOARCH                   string                 `json:"goarch"`
	CometVersion             string                 `json:"cometbft_version"`
	CIRCLVersion             string                 `json:"circl_version"`
	Kernel                   string                 `json:"kernel"`
	CPUModel                 string                 `json:"cpu_model"`
	LogicalCPUs              int                    `json:"logical_cpus"`
	MemoryBytes              uint64                 `json:"memory_bytes"`
	WSL2                     bool                   `json:"wsl2"`
	StorageNote              string                 `json:"storage_note"`
	SourceFS                 string                 `json:"source_filesystem"`
	HomeFS                   string                 `json:"home_filesystem"`
	CorpusSHA256             string                 `json:"corpus_sha256"`
	GenesisSHA256            string                 `json:"genesis_sha256"`
	StartUnixNano            int64                  `json:"start_unix_nano"`
	MeasurementStartUnixNano int64                  `json:"measurement_start_unix_nano"`
	MeasurementEndUnixNano   int64                  `json:"measurement_end_unix_nano"`
	EndUnixNano              int64                  `json:"end_unix_nano"`
	Events                   []phase05Event         `json:"events"`
	Samples                  []phase05ProcessSample `json:"process_samples"`
	MatchedHeight            int64                  `json:"matched_height"`
	BlockHashes              []string               `json:"block_hashes"`
	AppHashes                []string               `json:"app_hashes"`
	OrderedTxDigests         []string               `json:"ordered_tx_digests"`
	SenderBalances           []uint64               `json:"sender_balances"`
	SenderNonces             []uint64               `json:"sender_nonces"`
	ReceiverBalances         []uint64               `json:"receiver_balances"`
	ReceiverNonces           []uint64               `json:"receiver_nonces"`
	TotalSupplies            []uint64               `json:"total_supplies"`
	ValidSubmitted           int                    `json:"valid_submitted"`
	ValidIncluded            int                    `json:"valid_included"`
	InvalidRejected          int                    `json:"invalid_rejected"`
	Complete                 bool                   `json:"complete"`
}

func phase05Git(arg string) string {
	out, err := exec.Command("git", "rev-parse", arg).Output()
	if err != nil {
		return "unavailable"
	}
	return strings.TrimSpace(string(out))
}

func phase05StagedTree() string {
	out, err := exec.Command("git", "write-tree").Output()
	if err != nil {
		return "unavailable"
	}
	return strings.TrimSpace(string(out))
}

func phase05Filesystem(path string) string {
	out, err := exec.Command("stat", "-f", "-c", "%T", path).Output()
	if err != nil {
		return "unavailable"
	}
	return strings.TrimSpace(string(out))
}

func phase05Environment(result *phase05Result, home string) {
	result.LogicalCPUs = runtime.NumCPU()
	result.CometVersion = "v0.38.26"
	result.CIRCLVersion = "v1.6.4"
	result.SourceFS = phase05Filesystem("../../research/corpus/phase_05")
	result.HomeFS = phase05Filesystem(home)
	if raw, err := os.ReadFile("/proc/version"); err == nil {
		result.Kernel = strings.TrimSpace(string(raw))
		result.WSL2 = strings.Contains(strings.ToLower(result.Kernel), "microsoft")
	}
	if raw, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, "model name") {
				_, value, _ := strings.Cut(line, ":")
				result.CPUModel = strings.TrimSpace(value)
				break
			}
		}
	}
	if raw, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					n, _ := strconv.ParseUint(fields[1], 10, 64)
					result.MemoryBytes = n * 1024
				}
				break
			}
		}
	}
	result.StorageNote = "corpus loaded from repository path; disposable validator homes under Go t.TempDir (TMPDIR)"
}

func phase05WriteResult(t *testing.T, result *phase05Result) {
	t.Helper()
	dir := os.Getenv("PLUTO_PHASE05_RAW_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Error(err)
		return
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Error(err)
		return
	}
	path := filepath.Join(dir, result.Config.RunID+".json")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Error(err)
		return
	}
	if _, err := f.Write(append(raw, '\n')); err != nil {
		t.Error(err)
	}
	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

func phase05Network(t *testing.T, p corpus.Profile) *fourNetwork {
	t.Helper()
	genesis, err := p.Genesis()
	if err != nil {
		t.Fatal(err)
	}
	n := &fourNetwork{t: t, root: filepath.Join(t.TempDir(), "network"), genesis: genesis}
	n.plutod, _ = filepath.Abs("../../build/plutod")
	comet, _ := filepath.Abs("../../build/cometbft")
	var ports [4]node.B0Ports
	used := map[int]bool{}
	for i := range n.nodes {
		for _, ptr := range []*int{&ports[i].RPC, &ports[i].P2P, &ports[i].ABCI} {
			for {
				*ptr = freePort(t)
				if !used[*ptr] {
					used[*ptr] = true
					break
				}
			}
		}
		n.nodes[i].ports = ports[i]
		n.nodes[i].home = filepath.Join(n.root, fmt.Sprintf("validator-%d", i))
	}
	if p.Name == "b0" {
		err = node.InitB0FourValidators(n.root, comet, p.ChainID, genesis, ports, io.Discard)
	} else {
		err = node.InitB1FourValidators(n.root, comet, p.ChainID, genesis, ports, io.Discard)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(n.saveLogs)
	t.Cleanup(n.stopAll)
	return n
}

func phase05Proc(pid int) (uint64, uint64, uint64, bool) {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, 0, 0, false
	}
	closing := bytes.LastIndexByte(raw, ')')
	if closing < 0 {
		return 0, 0, 0, false
	}
	fields := strings.Fields(string(raw[closing+2:]))
	if len(fields) < 22 {
		return 0, 0, 0, false
	}
	u, e1 := strconv.ParseUint(fields[11], 10, 64)
	s, e2 := strconv.ParseUint(fields[12], 10, 64)
	rss, e3 := strconv.ParseUint(fields[21], 10, 64)
	if e1 != nil || e2 != nil || e3 != nil {
		return 0, 0, 0, false
	}
	return u, s, rss * uint64(os.Getpagesize()), true
}

func phase05Children(pid int) []int {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children", pid, pid))
	if err != nil {
		return nil
	}
	var result []int
	for _, field := range strings.Fields(string(raw)) {
		id, e := strconv.Atoi(field)
		if e == nil {
			result = append(result, id)
		}
	}
	return result
}

func phase05Samples(n *fourNetwork, at time.Time) []phase05ProcessSample {
	var out []phase05ProcessSample
	for i, item := range n.nodes {
		if item.pid <= 0 {
			continue
		}
		pids := append([]int{item.pid}, phase05Children(item.pid)...)
		for j, pid := range pids {
			user, system, rss, ok := phase05Proc(pid)
			if !ok {
				continue
			}
			role := "plutod"
			if j > 0 {
				role = "cometbft_child"
			}
			out = append(out, phase05ProcessSample{UnixNano: at.UnixNano(), Validator: i, Role: role, PID: pid, CPUTicks: user + system, UserTicks: user, SysTicks: system, RSSBytes: rss})
		}
	}
	return out
}

func phase05Invalid(p corpus.Profile, class string) (corpus.Case, error) {
	// S04 is nonempty, so CometBFT can pass it to ABCI CheckTx. S01 is
	// exercised by the direct oracle, where an RPC empty-transaction guard
	// cannot hide the application's structural rejection.
	ids := map[string]string{"structural": "S04", "binding": "I01", "classical": "P03", "ml": "P04", "state": "A03"}
	if p.Name == "b0" {
		ids["ml"] = "P02"
		ids["state"] = "A03"
	}
	id := ids[class]
	if id == "" {
		return corpus.Case{}, fmt.Errorf("unknown class %s", class)
	}
	cs, ok := p.Case(id)
	if !ok {
		return corpus.Case{}, fmt.Errorf("missing class %s case %s", class, id)
	}
	return cs, nil
}

type phase05Pending struct {
	index          int
	hash           string
	submitUnixNano int64
}
type phase05Observed struct {
	index        int
	height       int64
	code         uint32
	seenUnixNano int64
}

// A separate validator's RPC is polled throughout the submission window.
// This avoids delaying early transactions' inclusion observations until the
// whole workload has finished. Polling resolution and RPC overhead remain
// explicit measurement limits.
func phase05Observe(port int, submissions <-chan phase05Pending, observations chan<- phase05Observed, stop <-chan struct{}) {
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	pending := map[int]phase05Pending{}
	for {
		select {
		case item := <-submissions:
			pending[item.index] = item
		case <-ticker.C:
			for index, item := range pending {
				request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/tx?hash=0x%s", port, item.hash), nil)
				if err != nil {
					continue
				}
				response, err := client.Do(request)
				if err != nil {
					continue
				}
				var payload struct {
					Result struct {
						Height   string `json:"height"`
						TxResult struct {
							Code uint32 `json:"code"`
						} `json:"tx_result"`
					} `json:"result"`
					Error any `json:"error"`
				}
				decodeErr := json.NewDecoder(response.Body).Decode(&payload)
				_ = response.Body.Close()
				if response.StatusCode != http.StatusOK || decodeErr != nil || payload.Error != nil {
					continue
				}
				height, err := strconv.ParseInt(payload.Result.Height, 10, 64)
				if err != nil || height < 1 {
					continue
				}
				observations <- phase05Observed{index: index, height: height, code: payload.Result.TxResult.Code, seenUnixNano: time.Now().UnixNano()}
				delete(pending, index)
			}
		case <-stop:
			return
		}
	}
}

// TestPhase05Workload is a bounded real-P2P benchmark. A failed assertion
// fails the test and leaves its raw test output available to the caller.
func TestPhase05Workload(t *testing.T) {
	config := phase05Config{Profile: "b1", InvalidPercent: 50, Rate: 2, Count: 12, Seed: 5, Class: "ml", RunID: "smoke"}
	if raw := os.Getenv("PLUTO_PHASE05_CONFIG"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &config); err != nil {
			t.Fatal(err)
		}
	}
	if config.Profile != "b0" && config.Profile != "b1" || config.InvalidPercent < 0 || config.InvalidPercent > 100 || config.Rate <= 0 || config.Rate > 16 || config.Count < 1 || config.Count > corpus.SenderCount-1 || config.RunID == "" {
		t.Fatal("unsafe benchmark configuration")
	}
	for _, ch := range config.RunID {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			t.Fatal("unsafe run ID")
		}
	}
	c, corpusHash, err := corpus.Load("../../research/corpus/phase_05/public_corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	p, ok := c.Profile(config.Profile)
	if !ok {
		t.Fatal("missing profile")
	}
	if err := corpus.CheckOracle(c); err != nil {
		t.Fatal(err)
	}
	n := phase05Network(t, p)
	n.startAll()
	n.waitAll(2, 90*time.Second)
	result := phase05Result{Config: config, ValidatorCount: 4, AccountCount: p.AccountCount, SourceCommit: phase05Git("HEAD"), SourceTree: phase05StagedTree(), GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CorpusSHA256: corpusHash, GenesisSHA256: p.GenesisSHA256, StartUnixNano: time.Now().UnixNano()}
	phase05Environment(&result, n.root)
	defer phase05WriteResult(t, &result)
	invalidCount := (config.Count*config.InvalidPercent + 50) / 100
	schedule := make([]bool, config.Count)
	for i := 0; i < invalidCount; i++ {
		schedule[i] = true
	}
	rng := rand.New(rand.NewSource(config.Seed))
	rng.Shuffle(len(schedule), func(i, j int) { schedule[i], schedule[j] = schedule[j], schedule[i] })
	period := time.Duration(float64(time.Second) / config.Rate)
	started := time.Now()
	result.MeasurementStartUnixNano = started.UnixNano()
	submissions := make(chan phase05Pending, 64)
	observations := make(chan phase05Observed, 64)
	stopObserver := make(chan struct{})
	go phase05Observe(n.nodes[1].ports.RPC, submissions, observations, stopObserver)
	defer close(stopObserver)
	// V000 is reserved for state-invalid probes (A03). Using it as a
	// successful transfer would change A03's first-failure code from balance
	// to nonce after commit, making an offered workload state-dependent.
	validIndex := 1
	result.Samples = append(result.Samples, phase05Samples(n, time.Now())...)
	for i, isInvalid := range schedule {
		deadline := started.Add(time.Duration(i) * period)
		if wait := time.Until(deadline); wait > 0 {
			time.Sleep(wait)
		}
		var raw []byte
		var id string
		var expected uint32
		if isInvalid {
			class := config.Class
			if class == "mixed" {
				classes := []string{"structural", "binding", "classical", "ml", "state"}
				class = classes[i%len(classes)]
				if p.Name == "b0" && class == "binding" {
					class = "classical"
				}
			}
			cs, e := phase05Invalid(p, class)
			if e != nil {
				t.Fatal(e)
			}
			raw, e = cs.Bytes()
			if e != nil {
				t.Fatal(e)
			}
			id, expected = cs.ID, cs.ExpectedCode
		} else {
			v := p.Valid[validIndex]
			validIndex++
			raw, err = v.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			id = v.ID
			result.ValidSubmitted++
		}
		event := phase05Event{Index: i, CaseID: id, Size: len(raw), Expected: expected}
		start := time.Now()
		event.SubmitUnixNano = start.UnixNano()
		var response struct {
			Result struct {
				Code uint32 `json:"code"`
				Hash string `json:"hash"`
			} `json:"result"`
			Error any `json:"error"`
		}
		err := rpcGetLong(t, n.nodes[0].ports.RPC, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(raw), &response)
		event.LatencyNS = time.Since(start).Nanoseconds()
		if err != nil {
			event.Error = err.Error()
			result.Events = append(result.Events, event)
			t.Fatalf("submission %d: %v", i, err)
		}
		if response.Error != nil {
			event.Error = fmt.Sprint(response.Error)
			result.Events = append(result.Events, event)
			t.Fatalf("RPC submission %d: %v", i, response.Error)
		}
		event.Code, event.Hash = response.Result.Code, response.Result.Hash
		result.Events = append(result.Events, event)
		if event.Code != expected {
			t.Fatalf("%s expected code %d got %d", id, expected, event.Code)
		}
		if expected == 0 {
			submissions <- phase05Pending{index: i, hash: event.Hash, submitUnixNano: event.SubmitUnixNano}
		}
		if isInvalid {
			result.InvalidRejected++
		}
		result.Samples = append(result.Samples, phase05Samples(n, time.Now())...)
	}
	if result.InvalidRejected != invalidCount || result.ValidSubmitted != config.Count-invalidCount {
		t.Fatal("workload count mismatch")
	}
	// Rejected requests have no finality. The observer began polling as soon
	// as each valid admission succeeded, including during later submissions.
	deadline := time.NewTimer(35 * time.Second)
	defer deadline.Stop()
	for result.ValidIncluded < result.ValidSubmitted {
		select {
		case observed := <-observations:
			event := &result.Events[observed.index]
			if event.Expected != 0 || event.IncludedHeight != 0 || observed.code != 0 {
				t.Fatalf("invalid inclusion observation for %s", event.CaseID)
			}
			event.IncludedHeight = observed.height
			event.IncludedUnixNano = observed.seenUnixNano
			event.InclusionLatencyNS = observed.seenUnixNano - event.SubmitUnixNano
			result.ValidIncluded++
		case <-deadline.C:
			t.Fatalf("only %d/%d valid transactions observed included", result.ValidIncluded, result.ValidSubmitted)
		}
	}
	result.Samples = append(result.Samples, phase05Samples(n, time.Now())...)
	result.MeasurementEndUnixNano = time.Now().UnixNano()
	if result.ValidIncluded != result.ValidSubmitted {
		t.Fatal("valid inclusion count mismatch")
	}
	minHeight := int64(2)
	for _, event := range result.Events {
		if event.IncludedHeight > minHeight {
			minHeight = event.IncludedHeight
		}
	}
	n.waitAll(minHeight+1, 90*time.Second)
	result.MatchedHeight = minHeight
	for i := range n.nodes {
		blockHash, _, _, e := n.block(i, minHeight)
		if e != nil {
			t.Fatal(e)
		}
		_, appHash, _, e := n.block(i, minHeight+1)
		if e != nil {
			t.Fatal(e)
		}
		result.BlockHashes = append(result.BlockHashes, blockHash)
		result.AppHashes = append(result.AppHashes, appHash)
		if blockHash != result.BlockHashes[0] || appHash != result.AppHashes[0] {
			t.Fatalf("validator %d diverged at matching H%d", i, minHeight)
		}
		var sequence bytes.Buffer
		for height := int64(1); height <= minHeight; height++ {
			bh, _, txs, e := n.block(i, height)
			if e != nil {
				t.Fatal(e)
			}
			if i > 0 {
				refHash, _, refTxs, e := n.block(0, height)
				if e != nil {
					t.Fatal(e)
				}
				if bh != refHash || !slices.Equal(txs, refTxs) {
					t.Fatalf("validator %d differs in ordered block H%d", i, height)
				}
			}
			fmt.Fprintf(&sequence, "%d:%s:%v\n", height, bh, txs)
		}
		sum := sha256.Sum256(sequence.Bytes())
		result.OrderedTxDigests = append(result.OrderedTxDigests, hex.EncodeToString(sum[:]))
	}
	firstSender, _ := hex.DecodeString(p.Valid[1].SenderHex)
	receiver, _ := hex.DecodeString(p.Valid[0].ReceiverHex)
	var senderAddr, receiverAddr b0state.Address
	copy(senderAddr[:], firstSender)
	copy(receiverAddr[:], receiver)
	for i := range n.nodes {
		var sb, sn, rb, rn uint64
		if p.Name == "b0" {
			sb, sn, err = n.account(i, senderAddr)
			if err != nil {
				t.Fatal(err)
			}
			rb, rn, err = n.account(i, receiverAddr)
		} else {
			sb, sn, err = hybridAccount(n, i, senderAddr)
			if err != nil {
				t.Fatal(err)
			}
			rb, rn, err = hybridAccount(n, i, receiverAddr)
		}
		if err != nil {
			t.Fatal(err)
		}
		result.SenderBalances = append(result.SenderBalances, sb)
		result.SenderNonces = append(result.SenderNonces, sn)
		result.ReceiverBalances = append(result.ReceiverBalances, rb)
		result.ReceiverNonces = append(result.ReceiverNonces, rn)
		if i > 0 && (sb != result.SenderBalances[0] || sn != result.SenderNonces[0] || rb != result.ReceiverBalances[0] || rn != result.ReceiverNonces[0]) {
			t.Fatal("account divergence")
		}
		var supply uint64
		for j, v := range p.Valid {
			addressBytes, _ := hex.DecodeString(v.SenderHex)
			var address b0state.Address
			copy(address[:], addressBytes)
			var balance, nonce uint64
			if p.Name == "b0" {
				balance, nonce, err = n.account(i, address)
			} else {
				balance, nonce, err = hybridAccount(n, i, address)
			}
			if err != nil {
				t.Fatal(err)
			}
			supply += balance
			if j > 0 && j <= result.ValidSubmitted {
				if balance != 99 || nonce != 1 {
					t.Fatalf("validator %d sender %s effect repeated or missing: %d/%d", i, v.ID, balance, nonce)
				}
			} else if balance != 100 || nonce != 0 {
				t.Fatalf("validator %d untouched sender %s changed: %d/%d", i, v.ID, balance, nonce)
			}
		}
		supply += rb
		result.TotalSupplies = append(result.TotalSupplies, supply)
		expectedSupply := uint64(corpus.SenderCount * 100)
		if p.Name == "b0" {
			expectedSupply++
		}
		if supply != expectedSupply {
			t.Fatalf("validator %d supply=%d expected=%d", i, supply, expectedSupply)
		}
	}
	result.Samples = append(result.Samples, phase05Samples(n, time.Now())...)
	result.EndUnixNano = time.Now().UnixNano()
	result.Complete = true
	t.Logf("phase05 run=%s profile=%s invalid=%d/%d valid_included=%d height=%d apphash=%s", config.RunID, p.Name, invalidCount, config.Count, result.ValidIncluded, minHeight, result.AppHashes[0])
}
