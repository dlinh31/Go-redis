package main

import (
	"sync"
	"time"
)

// ExpirationConfig holds configuration parameters for active expiration.
type ExpirationConfig struct {
	Enabled           bool          // Enable/disable active expiration
	Interval          time.Duration // How often to run cycle (default: 100ms)
	SampleSize        int           // Keys to sample per iteration (default: 20)
	AdaptiveThreshold float64       // Stop if expired% < this (default: 0.10)
	MaxIterations     int           // Safety limit per cycle (default: 10)
}

// DefaultExpirationConfig returns sensible defaults matching Redis behavior.
func DefaultExpirationConfig() ExpirationConfig {
	return ExpirationConfig{
		Enabled:           true,
		Interval:          100 * time.Millisecond, // 10 Hz like Redis
		SampleSize:        20,                      // Redis default
		AdaptiveThreshold: 0.10,                    // Stop if <10% expired
		MaxIterations:     10,                      // Prevent infinite loops
	}
}

// ExpirationStats tracks basic statistics about the expiration process.
type ExpirationStats struct {
	CyclesRun   uint64 // Total cycles executed
	KeysScanned uint64 // Total keys examined
	KeysExpired uint64 // Total keys deleted
	mu          sync.RWMutex
}

// RecordCycle updates statistics after an expiration cycle completes.
func (s *ExpirationStats) RecordCycle(scanned, expired int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CyclesRun++
	s.KeysScanned += uint64(scanned)
	s.KeysExpired += uint64(expired)
}

// Snapshot returns a consistent view of current statistics.
func (s *ExpirationStats) Snapshot() (cycles, scanned, expired uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CyclesRun, s.KeysScanned, s.KeysExpired
}

// activeExpireCycle performs one cycle of active expiration using random
// sampling and adaptive stopping. It samples up to config.SampleSize keys
// per iteration, continuing if the expired ratio exceeds the threshold.
func activeExpireCycle(config ExpirationConfig, stats *ExpirationStats, aof *Aof) {
	totalScanned := 0
	totalExpired := 0

	for iter := 0; iter < config.MaxIterations; iter++ {
		scanned := 0
		expired := 0

		// Sample random keys from SETs. Go map iteration is already random,
		// providing natural randomization without explicit rand calls.
		SETsMu.Lock()
		for key, entry := range SETs {
			if scanned >= config.SampleSize {
				break
			}
			scanned++

			if isExpired(entry) {
				delete(SETs, key)
				expired++

				// Log deletion to AOF as DEL command for durability
				if aof != nil {
					delCmd := Value{
						typ: "array",
						array: []Value{
							{typ: "bulk", bulk: "DEL"},
							{typ: "bulk", bulk: key},
						},
					}
					aof.Write(delCmd)
				}
			}
		}
		SETsMu.Unlock()

		totalScanned += scanned
		totalExpired += expired

		// Adaptive stopping: quit if few keys expired (diminishing returns).
		// If expired/scanned < threshold (default 10%), we're wasting CPU
		// checking mostly non-expired keys, so stop and try again next cycle.
		if scanned == 0 || float64(expired)/float64(scanned) < config.AdaptiveThreshold {
			break
		}
	}

	// Record statistics for monitoring
	if stats != nil {
		stats.RecordCycle(totalScanned, totalExpired)
	}
}

// StartActiveExpiration spawns a background goroutine that periodically runs
// active expiration cycles. It returns a stats pointer for monitoring and a
// stopFunc for graceful shutdown.
func StartActiveExpiration(config ExpirationConfig, aof *Aof) (stats *ExpirationStats, stopFunc func()) {
	if !config.Enabled {
		return nil, func() {} // No-op if disabled
	}

	stats = &ExpirationStats{}
	ticker := time.NewTicker(config.Interval)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				activeExpireCycle(config, stats, aof)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	stopFunc = func() {
		close(done)
	}

	return stats, stopFunc
}
