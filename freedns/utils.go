package freedns

import "net"

// Parse ip with optional port, return normalized ip:port string
// For ips without port, default 53 port is appended
func normalizeDnsAddress(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// no port, try parse addr as host with default port
		host = addr
		port = "53"
	} else if host == "" {
		// for addrs like ":53", use default host
		host = "0.0.0.0"
	}

	if net.ParseIP(host) == nil {
		return "", Error("Invalid IP addr: " + host)
	}
	return net.JoinHostPort(host, port), nil
}
