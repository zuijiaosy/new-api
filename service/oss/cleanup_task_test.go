package oss

import (
	"testing"
	"time"
)

// 只测互斥：第二次并发调用立即返回 false
func TestCleanupMutex(t *testing.T) {
	var guard = &cleanupGuard{}
	if !guard.tryAcquire() {
		t.Fatalf("first acquire should succeed")
	}
	if guard.tryAcquire() {
		t.Fatalf("second acquire must fail")
	}
	guard.release()
	if !guard.tryAcquire() {
		t.Fatalf("after release should succeed again")
	}
	guard.release()
}

// 确保 ticker 周期由 cfg 决定
func TestComputeCleanupInterval(t *testing.T) {
	cases := []struct {
		in   int
		want time.Duration
	}{
		{0, 24 * time.Hour},
		{-1, 24 * time.Hour},
		{1, time.Hour},
		{6, 6 * time.Hour},
	}
	for _, c := range cases {
		got := computeCleanupInterval(c.in)
		if got != c.want {
			t.Fatalf("in=%d want=%v got=%v", c.in, c.want, got)
		}
	}
}
