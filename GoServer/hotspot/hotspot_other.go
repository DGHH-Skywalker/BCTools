//go:build !windows

package hotspot

// Status represents the mobile hotspot state.
type Status string

const (
	StatusIdle     Status = "idle"
	StatusStarting Status = "starting"
	StatusStarted  Status = "started"
	StatusFailed   Status = "failed"
)

// Manager is a no-op placeholder on non-Windows platforms.
type Manager struct{}

// NewManager creates a no-op manager on non-Windows platforms.
func NewManager(appDataDir string) *Manager { return &Manager{} }

// Start is a no-op on non-Windows platforms.
func (m *Manager) Start() error { return nil }

// Status returns idle on non-Windows platforms.
func (m *Manager) Status() Status { return StatusIdle }

// SSID returns empty on non-Windows platforms.
func (m *Manager) SSID() string { return "" }

// Password returns empty on non-Windows platforms.
func (m *Manager) Password() string { return "" }

// Message returns empty on non-Windows platforms.
func (m *Manager) Message() string { return "热点功能暂不支持当前系统" }

// Snapshot returns idle on non-Windows platforms.
func (m *Manager) Snapshot() (Status, string, string, string) {
	return StatusIdle, "", "", m.Message()
}
