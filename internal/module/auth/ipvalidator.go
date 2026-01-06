package auth

import (
	"net"
	"strings"
)

// IPValidator validates IP addresses against a whitelist.
type IPValidator struct{}

// NewIPValidator creates a new IP validator.
func NewIPValidator() *IPValidator {
	return &IPValidator{}
}

// IsAllowed checks if the given IP is allowed by the whitelist.
// If whitelist is empty, all IPs are allowed.
func (v *IPValidator) IsAllowed(clientIP string, whitelist []string) bool {
	if len(whitelist) == 0 {
		return true
	}

	// Parse client IP
	ip := net.ParseIP(clientIP)
	if ip == nil {
		// Try to extract IP from IP:port format
		host, _, err := net.SplitHostPort(clientIP)
		if err != nil {
			return false
		}
		ip = net.ParseIP(host)
		if ip == nil {
			return false
		}
	}

	for _, allowed := range whitelist {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}

		// Check if it's a CIDR range
		if strings.Contains(allowed, "/") {
			_, network, err := net.ParseCIDR(allowed)
			if err != nil {
				continue
			}
			if network.Contains(ip) {
				return true
			}
		} else {
			// Single IP address
			allowedIP := net.ParseIP(allowed)
			if allowedIP != nil && ip.Equal(allowedIP) {
				return true
			}
		}
	}

	return false
}

// ValidateCIDR checks if a string is a valid CIDR or IP address.
func ValidateCIDR(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	if strings.Contains(s, "/") {
		_, _, err := net.ParseCIDR(s)
		return err == nil
	}

	return net.ParseIP(s) != nil
}

// ValidateIPList validates a list of IP addresses or CIDR ranges.
func ValidateIPList(ips []string) []string {
	var invalid []string
	for _, ip := range ips {
		if !ValidateCIDR(ip) {
			invalid = append(invalid, ip)
		}
	}
	return invalid
}
