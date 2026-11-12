//go:build integration && fourval

package integration

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"example.com/pluto/internal/experiment/load"
)

// file02ObserveCommitsByBlock bounds first all-four observation without one
// RPC request per transaction. It polls each new block once per validator.
// The observation is an upper bound on inclusion time, not a consensus clock.
func file02ObserveCommitsByBlock(pilot *s06Network, plan load.Plan, stop <-chan struct{}, output chan<- []s07CommitObservation) {
	type pending struct {
		event  load.PlannedTx
		seen   [4]bool
		height int64
		latest time.Time
	}
	items := make(map[string]*pending)
	ordered := make([]string, 0)
	for _, event := range plan.Events {
		if event.Traffic == "legitimate" {
			items[event.SHA256] = &pending{event: event}
			ordered = append(ordered, event.SHA256)
		}
	}
	completed := make(map[string]s07CommitObservation)
	result := func() []s07CommitObservation {
		out := make([]s07CommitObservation, 0, len(completed))
		for _, hash := range ordered {
			if item, ok := completed[hash]; ok {
				out = append(out, item)
			}
		}
		return out
	}
	var nextHeight [4]int64
	for i := range nextHeight {
		nextHeight[i] = 1
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			output <- result()
			return
		case <-ticker.C:
			for node := range nextHeight {
				top, _, err := pilot.n.height(node)
				if err != nil {
					continue
				}
				for height := nextHeight[node]; height <= top; height++ {
					_, _, txs, err := pilot.n.block(node, height)
					if err != nil {
						break
					}
					observed := time.Now()
					for _, encoded := range txs {
						raw, err := base64.StdEncoding.DecodeString(encoded)
						if err != nil {
							continue
						}
						sum := sha256.Sum256(raw)
						hash := hex.EncodeToString(sum[:])
						item, ok := items[hash]
						if !ok || item.seen[node] {
							continue
						}
						if item.height == 0 || item.height == height {
							item.seen[node], item.height, item.latest = true, height, observed
							if item.seen == [4]bool{true, true, true, true} {
								completed[hash] = s07CommitObservation{ID: item.event.ID, SHA256: hash,
									AllFourSeenUTC: item.latest.UTC().Format(time.RFC3339Nano), Height: height}
							}
						}
					}
					nextHeight[node] = height + 1
				}
			}
			if len(completed) == len(items) {
				output <- result()
				return
			}
		}
	}
}
