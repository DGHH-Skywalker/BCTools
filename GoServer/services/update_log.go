package services

import (
	"crypto/ed25519"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	updateLogSchemaVersion = 1
	updateLogJSONName      = "update-log.json"
	updateLogSignatureName = "update-log.sig"
	maxUpdateLogJSONSize   = 128 * 1024
	maxUpdateLogSigSize    = 4 * 1024
	updateLogPublicKeyB64  = "JJ0cKtM5/S31DGK9YC77DptfxCvPLpwZpQh6DfvpqfY="
)

//go:embed update_log_default.json
var updateLogAssets embed.FS

var updateLogVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:\.[0-9]+)?$`)

// UpdateLog is the only update-log shape exposed to the rest of the backend.
// Values from disk reach this type only after signature and schema validation.
type UpdateLog struct {
	SchemaVersion int      `json:"schemaVersion"`
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	PublishedAt   string   `json:"publishedAt"`
	ExpiresAt     string   `json:"expiresAt"`
	Title         string   `json:"title"`
	Content       []string `json:"content"`
	DownloadURL   string   `json:"downloadUrl,omitempty"`
}

type UpdateLogResponse struct {
	UpdateLog
	Recent    bool   `json:"recent"`
	StartupID string `json:"startupId"`
}

type UpdateLogService struct {
	latest    UpdateLog
	now       func() time.Time
	startupID string
}

func NewUpdateLogService(dataDir string) (*UpdateLogService, error) {
	defaultJSON, err := updateLogAssets.ReadFile("update_log_default.json")
	if err != nil {
		return nil, fmt.Errorf("read embedded update log: %w", err)
	}
	return newUpdateLogService(dataDir, defaultJSON, mustUpdateLogPublicKey(), time.Now)
}

func newUpdateLogService(dataDir string, defaultJSON []byte, publicKey ed25519.PublicKey, now func() time.Time) (*UpdateLogService, error) {
	fallback, err := validateUpdateLog(defaultJSON, now(), false)
	if err != nil {
		return nil, fmt.Errorf("invalid embedded update log: %w", err)
	}

	service := &UpdateLogService{latest: fallback, now: now, startupID: newUpdateLogStartupID(now())}
	jsonBytes, jsonErr := readUpdateLogFile(filepath.Join(dataDir, updateLogJSONName), maxUpdateLogJSONSize)
	signatureBytes, signatureErr := readUpdateLogFile(filepath.Join(dataDir, updateLogSignatureName), maxUpdateLogSigSize)
	if jsonErr != nil || signatureErr != nil {
		return service, nil
	}

	external, err := verifyAndValidateUpdateLog(jsonBytes, signatureBytes, publicKey, now())
	if err == nil {
		service.latest = external
	}
	return service, nil
}

func readUpdateLogFile(path string, maxSize int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxSize {
		return nil, errors.New("invalid update-log file")
	}
	return os.ReadFile(path)
}

func (s *UpdateLogService) Latest() UpdateLogResponse {
	return UpdateLogResponse{
		UpdateLog: s.latest,
		Recent:    PublishedWithin(s.latest.PublishedAt, s.now(), 14*24*time.Hour),
		StartupID: s.startupID,
	}
}

func newUpdateLogStartupID(now time.Time) string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err == nil {
		return hex.EncodeToString(value)
	}
	return fmt.Sprintf("%d", now.UnixNano())
}

func PublishedWithin(value string, now time.Time, window time.Duration) bool {
	publishedAt, err := time.Parse(time.RFC3339, value)
	if err != nil || publishedAt.After(now) {
		return false
	}
	return now.Sub(publishedAt) <= window
}

func verifyAndValidateUpdateLog(rawJSON, rawSignature []byte, publicKey ed25519.PublicKey, now time.Time) (UpdateLog, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return UpdateLog{}, errors.New("invalid public key")
	}
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(rawSignature)))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return UpdateLog{}, errors.New("invalid signature encoding")
	}
	if !ed25519.Verify(publicKey, rawJSON, signature) {
		return UpdateLog{}, errors.New("signature verification failed")
	}
	return validateUpdateLog(rawJSON, now, true)
}

func validateUpdateLog(rawJSON []byte, now time.Time, requireActive bool) (UpdateLog, error) {
	var log UpdateLog
	decoder := json.NewDecoder(strings.NewReader(string(rawJSON)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&log); err != nil {
		return UpdateLog{}, fmt.Errorf("decode update log: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return UpdateLog{}, errors.New("unexpected trailing JSON")
	}
	if log.SchemaVersion != updateLogSchemaVersion {
		return UpdateLog{}, errors.New("unsupported schema version")
	}
	if strings.TrimSpace(log.ID) == "" || len(log.ID) > 128 {
		return UpdateLog{}, errors.New("invalid id")
	}
	if !updateLogVersionPattern.MatchString(log.Version) {
		return UpdateLog{}, errors.New("invalid version")
	}
	publishedAt, err := time.Parse(time.RFC3339, log.PublishedAt)
	if err != nil || publishedAt.After(now.Add(5*time.Minute)) {
		return UpdateLog{}, errors.New("invalid publishedAt")
	}
	expiresAt, err := time.Parse(time.RFC3339, log.ExpiresAt)
	if err != nil || !expiresAt.After(publishedAt) {
		return UpdateLog{}, errors.New("invalid expiresAt")
	}
	if requireActive && !expiresAt.After(now) {
		return UpdateLog{}, errors.New("expired update log")
	}
	if title := strings.TrimSpace(log.Title); title == "" || len([]rune(title)) > 120 {
		return UpdateLog{}, errors.New("invalid title")
	}
	if len(log.Content) == 0 || len(log.Content) > 50 {
		return UpdateLog{}, errors.New("invalid content")
	}
	for _, line := range log.Content {
		if text := strings.TrimSpace(line); text == "" || len([]rune(text)) > 1000 {
			return UpdateLog{}, errors.New("invalid content item")
		}
	}
	if log.DownloadURL != "" {
		downloadURL, err := url.Parse(log.DownloadURL)
		if err != nil || downloadURL.Scheme != "https" || downloadURL.Host == "" {
			return UpdateLog{}, errors.New("invalid downloadUrl")
		}
	}
	return log, nil
}

func mustUpdateLogPublicKey() ed25519.PublicKey {
	decoded, err := base64.StdEncoding.DecodeString(updateLogPublicKeyB64)
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		panic("invalid embedded update-log public key")
	}
	return ed25519.PublicKey(decoded)
}

// UpdateLogPublicKey returns a copy for release tooling. It contains no secret.
func UpdateLogPublicKey() ed25519.PublicKey {
	return append(ed25519.PublicKey(nil), mustUpdateLogPublicKey()...)
}
