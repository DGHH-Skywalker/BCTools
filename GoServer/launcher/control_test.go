package launcher

import (
	"errors"
	"testing"
	"time"
)

func TestBrowserToDesktopHandoff(t *testing.T) {
	dir := t.TempDir()
	commands := make(chan string, 1)
	first, err := Claim(dir, "browser", commands)
	if err != nil || first == nil {
		t.Fatalf("browser claim: %v", err)
	}
	defer first.Close()
	second, err := Claim(dir, "desktop", commands)
	if err != nil || second != nil {
		t.Fatalf("two processes can claim one data root: %v", err)
	}
	if err := SendControl(dir, "desktop"); !errors.Is(err, ErrSwitching) {
		t.Fatalf("handoff: %v", err)
	}
	select {
	case command := <-commands:
		if command != "shutdown" {
			t.Fatalf("unexpected %s", command)
		}
	case <-time.After(time.Second):
		t.Fatal("missing shutdown")
	}
	_ = first.Close()
	second, err = Claim(dir, "desktop", commands)
	if err != nil || second == nil {
		t.Fatalf("desktop claim: %v", err)
	}
	defer second.Close()
	if err := SendControl(dir, "desktop"); err != nil {
		t.Fatal(err)
	}
	select {
	case command := <-commands:
		if command != "activate" {
			t.Fatalf("unexpected %s", command)
		}
	case <-time.After(time.Second):
		t.Fatal("missing activation")
	}
}
