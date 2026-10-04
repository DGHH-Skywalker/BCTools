package launcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNumericVersions(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{{"5.10.0.0", "5.9.0.0", 1}, {"5.9", "5.9.0.0", 0}, {"5.8.9.9", "5.9.0.0", -1}} {
		if CompareVersions(tc.a, tc.b) != tc.want {
			t.Fatal(tc)
		}
	}
}

func TestInvalidVersionCannotEscapeManagedDirectory(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(t.TempDir(), "bctools.exe")
	os.WriteFile(exe, []byte("program"), 0700)
	if _, _, err := RegisterVersion(dir, exe, "../outside"); err == nil {
		t.Fatal("invalid version accepted")
	}
}

func TestRegistrationRoutingAndTamper(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(t.TempDir(), "bctools.exe")
	if err := os.WriteFile(exe, []byte("release A"), 0700); err != nil {
		t.Fatal(err)
	}
	one, _, err := RegisterVersion(dir, exe, "5.9.0.0")
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(exe, []byte("release B"), 0700)
	two, _, err := RegisterVersion(dir, exe, "5.10.0.0")
	if err != nil {
		t.Fatal(err)
	}
	best, _, err := RegisterVersion(dir, one.Path, "5.9.0.0")
	if err != nil || best.Path != two.Path {
		t.Fatalf("newest: %v %v", best, err)
	}
	os.WriteFile(two.Path, []byte("tampered"), 0700)
	best, _, err = RegisterVersion(dir, one.Path, "5.9.0.0")
	if err != nil || best.Path != one.Path {
		t.Fatalf("tampered version selected: %v %v", best, err)
	}
}
