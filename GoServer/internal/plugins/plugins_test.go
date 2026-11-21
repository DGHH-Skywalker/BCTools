package plugins

import (
	"context"
	"testing"
)

func TestExampleInstallationActivationAndRemoval(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	registry, err := New(dir, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ExamplePackage()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Install(ctx, b); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Call(ctx, "example-layout", "ping", nil); err == nil {
		t.Fatal("disabled plugin ran")
	}
	if err := registry.Configure("example-layout", true, true); err != nil {
		t.Fatal(err)
	}
	if out, err := registry.Call(ctx, "example-layout", "ping", nil); err != nil || len(out) != 0 {
		t.Fatalf("command failed: %q %v", out, err)
	}
	registry, err = New(dir, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if state := registry.Snapshot(); state.LayoutProvider != "example-layout" || state.Layout == nil {
		t.Fatalf("layout missing after restart: %+v", state)
	}
	if err := registry.Remove("example-layout"); err != nil {
		t.Fatal(err)
	}
	if state := registry.Snapshot(); state.Layout != nil || len(state.Plugins) != 0 {
		t.Fatalf("plugin state not revoked: %+v", state)
	}
}
