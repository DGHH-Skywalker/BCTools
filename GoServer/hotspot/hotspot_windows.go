//go:build windows

package hotspot

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"broadcast-tool/paths"
	"broadcast-tool/runner"
)

// Status represents the mobile hotspot state.
type Status string

const (
	StatusIdle     Status = "idle"
	StatusStarting Status = "starting"
	StatusStarted  Status = "started"
	StatusFailed   Status = "failed"
)

// Manager controls the Windows mobile hotspot.
// It prefers the Windows Runtime (WinRT) API and falls back to netsh wlan hostednetwork.
type Manager struct {
	mu         sync.RWMutex
	status     Status
	ssid       string
	password   string
	message    string
	appDataDir string
}

// NewManager creates a new hotspot manager.
func NewManager(appDataDir string) *Manager {
	return &Manager{status: StatusIdle, appDataDir: appDataDir}
}

// Start requests to start the mobile hotspot.
func (m *Manager) Start() error {
	m.mu.Lock()
	m.status = StatusStarting
	m.ssid = "BroadcastTool"
	m.password = randomPassword(8)
	m.message = "正在检测无线网卡..."
	m.mu.Unlock()

	go m.runStart()
	return nil
}

func (m *Manager) runStart() {
	if err := detectWirelessAdapter(); err != nil {
		m.setFailed(err.Error())
		return
	}

	m.setMessage("正在尝试开启移动热点...")
	result := make(chan bool, 2)
	go func() { result <- m.runWinRTStart() }()
	go func() { result <- m.runNetshStart() }()

	failCount := 0
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		select {
		case ok := <-result:
			if ok {
				return
			}
			failCount++
			if failCount >= 2 {
				m.setFailedIfStarting("热点开启失败，请检查无线网卡是否支持移动热点或已授权管理员权限")
				return
			}
		case <-timer.C:
			m.setFailedIfStarting("热点开启超时，请检查无线网卡是否支持移动热点或已授权管理员权限")
			return
		}
	}
}

func detectWirelessAdapter() error {
	out, err := runner.Command("cmd", "/c", "netsh wlan show interfaces").Output()
	if err != nil {
		return errors.New("检测无线网卡失败，请确认已启用 Wi-Fi 或安装无线网卡驱动")
	}
	output := string(out)
	lower := strings.ToLower(output)
	if strings.Contains(lower, "no wireless interface") ||
		strings.Contains(lower, "there is no wireless interface") ||
		strings.Contains(lower, "系统中没有无线接口") ||
		strings.Contains(lower, "无线自动配置服务") {
		return errors.New("未检测到无线网卡，请确认已启用 Wi-Fi 或安装无线网卡驱动")
	}
	if !strings.Contains(lower, "ssid") && !strings.Contains(lower, "bssid") && !strings.Contains(lower, "接口名称") {
		return errors.New("未检测到无线网卡，请确认已启用 Wi-Fi 或安装无线网卡驱动")
	}
	return nil
}

// runNetshStart uses the legacy netsh wlan hostednetwork command.
func (m *Manager) runNetshStart() bool {
	cmdLine := fmt.Sprintf(`netsh wlan set hostednetwork mode=allow ssid="%s" key="%s" && netsh wlan start hostednetwork`, m.SSID(), m.Password())

	if err := runElevated("cmd.exe", "/c "+cmdLine); err != nil {
		m.setFailedIfStarting(fmt.Sprintf("请求管理员权限失败: %v", err))
		return false
	}

	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		m.refreshNetshStatus()
		st := m.Status()
		if st == StatusStarted {
			return true
		}
		if st == StatusFailed {
			return false
		}
	}
	m.setFailedIfStarting("netsh 热点开启超时，请检查无线网卡是否支持移动热点")
	return false
}

// runWinRTStart uses PowerShell with Windows Runtime projection to start the hotspot.
// It returns true if the hotspot was started successfully.
func (m *Manager) runWinRTStart() bool {
	tempDir := paths.GetTempDir(m.appDataDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		m.setFailedIfStarting(fmt.Sprintf("无法创建临时目录: %v", err))
		return false
	}
	scriptPath := filepath.Join(tempDir, "hotspot-winrt.ps1")
	statusPath := filepath.Join(tempDir, "hotspot-winrt-status.json")
	os.Remove(statusPath)

	if err := os.WriteFile(scriptPath, []byte(winRTScript), 0644); err != nil {
		m.setFailedIfStarting(fmt.Sprintf("无法写入 WinRT 脚本: %v", err))
		return false
	}

	args := fmt.Sprintf(`-ExecutionPolicy Bypass -WindowStyle Hidden -NoProfile -File "%s" -Ssid "%s" -Password "%s" -StatusFile "%s"`,
		scriptPath, m.SSID(), m.Password(), statusPath)
	if err := runElevated("powershell.exe", args); err != nil {
		m.setFailedIfStarting(fmt.Sprintf("请求管理员权限失败: %v", err))
		return false
	}

	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		data, err := os.ReadFile(statusPath)
		if err != nil {
			continue
		}
		var result struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}
		if result.Status == "started" {
			m.setStarted("热点已开启")
			return true
		}
		if result.Status == "failed" {
			m.setFailedIfStarting(result.Message)
			return false
		}
	}
	m.setFailedIfStarting("WinRT 热点开启超时，请检查无线网卡是否支持移动热点或已授权管理员权限")
	return false
}

// Status returns the current hotspot status.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// SSID returns the configured hotspot SSID.
func (m *Manager) SSID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ssid
}

// Password returns the configured hotspot password.
func (m *Manager) Password() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.password
}

// Message returns a human-readable status message.
func (m *Manager) Message() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.message
}

// Snapshot returns a copy of the current state.
func (m *Manager) Snapshot() (status Status, ssid, password, message string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status, m.ssid, m.password, m.message
}

func (m *Manager) setStarted(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = StatusStarted
	m.message = msg
}

func (m *Manager) setFailed(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = StatusFailed
	m.message = msg
}

func (m *Manager) setMessage(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.message = msg
}

func (m *Manager) setFailedIfStarting(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status != StatusStarting {
		return
	}
	m.status = StatusFailed
	m.message = msg
}

func (m *Manager) refreshNetshStatus() {
	out, err := runner.Command("cmd", "/c", "netsh wlan show hostednetwork").Output()
	if err != nil {
		m.setFailedIfStarting("无法查询热点状态，可能当前系统不支持移动热点")
		return
	}

	output := string(out)
	lower := strings.ToLower(output)

	if strings.Contains(lower, "无法") || strings.Contains(lower, "cannot") || strings.Contains(lower, "not available") {
		m.setFailedIfStarting("热点开启失败，请检查无线网卡是否支持移动热点或是否已授权管理员权限")
		return
	}

	if strings.Contains(lower, "started") || strings.Contains(lower, "已启动") {
		m.setStarted("热点已开启")
		return
	}
}

func randomPassword(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			b[i] = chars[i%len(chars)]
			continue
		}
		b[i] = chars[n.Int64()]
	}
	return string(b)
}

// runElevated launches a program with the "runas" verb to trigger UAC.
func runElevated(program, args string) error {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecuteW := shell32.NewProc("ShellExecuteW")

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(program)
	params, _ := syscall.UTF16PtrFromString(args)

	ret, _, err := shellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		0,
		0, // SW_HIDE
	)
	if ret > 32 {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("ShellExecute failed with code %d", ret)
}

const winRTScript = `param(
    [Parameter(Mandatory=$true)]
    [string]$Ssid,
    [Parameter(Mandatory=$true)]
    [string]$Password,
    [Parameter(Mandatory=$true)]
    [string]$StatusFile
)

try {
    Add-Type -AssemblyName System.Runtime.WindowsRuntime
} catch {
    # System.Runtime.WindowsRuntime may not be available; continue anyway
}

function Write-Status($status, $message) {
    @{ status = $status; message = $message } | ConvertTo-Json -Compress | Out-File -FilePath $StatusFile -Encoding utf8 -Force
}

try {
    $networkInfoType = 'Windows.Networking.Connectivity.NetworkInformation, Windows.Networking.Connectivity, ContentType=WindowsRuntime'
    $tetheringManagerType = 'Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager, Windows.Networking.NetworkOperators, ContentType=WindowsRuntime'

    $profile = [Type]::GetType($networkInfoType)::GetInternetConnectionProfile()
    if ($profile -eq $null) {
        throw "无法获取网络连接配置文件"
    }

    $capability = [Type]::GetType($tetheringManagerType)::GetTetheringCapability($profile)
    $enabled = [Windows.Networking.NetworkOperators.TetheringCapability]::Enabled
    if ($capability -ne $enabled) {
        throw "移动热点当前不可用（状态: $capability），请检查无线网卡驱动或系统设置"
    }

    $manager = [Type]::GetType($tetheringManagerType)::CreateFromConnectionProfile($profile)

    $config = $manager.GetCurrentAccessPointConfiguration()
    $config.Ssid = $Ssid
    $config.Passphrase = $Password

    $configureOp = $manager.ConfigureAccessPointAsync($config)
    while ($configureOp.Status -eq [Windows.Foundation.AsyncStatus]::Started) {
        Start-Sleep -Milliseconds 200
    }
    if ($configureOp.Status -ne [Windows.Foundation.AsyncStatus]::Completed) {
        throw "配置热点失败: $($configureOp.ErrorCode)"
    }

    $startOp = $manager.StartTetheringAsync()
    while ($startOp.Status -eq [Windows.Foundation.AsyncStatus]::Started) {
        Start-Sleep -Milliseconds 200
    }
    if ($startOp.Status -eq [Windows.Foundation.AsyncStatus]::Completed) {
        Write-Status 'started' '热点已开启'
    } else {
        throw "启动热点失败: $($startOp.ErrorCode)"
    }
} catch {
    Write-Status 'failed' $_.Exception.Message
}
`
