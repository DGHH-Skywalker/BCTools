package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"broadcast-tool/services"
)

func main() {
	if len(os.Args) != 3 {
		fatal("usage: sign-update-log <update-log.json> <update-log.sig>")
	}
	encodedKey := strings.TrimSpace(os.Getenv("BCTOOLS_UPDATE_LOG_PRIVATE_KEY"))
	keyBytes, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		fatal("BCTOOLS_UPDATE_LOG_PRIVATE_KEY must be base64")
	}
	var privateKey ed25519.PrivateKey
	switch len(keyBytes) {
	case ed25519.SeedSize:
		privateKey = ed25519.NewKeyFromSeed(keyBytes)
	case ed25519.PrivateKeySize:
		privateKey = ed25519.PrivateKey(keyBytes)
	default:
		fatal("private key must contain a 32-byte seed or 64-byte Ed25519 key")
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	if !publicKey.Equal(services.UpdateLogPublicKey()) {
		fatal("private key does not match the public key embedded in BCTools")
	}
	rawJSON, err := os.ReadFile(os.Args[1])
	if err != nil {
		fatal(err.Error())
	}
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, rawJSON))
	if err := os.WriteFile(os.Args[2], []byte(signature+"\n"), 0600); err != nil {
		fatal(err.Error())
	}
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
