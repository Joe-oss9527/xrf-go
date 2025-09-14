package utils

import (
	"net"
)

// GetOutboundIP returns the preferred outbound IP of this machine by
// opening a UDP connection to a well-known address without sending data.
// It does not perform any external API calls and works even when HTTP
// egress is restricted.
func GetOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// IsPublicIP reports whether the provided IP string is a global unicast
// address (i.e., not loopback, link-local, or RFC1918 private ranges).
func IsPublicIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	// Only check IPv4 private ranges here; IPv6 ULA (fc00::/7) treated non-public
	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == 10 {
			return false
		}
		// 172.16.0.0/12
		if ip4[0] == 172 && ip4[1]&0xf0 == 16 {
			return false
		}
		// 192.168.0.0/16
		if ip4[0] == 192 && ip4[1] == 168 {
			return false
		}
		// 169.254.0.0/16 (link-local)
		if ip4[0] == 169 && ip4[1] == 254 {
			return false
		}
		// 127.0.0.0/8 loopback covered by IsLoopback
	} else {
		// IPv6: exclude ULA fc00::/7
		if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
			return false
		}
	}
	return true
}
