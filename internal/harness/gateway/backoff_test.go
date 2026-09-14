package gateway

import (
	"errors"
	"testing"
	"time"
)

func TestBackoffClasses(t *testing.T) {
	p := RetryPolicy{Attempts: 3, Backoff: 100 * time.Millisecond}
	rate := backoffFor(p, 1, errors.New("429 overloaded"))
	server := backoffFor(p, 1, errors.New("provider http 503"))
	other := backoffFor(p, 1, errors.New("boom"))
	if !(rate > server && server > other) {
		t.Fatalf("class ordering broken: %v %v %v", rate, server, other)
	}
	if backoffFor(p, 2, errors.New("x")) <= backoffFor(p, 1, errors.New("x")) {
		t.Fatal("backoff must grow with attempt")
	}
	j := RetryPolicy{Attempts: 3, Backoff: 100 * time.Millisecond, Jitter: true}
	a, b := backoffFor(j, 1, errors.New("x")), backoffFor(j, 1, errors.New("x"))
	if a != b {
		t.Fatal("jitter must be deterministic")
	}
	if a < 100*time.Millisecond || a >= 200*time.Millisecond {
		t.Fatalf("jitter out of band: %v", a)
	}
}
