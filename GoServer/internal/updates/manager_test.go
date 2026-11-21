package updates

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf/v2/metadata"
)

func TestSignedUpdateWithMirrorFallback(t *testing.T) {
	now := time.Now()
	root := metadata.Root(now.Add(365 * 24 * time.Hour))
	root.Signed.ConsistentSnapshot = false
	_, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	key, err := metadata.KeyFromPublicKey(private.Public())
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"root", "timestamp", "snapshot", "targets"} {
		if err := root.Signed.AddKey(key, role); err != nil {
			t.Fatal(err)
		}
	}
	signer, err := signature.LoadED25519Signer(private)
	if err != nil {
		t.Fatal(err)
	}
	root.Sign(signer)
	rootBytes, err := root.ToBytes(false)
	if err != nil {
		t.Fatal(err)
	}

	catalog, err := json.Marshal(Catalog{Schema: 1, Items: []Item{
		{ID: "announcement", Kind: "resource", Version: "1.0.0", Target: "announcement.json", APIVersion: 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	resource := []byte(`{"announcement":"verified","config":{"themeColor":"#123456"}}`)
	targets := metadata.Targets(now.Add(24 * time.Hour))
	for file, data := range map[string][]byte{"catalog.json": catalog, "announcement.json": resource} {
		target, err := metadata.TargetFile().FromBytes(file, data, "sha256")
		if err != nil {
			t.Fatal(err)
		}
		targets.Signed.Targets[file] = target
	}
	if _, err := targets.Sign(signer); err != nil {
		t.Fatal(err)
	}
	targetBytes, err := targets.ToBytes(false)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := metadata.Snapshot(now.Add(24 * time.Hour))
	snapshot.Signed.Meta["targets.json"] = metadata.MetaFile(targets.Signed.Version)
	if _, err := snapshot.Sign(signer); err != nil {
		t.Fatal(err)
	}
	snapshotBytes, err := snapshot.ToBytes(false)
	if err != nil {
		t.Fatal(err)
	}
	timestamp := metadata.Timestamp(now.Add(24 * time.Hour))
	timestamp.Signed.Meta["snapshot.json"] = metadata.MetaFile(snapshot.Signed.Version)
	if _, err := timestamp.Sign(signer); err != nil {
		t.Fatal(err)
	}
	timestampBytes, err := timestamp.ToBytes(false)
	if err != nil {
		t.Fatal(err)
	}

	makeServer := func(tampered bool) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/metadata/timestamp.json":
				w.Write(timestampBytes)
			case "/metadata/snapshot.json":
				w.Write(snapshotBytes)
			case "/metadata/targets.json":
				w.Write(targetBytes)
			case "/targets/catalog.json":
				w.Write(catalog)
			case "/targets/announcement.json":
				if tampered {
					w.Write(bytes.Repeat([]byte{'x'}, len(resource)))
				} else {
					w.Write(resource)
				}
			default:
				http.NotFound(w, r)
			}
		}))
	}
	bad, good := makeServer(true), makeServer(false)
	defer bad.Close()
	defer good.Close()
	m := &Manager{
		dir: t.TempDir(), version: "6.0.0-beta1", source: SourceConfig{Root: rootBytes},
		sources: []source{{"tampered", bad.URL + "/metadata/", bad.URL + "/targets/"}, {"good", good.URL + "/metadata/", good.URL + "/targets/"}},
		client:  good.Client(), state: state{Resources: map[string]activeVersion{}},
	}
	if err := m.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := m.Status(); len(got.Available) != 1 || got.Source != "tampered" {
		t.Fatalf("unexpected catalog: %+v", got)
	}
	if err := m.Apply(context.Background(), "announcement"); err != nil {
		t.Fatal(err)
	}
	if got := m.Resources()["announcement"].Announcement; got != "verified" {
		t.Fatalf("expected verified bytes, got %q", got)
	}
}
