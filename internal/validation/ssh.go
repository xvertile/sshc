package validation

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ValidateHostname checks if a hostname is valid
func ValidateHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}
	if strings.HasPrefix(hostname, ".") || strings.HasSuffix(hostname, ".") {
		return false
	}
	if strings.Contains(hostname, " ") {
		return false
	}

	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	return hostnameRegex.MatchString(hostname)
}

// ValidateIP checks if an IP address is valid
func ValidateIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ValidatePort checks if a port is valid
func ValidatePort(port string) bool {
	if port == "" {
		return true // Empty port defaults to 22
	}
	portNum, err := strconv.Atoi(port)
	return err == nil && portNum >= 1 && portNum <= 65535
}

// ValidateHostName checks if a host name is valid for SSH config
func ValidateHostName(name string) bool {
	if len(name) == 0 || len(name) > 50 {
		return false
	}
	// Host name cannot contain whitespace or special SSH config characters
	return !strings.ContainsAny(name, " \t\n\r#")
}

// ValidateIdentityFile checks if an identity file path is valid
func ValidateIdentityFile(path string) bool {
	if path == "" {
		return true // Optional field
	}
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		path = filepath.Join(homeDir, path[2:])
	}
	_, err := os.Stat(path)
	return err == nil
}

// ValidateHost validates all host fields
func ValidateHost(name, hostname, port, identity string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("host name is required")
	}

	if !ValidateHostName(name) {
		return fmt.Errorf("invalid host name: cannot contain spaces or special characters")
	}

	if strings.TrimSpace(hostname) == "" {
		return fmt.Errorf("hostname/IP is required")
	}

	if !ValidateHostname(hostname) && !ValidateIP(hostname) {
		return fmt.Errorf("invalid hostname or IP address format")
	}

	if !ValidatePort(port) {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if identity != "" && !ValidateIdentityFile(identity) {
		return fmt.Errorf("identity file does not exist: %s", identity)
	}

	return nil
}
