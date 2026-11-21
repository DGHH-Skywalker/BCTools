package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBoundedDesktopAudioRanges(t *testing.T) {
	const length = 13 << 20
	tests := []struct{ header, want string }{
		{"bytes=0-", "bytes=0-4194303"},
		{"bytes=30-39", "bytes=30-39"},
		{"bytes=-25", fmt.Sprintf("bytes=%d-%d", length-25, length-1)},
		{"bytes=4194304-", "bytes=4194304-8388607"},
	}
	for _, tc := range tests {
		r := httptest.NewRequest(http.MethodGet, "http://wails.localhost/api/files/stream", nil)
		r.Header.Set("Range", tc.header)
		w := httptest.NewRecorder()
		result := boundedAudioRequest(w, r, length)
		if result == nil || result.Header.Get("Range") != tc.want {
			t.Fatalf("%s did not match %s", tc.header, tc.want)
		}
		if got := r.Header.Get("Range"); got != tc.header {
			t.Fatalf("original request was changed: %q", got)
		}
	}
	bad := httptest.NewRequest(http.MethodGet, "http://wails.localhost/api/files/stream", nil)
	bad.Header.Set("Range", "bytes=0-7,30-39")
	w := httptest.NewRecorder()
	if result := boundedAudioRequest(w, bad, length); result != nil || w.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("expected 416, got %d", w.Code)
	}
}
