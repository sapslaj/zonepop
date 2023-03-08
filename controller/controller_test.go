package controller

import (
	"testing"
	"time"
)

func TestShouldRunOnce(t *testing.T) {
	ctrl := &Controller{
		Interval: 10 * time.Minute,
	}

	now := time.Now()

	if !ctrl.ShouldRunOnce(now) {
		t.Errorf("controller.ShouldRunOnce(now) should be true on first run")
	}
	if ctrl.ShouldRunOnce(now) {
		t.Errorf("controller.ShouldRunOnce(now) should be false on second run")
	}

	now = now.Add(10 * time.Second)
	if ctrl.ShouldRunOnce(now) {
		t.Fatalf("controller.ShouldRunOnce(now) should be false after only a short time after first schedule")
	}

	now = now.Add(10 * time.Minute)
	if !ctrl.ShouldRunOnce(now) {
		t.Fatalf("controller.ShouldRunOnce(now) should be true after the full interval is elapsed")
	}
}
