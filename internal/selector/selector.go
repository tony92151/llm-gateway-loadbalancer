package selector

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Key struct {
	Label          string
	Secret         string
	Weight         int
	RPMLimit       int
	TPMLimit       int
	Enabled        bool
	InFlight       int
	MinuteRequests int
	MinuteTokens   int
	CooldownUntil  time.Time
	LastError      string
	windowStarted  time.Time
}

type Pool struct {
	mu       sync.Mutex
	strategy string
	keys     []Key
	next     int
}

func NewPool(strategy string, keys []Key) (*Pool, error) {
	if strategy == "" {
		strategy = "leastload"
	}
	switch strategy {
	case "leastload", "roundrobin", "weighted":
	default:
		return nil, fmt.Errorf("unsupported selector strategy %q", strategy)
	}
	copied := make([]Key, len(keys))
	copy(copied, keys)
	for i := range copied {
		if copied[i].Weight <= 0 {
			copied[i].Weight = 1
		}
	}
	return &Pool{strategy: strategy, keys: copied}, nil
}

func (p *Pool) Select() (Key, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	p.resetExpiredWindows(now)
	switch p.strategy {
	case "roundrobin":
		return p.selectRoundRobin(now)
	case "weighted":
		return p.selectWeighted(now)
	default:
		return p.selectLeastLoad(now)
	}
}

func (p *Pool) MarkDone(label string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.keys {
		if p.keys[i].Label == label && p.keys[i].InFlight > 0 {
			p.keys[i].InFlight--
			return
		}
	}
}

func (p *Pool) MarkFailure(label string, cooldown time.Duration, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.keys {
		if p.keys[i].Label == label {
			p.keys[i].LastError = message
			if cooldown > 0 {
				p.keys[i].CooldownUntil = time.Now().Add(cooldown)
			}
			return
		}
	}
}

func (p *Pool) MarkHealthy(label string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.keys {
		if p.keys[i].Label == label {
			p.keys[i].LastError = ""
			p.keys[i].CooldownUntil = time.Time{}
			return
		}
	}
}

func (p *Pool) RecordUsage(label string, tokens int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	for i := range p.keys {
		if p.keys[i].Label == label {
			p.resetExpiredWindow(i, now)
			p.keys[i].MinuteTokens += max(tokens, 0)
			return
		}
	}
}

func (p *Pool) Snapshot() []Key {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Key, len(p.keys))
	copy(out, p.keys)
	return out
}

func (p *Pool) selectRoundRobin(now time.Time) (Key, error) {
	if len(p.keys) == 0 {
		return Key{}, errors.New("no upstream keys configured")
	}
	for attempts := 0; attempts < len(p.keys); attempts++ {
		idx := p.next % len(p.keys)
		p.next++
		if p.available(idx, now) {
			p.keys[idx].InFlight++
			p.keys[idx].MinuteRequests++
			return p.keys[idx], nil
		}
	}
	return Key{}, errors.New("no available upstream keys")
}

func (p *Pool) selectLeastLoad(now time.Time) (Key, error) {
	best := -1
	for i := range p.keys {
		if !p.available(i, now) {
			continue
		}
		if best == -1 ||
			p.keys[i].InFlight < p.keys[best].InFlight ||
			(p.keys[i].InFlight == p.keys[best].InFlight && p.keys[i].Weight > p.keys[best].Weight) {
			best = i
		}
	}
	if best == -1 {
		return Key{}, errors.New("no available upstream keys")
	}
	p.keys[best].InFlight++
	p.keys[best].MinuteRequests++
	return p.keys[best], nil
}

func (p *Pool) selectWeighted(now time.Time) (Key, error) {
	total := 0
	for i := range p.keys {
		if p.available(i, now) {
			total += p.keys[i].Weight
		}
	}
	if total == 0 {
		return Key{}, errors.New("no available upstream keys")
	}
	pick := rand.Intn(total)
	for i := range p.keys {
		if !p.available(i, now) {
			continue
		}
		if pick < p.keys[i].Weight {
			p.keys[i].InFlight++
			p.keys[i].MinuteRequests++
			return p.keys[i], nil
		}
		pick -= p.keys[i].Weight
	}
	return Key{}, errors.New("no available upstream keys")
}

func (p *Pool) available(i int, now time.Time) bool {
	key := p.keys[i]
	if !key.Enabled || (!key.CooldownUntil.IsZero() && now.Before(key.CooldownUntil)) {
		return false
	}
	if key.RPMLimit > 0 && key.MinuteRequests >= key.RPMLimit {
		return false
	}
	if key.TPMLimit > 0 && key.MinuteTokens >= key.TPMLimit {
		return false
	}
	return true
}

func (p *Pool) resetExpiredWindows(now time.Time) {
	for i := range p.keys {
		p.resetExpiredWindow(i, now)
	}
}

func (p *Pool) resetExpiredWindow(i int, now time.Time) {
	if p.keys[i].windowStarted.IsZero() {
		p.keys[i].windowStarted = now.Truncate(time.Minute)
		return
	}
	if now.Sub(p.keys[i].windowStarted) >= time.Minute {
		p.keys[i].windowStarted = now.Truncate(time.Minute)
		p.keys[i].MinuteRequests = 0
		p.keys[i].MinuteTokens = 0
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
