// Package load defines the bounded, isolated S06 workload contract.
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
)

var ErrConfig = errors.New("invalid isolated testbed config")

type NetworkConfig struct {
	ValidatorCount       int    `json:"validator_count"`
	Placement            string `json:"placement"`
	CometVersion         string `json:"comet_version"`
	MempoolType          string `json:"mempool_type"`
	MempoolSize          int    `json:"mempool_size"`
	MempoolMaxBytes      int64  `json:"mempool_max_bytes"`
	MempoolCacheSize     int    `json:"mempool_cache_size"`
	MempoolMaxTxBytes    int    `json:"mempool_max_tx_bytes"`
	MempoolRecheck       bool   `json:"mempool_recheck"`
	MempoolBroadcast     bool   `json:"mempool_broadcast"`
	MaxBlockBytes        int64  `json:"max_block_bytes"`
	CreateEmptyBlocks    bool   `json:"create_empty_blocks"`
	EmptyBlockIntervalMS int    `json:"empty_block_interval_ms"`
	TimeoutCommitMS      int    `json:"timeout_commit_ms"`
	CollectorIntervalMS  int    `json:"collector_interval_ms"`
}

type SafetyConfig struct {
	MaxTxPerSecond      int     `json:"max_tx_per_second"`
	MaxBytesPerSecond   int     `json:"max_bytes_per_second"`
	MaxOutstanding      int     `json:"max_outstanding"`
	MaxDurationSeconds  int     `json:"max_duration_seconds"`
	RequestTimeoutMS    int     `json:"request_timeout_ms"`
	MaxCPUFraction      float64 `json:"max_cpu_fraction"`
	CPUBreachSeconds    int     `json:"cpu_breach_seconds"`
	MaxRSSFraction      float64 `json:"max_rss_fraction"`
	MinFreeDiskFraction float64 `json:"min_free_disk_fraction"`
	MaxObserverStaleMS  int     `json:"max_observer_stale_ms"`
	AllowedRPCHost      string  `json:"allowed_rpc_host"`
}

type Config struct {
	Version int           `json:"version"`
	Network NetworkConfig `json:"network"`
	Safety  SafetyConfig  `json:"safety"`
}

func ReadConfig(path string) (Config, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, "", err
	}
	var c Config
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return Config{}, "", fmt.Errorf("%w: %v", ErrConfig, err)
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Config{}, "", fmt.Errorf("%w: trailing document", ErrConfig)
	}
	if err := c.Validate(); err != nil {
		return Config{}, "", err
	}
	h := sha256.Sum256(raw)
	return c, hex.EncodeToString(h[:]), nil
}

func (c Config) Validate() error {
	n := c.Network
	s := c.Safety
	if c.Version != 1 || n.ValidatorCount != 4 || n.Placement != "single_host_pilot" || n.CometVersion != "v0.38.26" || n.MempoolType != "flood" ||
		n.MempoolSize < 1 || n.MempoolSize > 5000 || n.MempoolMaxBytes < 1 || n.MempoolMaxBytes > 1<<30 || n.MempoolCacheSize < 1 || n.MempoolMaxTxBytes < 1 || n.MempoolMaxTxBytes > 8192 ||
		!n.MempoolRecheck || !n.MempoolBroadcast || n.MaxBlockBytes < 1 || n.MaxBlockBytes > 1<<20 || !n.CreateEmptyBlocks || n.EmptyBlockIntervalMS < 1 || n.TimeoutCommitMS < 1 || n.CollectorIntervalMS < 100 ||
		s.MaxTxPerSecond < 1 || s.MaxTxPerSecond > 100 || s.MaxBytesPerSecond < 1 || s.MaxBytesPerSecond > 1<<20 || s.MaxOutstanding < 1 || s.MaxOutstanding > 64 ||
		s.MaxDurationSeconds < 1 || s.MaxDurationSeconds > 600 || s.RequestTimeoutMS < 1 || s.RequestTimeoutMS > 30000 || s.MaxCPUFraction <= 0 || s.MaxCPUFraction > .90 ||
		s.CPUBreachSeconds < 1 || s.CPUBreachSeconds > 30 || s.MaxRSSFraction <= 0 || s.MaxRSSFraction > .80 || s.MinFreeDiskFraction < .10 || s.MinFreeDiskFraction >= 1 ||
		s.MaxObserverStaleMS < n.CollectorIntervalMS || s.MaxObserverStaleMS > 10000 || s.AllowedRPCHost != "127.0.0.1" {
		return ErrConfig
	}
	return nil
}
