package services

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var updateLogTestNow = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

func testUpdateLogJSON(publishedAt, expiresAt string) []byte {
	return []byte(`{"schemaVersion":1,"id":"test-update","version":"5.8.0.0","publishedAt":"` + publishedAt + `","expiresAt":"` + expiresAt + `","title":"Test update","content":["Item one"]}`)
}

func newTestUpdateLogService(t *testing.T, externalJSON, signature []byte, publicKey ed25519.PublicKey) *UpdateLogService {
	t.Helper()
	dir := t.TempDir()
	if externalJSON != nil {
		if err := os.WriteFile(filepath.Join(dir, updateLogJSONName), externalJSON, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if signature != nil {
		if err := os.WriteFile(filepath.Join(dir, updateLogSignatureName), signature, 0600); err != nil {
			t.Fatal(err)
		}
	}
	fallback := testUpdateLogJSON("2026-09-01T00:00:00Z", "2099-12-31T00:00:00Z")
	service, err := newUpdateLogService(dir, fallback, publicKey, func() time.Time { return updateLogTestNow })
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestUpdateLogValidSignature(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raw := testUpdateLogJSON("2026-09-10T00:00:00Z", "2026-10-01T00:00:00Z")
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, raw)))
	service := newTestUpdateLogService(t, raw, sig, publicKey)
	if got := service.Latest().ID; got != "test-update" {
		t.Fatalf("got %q", got)
	}
}

func TestUpdateLogInvalidSignatureFallsBack(t *testing.T) {
	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	raw := testUpdateLogJSON("2026-09-10T00:00:00Z", "2026-10-01T00:00:00Z")
	service := newTestUpdateLogService(t, raw, []byte(base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))), publicKey)
	if got := service.Latest().PublishedAt; got != "2026-09-01T00:00:00Z" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestUpdateLogMissingExternalFileFallsBack(t *testing.T) {
	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	service := newTestUpdateLogService(t, nil, nil, publicKey)
	if service.Latest().PublishedAt != "2026-09-01T00:00:00Z" {
		t.Fatal("embedded fallback was not selected")
	}
}

func TestUpdateLogCorruptedJSONFallsBack(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	raw := []byte(`{"broken"`)
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, raw)))
	service := newTestUpdateLogService(t, raw, sig, publicKey)
	if service.Latest().PublishedAt != "2026-09-01T00:00:00Z" {
		t.Fatal("embedded fallback was not selected")
	}
}

func TestUpdateLogExpiredFallsBack(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	raw := testUpdateLogJSON("2026-08-01T00:00:00Z", "2026-09-01T00:00:00Z")
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, raw)))
	service := newTestUpdateLogService(t, raw, sig, publicKey)
	if service.Latest().PublishedAt != "2026-09-01T00:00:00Z" {
		t.Fatal("expired data did not fall back")
	}
}

func TestUpdateLogOlderThanEmbeddedFallsBack(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	raw := []byte(`{"schemaVersion":1,"id":"old-update","version":"5.7.0.0","publishedAt":"2026-09-10T00:00:00Z","expiresAt":"2026-10-01T00:00:00Z","title":"Old update","content":["Item one"]}`)
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, raw)))
	service := newTestUpdateLogService(t, raw, sig, publicKey)
	if service.Latest().Version != "5.8.0.0" {
		t.Fatal("older signed data replaced the embedded update log")
	}
}

func TestUpdateLogLifetimeTooLongFallsBack(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	raw := testUpdateLogJSON("2026-09-10T00:00:00Z", "2027-09-10T00:00:00Z")
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, raw)))
	service := newTestUpdateLogService(t, raw, sig, publicKey)
	if service.Latest().PublishedAt != "2026-09-01T00:00:00Z" {
		t.Fatal("overlong external update log did not fall back")
	}
}

func TestUpdateLogClaimedOncePerStartup(t *testing.T) {
	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	service := newTestUpdateLogService(t, nil, nil, publicKey)
	service.now = func() time.Time { return time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC) }
	if !service.ClaimLatest().ShouldShow {
		t.Fatal("first client should claim a recent update")
	}
	if service.ClaimLatest().ShouldShow {
		t.Fatal("second client must not claim the same startup update")
	}
}

func TestPublishedWithinFourteenDays(t *testing.T) {
	if !PublishedWithin("2026-09-01T12:00:00Z", updateLogTestNow, 14*24*time.Hour) {
		t.Fatal("14-day boundary should be visible")
	}
	if PublishedWithin("2026-09-01T11:59:59Z", updateLogTestNow, 14*24*time.Hour) {
		t.Fatal("older update should not be visible")
	}
	if PublishedWithin("2026-09-16T00:00:00Z", updateLogTestNow, 14*24*time.Hour) {
		t.Fatal("future update should not be visible")
	}
}
