//go:build windows

package launcher

import (
	"io"
	"path/filepath"
	"testing"
	"time"
)

func TestExistingStartupMutexWaitsAndCanBeReacquired(t *testing.T) {
	key := filepath.Join(t.TempDir(), "startup")
	first, err := AcquireLock(key)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	type result struct {
		lock io.Closer
		err  error
	}
	ready := make(chan result, 1)
	go func() { lock, err := AcquireLock(key); ready <- result{lock, err} }()
	select {
	case r := <-ready:
		if r.lock != nil {
			r.lock.Close()
		}
		t.Fatalf("second launcher did not wait: %v", r.err)
	case <-time.After(200 * time.Millisecond):
	}
	first.Close()
	select {
	case r := <-ready:
		if r.err != nil {
			t.Fatalf("existing mutex could not be acquired: %v", r.err)
		}
		r.lock.Close()
	case <-time.After(5 * time.Second):
		t.Fatal("second launcher did not acquire the released mutex")
	}
}
