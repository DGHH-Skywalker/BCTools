//go:build windows

package network

import "golang.org/x/sys/windows"

// GetWindowsInfo returns whether the server is running on Windows and the
// simplified version string. Only Windows 10 and 11 are reported as "10"/"11";
// other versions return an empty version string.
func GetWindowsInfo() (isWindows bool, version string) {
	info := windows.RtlGetVersion()
	if info == nil {
		return true, ""
	}
	// Windows 11 is still major 10 with build number >= 22000.
	if info.MajorVersion == 10 {
		if info.BuildNumber >= 22000 {
			return true, "11"
		}
		return true, "10"
	}
	if info.MajorVersion > 10 {
		return true, ""
	}
	return true, ""
}
