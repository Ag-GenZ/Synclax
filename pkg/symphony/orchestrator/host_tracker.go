package orchestrator

import (
	"errors"
	"math"
	"sync"
)

var ErrNoWorkerHostSlots = errors.New("no_worker_host_slots")

type hostTracker struct {
	mu      sync.Mutex
	hosts   []string
	running map[string]int // host -> running count
	perHost int
}

func newHostTracker(hosts []string, perHost int) *hostTracker {
	cp := append([]string(nil), hosts...)
	ht := &hostTracker{
		hosts:   cp,
		running: map[string]int{},
		perHost: perHost,
	}
	for _, h := range cp {
		ht.running[h] = 0
	}
	return ht
}

func (ht *hostTracker) acquire() (*string, error) {
	if ht == nil || len(ht.hosts) == 0 {
		return nil, nil
	}

	ht.mu.Lock()
	defer ht.mu.Unlock()

	bestHost := ""
	bestCount := math.MaxInt
	for _, h := range ht.hosts {
		c := ht.running[h]
		if c < bestCount {
			bestHost = h
			bestCount = c
		}
	}

	if bestHost == "" {
		return nil, ErrNoWorkerHostSlots
	}
	if ht.perHost > 0 && bestCount >= ht.perHost {
		return nil, ErrNoWorkerHostSlots
	}

	ht.running[bestHost] = bestCount + 1
	out := bestHost
	return &out, nil
}

func (ht *hostTracker) release(host *string) {
	if ht == nil || host == nil || *host == "" {
		return
	}

	ht.mu.Lock()
	defer ht.mu.Unlock()

	cur := ht.running[*host]
	if cur <= 0 {
		ht.running[*host] = 0
		return
	}
	ht.running[*host] = cur - 1
}

