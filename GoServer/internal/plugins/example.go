package plugins

import (
	"archive/zip"
	"bytes"
	_ "embed"
)

//go:embed example-layout.json
var exampleManifest []byte

// A minimal WASM module exporting run() -> i32 with exit status 0.
var exampleWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
	0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f,
	0x03, 0x02, 0x01, 0x00,
	0x07, 0x07, 0x01, 0x03, 0x72, 0x75, 0x6e, 0x00, 0x00,
	0x0a, 0x06, 0x01, 0x04, 0x00, 0x41, 0x00, 0x0b,
}

func ExamplePackage() ([]byte, error) {
	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	for _, entry := range []struct {
		name string
		body []byte
	}{{"manifest.json", exampleManifest}, {"plugin.wasm", exampleWasm}} {
		dest, err := archive.Create(entry.name)
		if err != nil {
			return nil, err
		}
		if _, err := dest.Write(entry.body); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
