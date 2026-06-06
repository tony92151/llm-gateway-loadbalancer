package selector

import (
	"testing"
	"time"
)

func TestLeastLoadSkipsCooldownAndChoosesLowestInFlight(t *testing.T) {
	pool, err := NewPool("leastload", []Key{
		{Label: "busy", Weight: 10, Enabled: true, InFlight: 4},
		{Label: "cooldown", Weight: 10, Enabled: true, CooldownUntil: time.Now().Add(time.Minute)},
		{Label: "ready", Weight: 10, Enabled: true, InFlight: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	key, err := pool.Select()
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if key.Label != "ready" {
		t.Fatalf("selected %q, want ready", key.Label)
	}
}

func TestRoundRobinCyclesAvailableKeys(t *testing.T) {
	pool, err := NewPool("roundrobin", []Key{
		{Label: "a", Weight: 1, Enabled: true},
		{Label: "b", Weight: 1, Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	first, _ := pool.Select()
	second, _ := pool.Select()
	third, _ := pool.Select()

	if first.Label != "a" || second.Label != "b" || third.Label != "a" {
		t.Fatalf("sequence = %s,%s,%s", first.Label, second.Label, third.Label)
	}
}

func TestSelectSkipsKeyAtRPMLimit(t *testing.T) {
	pool, err := NewPool("roundrobin", []Key{
		{Label: "a", Weight: 1, Enabled: true, RPMLimit: 1},
		{Label: "b", Weight: 1, Enabled: true, RPMLimit: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	first, err := pool.Select()
	if err != nil {
		t.Fatal(err)
	}
	pool.MarkDone(first.Label)
	second, err := pool.Select()
	if err != nil {
		t.Fatal(err)
	}

	if first.Label != "a" || second.Label != "b" {
		t.Fatalf("sequence = %s,%s", first.Label, second.Label)
	}
}

func TestSelectSkipsKeyAtTPMLimitAfterUsageRecorded(t *testing.T) {
	pool, err := NewPool("roundrobin", []Key{
		{Label: "a", Weight: 1, Enabled: true, TPMLimit: 10},
		{Label: "b", Weight: 1, Enabled: true, TPMLimit: 10},
	})
	if err != nil {
		t.Fatal(err)
	}

	first, err := pool.Select()
	if err != nil {
		t.Fatal(err)
	}
	pool.RecordUsage(first.Label, 11)
	pool.MarkDone(first.Label)
	second, err := pool.Select()
	if err != nil {
		t.Fatal(err)
	}

	if first.Label != "a" || second.Label != "b" {
		t.Fatalf("sequence = %s,%s", first.Label, second.Label)
	}
}
