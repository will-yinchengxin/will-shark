package utils

import (
	"net"
	"strings"
)

func GetIps() ([]string, error) {
	var (
		ips = make([]string, 0)
	)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips, err
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips, nil
}

func ValidateIP(ip string) bool {
	// 检查是否为固定 IP 格式
	if net.ParseIP(ip) != nil {
		return true
	}

	// 检查是否为网段格式
	if strings.Contains(ip, "/") {
		parts := strings.Split(ip, "/")
		if len(parts) != 2 {
			return false
		}
		_, _, err := net.ParseCIDR(ip)
		if err != nil {
			return false
		}
		return true
	}

	// 检查是否为范围格式
	if strings.Contains(ip, "~") {
		parts := strings.Split(ip, "~")
		if len(parts) != 2 {
			return false
		}
		startIP := net.ParseIP(strings.TrimSpace(parts[0]))
		endIP := net.ParseIP(strings.TrimSpace(parts[1]))
		if startIP == nil || endIP == nil {
			return false
		}
		return true
	}

	return false
}

func InternalIp() string {
	infs, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, inf := range infs {
		if isEthDown(inf.Flags) || isLoopback(inf.Flags) {
			continue
		}

		addrs, err := inf.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}

	return ""
}

func isEthDown(f net.Flags) bool {
	return f&net.FlagUp != net.FlagUp
}

func isLoopback(f net.Flags) bool {
	return f&net.FlagLoopback == net.FlagLoopback
}
