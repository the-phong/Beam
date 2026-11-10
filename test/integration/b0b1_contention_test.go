//go:build integration && fourval

package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/benchmarks/corpus"
)

type contentionConfig struct {
	Profile      string  `json:"profile"`
	ValidRate    float64 `json:"valid_rate_tps"`
	InvalidRate  float64 `json:"invalid_rate_tps"`
	ValidCount   int     `json:"valid_count"`
	InvalidCount int     `json:"invalid_count"`
	Class        string  `json:"class"`
	Seed         int64   `json:"seed"`
	RunID        string  `json:"run_id"`
}

type contentionResult struct {
	phase05Result
	Condition             contentionConfig `json:"condition"`
	ObservedHeight        int64            `json:"observed_height"`
	ScheduledEndUnixNano  int64            `json:"scheduled_end_unix_nano"`
	MeasurementDurationNS int64            `json:"measurement_duration_ns"`
	AllValidatorsMatched  bool             `json:"all_validators_matched"`
}

type contentionScheduled struct {
	at      time.Duration
	valid   bool
	ordinal int
}

func contentionSchedule(c contentionConfig) []contentionScheduled {
	items := make([]contentionScheduled, 0, c.ValidCount+c.InvalidCount)
	for i := 0; i < c.ValidCount; i++ {
		items = append(items, contentionScheduled{time.Duration(float64(i) * float64(time.Second) / c.ValidRate), true, i})
	}
	for i := 0; i < c.InvalidCount; i++ {
		items = append(items, contentionScheduled{time.Duration(float64(i) * float64(time.Second) / c.InvalidRate), false, i})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].at == items[j].at {
			return items[i].valid
		}
		return items[i].at < items[j].at
	})
	return items
}

func TestPhase05ContentionSchedule(t *testing.T) {
	for _, tc := range []struct {
		valid, invalid   float64
		validN, invalidN int
	}{
		{1.5, 0, 300, 0}, {1.5, .5, 300, 100}, {1.5, 1.5, 300, 300}, {1.5, 4.5, 300, 900}, {1.5, 13.5, 300, 2700},
		{0, 15, 0, 3000}, {4, 0, 300, 0}, {4, 4, 300, 300}, {4, 12, 300, 900},
	} {
		cfg := contentionConfig{Profile: "b1", ValidRate: tc.valid, InvalidRate: tc.invalid, ValidCount: tc.validN, InvalidCount: tc.invalidN, Class: "mixed", RunID: "schedule"}
		if err := contentionConfigValid(cfg); err != nil {
			t.Fatal(err)
		}
		items := contentionSchedule(cfg)
		if len(items) != tc.validN+tc.invalidN {
			t.Fatal("count mismatch")
		}
		valid := 0
		for i, item := range items {
			if i > 0 && item.at < items[i-1].at {
				t.Fatal("nonmonotonic schedule")
			}
			if item.valid {
				valid++
			}
		}
		if valid != tc.validN {
			t.Fatal("valid stream count mismatch")
		}
	}
}

func contentionConfigValid(c contentionConfig) error {
	if c.Profile != "b0" && c.Profile != "b1" || c.ValidRate < 0 || c.InvalidRate < 0 || c.ValidRate+c.InvalidRate <= 0 || c.ValidRate+c.InvalidRate > 16 || c.ValidCount < 0 || c.ValidCount > 378 || c.InvalidCount < 0 || c.InvalidCount > 3200 || c.ValidCount+c.InvalidCount < 1 || c.ValidCount > 0 && c.ValidRate == 0 || c.InvalidCount > 0 && c.InvalidRate == 0 || c.ValidCount == 0 && c.ValidRate != 0 || c.InvalidCount == 0 && c.InvalidRate != 0 {
		return fmt.Errorf("unsafe contention configuration")
	}
	if c.RunID == "" || len(c.RunID) > 120 {
		return fmt.Errorf("invalid run id")
	}
	for _, ch := range c.RunID {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
			return fmt.Errorf("invalid run id")
		}
	}
	if c.Class != "structural" && c.Class != "binding" && c.Class != "classical" && c.Class != "ml" && c.Class != "state" && c.Class != "mixed" {
		return fmt.Errorf("invalid class")
	}
	return nil
}

func contentionWrite(t *testing.T, r *contentionResult) {
	t.Helper()
	dir := os.Getenv("PLUTO_PHASE05_RAW_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Error(err)
		return
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		t.Error(err)
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, r.Condition.RunID+".json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Error(err)
		return
	}
	if _, err = f.Write(append(raw, '\n')); err != nil {
		t.Error(err)
	}
	if err = f.Close(); err != nil {
		t.Error(err)
	}
}

// Observe committed blocks through a second validator. This uses one RPC per
// new block, rather than O(pending transactions) /tx polls per timer tick.
func contentionObserve(n *fourNetwork, stop <-chan struct{}, seen chan<- phase05Observed, errors chan<- error) {
	last := int64(0)
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			h, _, err := n.height(1)
			if err != nil {
				continue
			}
			for next := last + 1; next <= h; next++ {
				_, _, txs, err := n.block(1, next)
				if err != nil {
					break
				}
				var results struct {
					Result struct {
						TxsResults []struct {
							Code uint32 `json:"code"`
						} `json:"txs_results"`
					} `json:"result"`
				}
				if len(txs) > 0 {
					if err = rpcGetLong(n.t, n.nodes[1].ports.RPC, fmt.Sprintf("/block_results?height=%d", next), &results); err != nil {
						break
					}
					if len(results.Result.TxsResults) != len(txs) {
						errors <- fmt.Errorf("block %d result count mismatch", next)
						return
					}
				}
				at := time.Now()
				for i, encoded := range txs {
					raw, e := base64.StdEncoding.DecodeString(encoded)
					if e != nil {
						errors <- e
						return
					}
					digest := sha256.Sum256(raw)
					seen <- phase05Observed{hash: strings.ToUpper(hex.EncodeToString(digest[:])), height: next, code: results.Result.TxsResults[i].Code, seenUnixNano: at.UnixNano(), seenAt: at}
				}
				last = next
			}
		}
	}
}

// TestPhase05Contention holds legitimate offered rate constant while adding
// invalid traffic. It runs only on four disposable, real CometBFT validators.
func TestPhase05Contention(t *testing.T) {
	cfg := contentionConfig{Profile: "b1", ValidRate: 1, InvalidRate: 1, ValidCount: 12, InvalidCount: 12, Class: "ml", Seed: 5, RunID: "contention-smoke"}
	if raw := os.Getenv("PLUTO_PHASE05_CONTENTION_CONFIG"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			t.Fatal(err)
		}
	}
	if err := contentionConfigValid(cfg); err != nil {
		t.Fatal(err)
	}
	c, corpusHash, err := corpus.Load("../../research/corpus/b0b1/public_corpus_contention.json")
	if err != nil {
		t.Fatal(err)
	}
	if c.Version != corpus.ContentionVersion {
		t.Fatal("wrong corpus version")
	}
	p, ok := c.Profile(cfg.Profile)
	if !ok {
		t.Fatal("missing profile")
	}
	if err := corpus.CheckOracle(c); err != nil {
		t.Fatal(err)
	}
	n := phase05Network(t, p)
	n.startAll()
	n.waitAll(2, 90*time.Second)
	r := contentionResult{Condition: cfg}
	r.phase05Result = phase05Result{Config: phase05Config{Profile: cfg.Profile, Rate: cfg.ValidRate + cfg.InvalidRate, Count: cfg.ValidCount + cfg.InvalidCount, Class: cfg.Class, Seed: cfg.Seed, RunID: cfg.RunID}, ValidatorCount: 4, AccountCount: p.AccountCount, SourceCommit: phase05Git("HEAD"), SourceTree: phase05StagedTree(), GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CorpusSHA256: corpusHash, GenesisSHA256: p.GenesisSHA256, StartUnixNano: time.Now().UnixNano()}
	phase05Environment(&r.phase05Result, n.root)
	defer contentionWrite(t, &r)
	items := contentionSchedule(cfg)
	validVectors := make([]corpus.Vector, 0, 378)
	for _, v := range p.Valid {
		if strings.HasSuffix(v.ID, "000") {
			continue
		}
		index, e := strconv.Atoi(strings.TrimPrefix(v.ID, "V"))
		if e != nil {
			t.Fatal(e)
		}
		if index%corpus.SenderCount != 0 {
			validVectors = append(validVectors, v)
		}
	}
	if len(validVectors) < cfg.ValidCount {
		t.Fatal("insufficient signed valid vectors")
	}
	seen := make(chan phase05Observed, 1024)
	observerErrors := make(chan error, 1)
	stop := make(chan struct{})
	go contentionObserve(n, stop, seen, observerErrors)
	defer close(stop)
	start := time.Now()
	r.MeasurementStartUnixNano = start.UnixNano()
	r.Samples = append(r.Samples, phase05Samples(n, start)...)
	expectedByHash := make(map[string]int, cfg.ValidCount)
	submittedAt := make(map[string]time.Time, cfg.ValidCount)
	effects := map[string]uint64{}
	for i, item := range items {
		if wait := time.Until(start.Add(item.at)); wait > 0 {
			time.Sleep(wait)
		}
		var raw []byte
		var id string
		var expected uint32
		if item.valid {
			v := validVectors[item.ordinal]
			raw, err = v.Bytes()
			id = v.ID
			effects[v.SenderHex]++
		} else {
			class := cfg.Class
			if class == "mixed" {
				classes := []string{"structural", "binding", "classical", "ml", "state"}
				class = classes[item.ordinal%len(classes)]
				if p.Name == "b0" && class == "binding" {
					class = "classical"
				}
			}
			cs, e := phase05Invalid(p, class)
			if e != nil {
				t.Fatal(e)
			}
			raw, err = cs.Bytes()
			id = cs.ID
			expected = cs.ExpectedCode
		}
		if err != nil {
			t.Fatal(err)
		}
		event := phase05Event{Index: i, CaseID: id, Size: len(raw), Expected: expected}
		begin := time.Now()
		event.SubmitUnixNano = begin.UnixNano()
		event.SubmitOffsetNS = begin.Sub(start).Nanoseconds()
		var response struct {
			Result struct {
				Code uint32 `json:"code"`
				Hash string `json:"hash"`
			} `json:"result"`
			Error any `json:"error"`
		}
		err = rpcGetLong(t, n.nodes[0].ports.RPC, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(raw), &response)
		event.LatencyNS = time.Since(begin).Nanoseconds()
		if err != nil {
			event.Error = err.Error()
		} else if response.Error != nil {
			event.Error = fmt.Sprint(response.Error)
		} else {
			event.Code = response.Result.Code
			event.Hash = response.Result.Hash
		}
		r.Events = append(r.Events, event)
		if event.Error != "" {
			t.Fatalf("submission %d: %s", i, event.Error)
		}
		if event.Code != expected {
			t.Fatalf("%s admission code=%d expected=%d", id, event.Code, expected)
		}
		if item.valid {
			r.ValidSubmitted++
			hash := strings.ToUpper(event.Hash)
			expectedByHash[hash] = i
			submittedAt[hash] = begin
		} else {
			r.InvalidRejected++
		}
		if i%8 == 0 {
			r.Samples = append(r.Samples, phase05Samples(n, time.Now())...)
		}
	}
	r.ScheduledEndUnixNano = start.Add(items[len(items)-1].at).UnixNano()
	deadline := time.NewTimer(45 * time.Second)
	defer deadline.Stop()
	for r.ValidIncluded < r.ValidSubmitted {
		select {
		case obs := <-seen:
			index, ok := expectedByHash[obs.hash]
			if !ok {
				t.Fatalf("unexpected committed tx %s at H%d", obs.hash, obs.height)
			}
			e := &r.Events[index]
			if e.IncludedHeight != 0 || obs.code != 0 {
				t.Fatalf("duplicate or rejected inclusion %s H%d code=%d", e.CaseID, obs.height, obs.code)
			}
			e.IncludedHeight = obs.height
			e.IncludedUnixNano = obs.seenUnixNano
			e.InclusionLatencyNS = obs.seenAt.Sub(submittedAt[obs.hash]).Nanoseconds()
			if e.InclusionLatencyNS <= 0 {
				t.Fatalf("nonpositive monotonic inclusion latency for %s", e.CaseID)
			}
			r.ValidIncluded++
		case err := <-observerErrors:
			t.Fatal(err)
		case <-deadline.C:
			t.Fatalf("included only %d/%d valid", r.ValidIncluded, r.ValidSubmitted)
		}
	}
	r.MeasurementDurationNS = time.Since(start).Nanoseconds()
	r.MeasurementEndUnixNano = time.Now().UnixNano()
	r.Samples = append(r.Samples, phase05Samples(n, time.Now())...)
	lastHeight := int64(2)
	for _, e := range r.Events {
		if e.IncludedHeight > lastHeight {
			lastHeight = e.IncludedHeight
		}
	}
	n.waitAll(lastHeight+1, 90*time.Second)
	r.MatchedHeight = lastHeight
	r.ObservedHeight = lastHeight + 1
	for i := range n.nodes {
		blockHash, _, _, e := n.block(i, lastHeight)
		if e != nil {
			t.Fatal(e)
		}
		_, appHash, _, e := n.block(i, lastHeight+1)
		if e != nil {
			t.Fatal(e)
		}
		r.BlockHashes = append(r.BlockHashes, blockHash)
		r.AppHashes = append(r.AppHashes, appHash)
		if i > 0 && (blockHash != r.BlockHashes[0] || appHash != r.AppHashes[0]) {
			t.Fatalf("validator %d matching-height hash mismatch", i)
		}
	}
	var sequence bytes.Buffer
	for h := int64(1); h <= lastHeight; h++ {
		referenceHash, _, referenceTxs, e := n.block(0, h)
		if e != nil {
			t.Fatal(e)
		}
		fmt.Fprintf(&sequence, "%d:%s:%v\n", h, referenceHash, referenceTxs)
		for i := 1; i < 4; i++ {
			hash, _, txs, e := n.block(i, h)
			if e != nil {
				t.Fatal(e)
			}
			if hash != referenceHash || !slices.Equal(txs, referenceTxs) {
				t.Fatalf("validator %d block mismatch H%d", i, h)
			}
		}
	}
	digest := sha256.Sum256(sequence.Bytes())
	for range n.nodes {
		r.OrderedTxDigests = append(r.OrderedTxDigests, hex.EncodeToString(digest[:]))
	}
	for i := range n.nodes {
		var supply uint64
		for sender := 0; sender < corpus.SenderCount; sender++ {
			v := p.Valid[sender]
			raw, e := hex.DecodeString(v.SenderHex)
			if e != nil {
				t.Fatal(e)
			}
			var addr b0state.Address
			copy(addr[:], raw)
			var balance, nonce uint64
			if p.Name == "b0" {
				balance, nonce, e = n.account(i, addr)
			} else {
				balance, nonce, e = hybridAccount(n, i, addr)
			}
			if e != nil {
				t.Fatal(e)
			}
			want := effects[v.SenderHex]
			if balance != 100-want || nonce != want {
				t.Fatalf("validator %d sender %d balance/nonce=%d/%d want=%d/%d", i, sender, balance, nonce, 100-want, want)
			}
			supply += balance
			if sender == 1 {
				r.SenderBalances = append(r.SenderBalances, balance)
				r.SenderNonces = append(r.SenderNonces, nonce)
			}
		}
		raw, e := hex.DecodeString(p.Valid[0].ReceiverHex)
		if e != nil {
			t.Fatal(e)
		}
		var receiver b0state.Address
		copy(receiver[:], raw)
		var balance, nonce uint64
		if p.Name == "b0" {
			balance, nonce, e = n.account(i, receiver)
		} else {
			balance, nonce, e = hybridAccount(n, i, receiver)
		}
		if e != nil {
			t.Fatal(e)
		}
		initial := uint64(0)
		if p.Name == "b0" {
			initial = 1
		}
		if balance != initial+uint64(cfg.ValidCount) || nonce != 0 {
			t.Fatalf("validator %d receiver balance/nonce=%d/%d", i, balance, nonce)
		}
		r.ReceiverBalances = append(r.ReceiverBalances, balance)
		r.ReceiverNonces = append(r.ReceiverNonces, nonce)
		supply += balance
		r.TotalSupplies = append(r.TotalSupplies, supply)
		if supply != uint64(corpus.SenderCount*100)+initial {
			t.Fatalf("validator %d supply=%d", i, supply)
		}
	}
	r.AllValidatorsMatched = true
	r.EndUnixNano = time.Now().UnixNano()
	r.Complete = true
	t.Logf("contention run=%s valid=%d/%d invalid_rejected=%d H%d AppHash=%s", cfg.RunID, r.ValidIncluded, cfg.ValidCount, r.InvalidRejected, lastHeight+1, r.AppHashes[0])
}
