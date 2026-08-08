package network

import (
	"net"
)

// GetLocalIP returns the best local IPv4 address for LAN access.
// It prefers RFC1918 private addresses and skips loopback/virtual interfaces.
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	// Prefer private addresses
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
			continue
		}
		if isPrivate(ipnet.IP) {
			return ipnet.IP.String()
		}
	}

	// Fallback to any non-loopback IPv4
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
			continue
		}
		return ipnet.IP.String()
	}

	return "127.0.0.1"
}

func isPrivate(ip net.IP) bool {
	// RFC1918 private ranges
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}
	return false
}

// IsLocalhost reports whether host is a loopback address or localhost name.
func IsLocalhost(host string) bool {
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
