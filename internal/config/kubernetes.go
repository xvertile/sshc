package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// K8sHost represents a Kubernetes pod connection configuration
type K8sHost struct {
	Name       string   `yaml:"name"`
	Namespace  string   `yaml:"namespace"`
	Pod        string   `yaml:"pod"`
	Container  string   `yaml:"container,omitempty"`
	Context    string   `yaml:"context,omitempty"`
	Kubeconfig string   `yaml:"kubeconfig,omitempty"`
	Shell      string   `yaml:"shell,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
}

// K8sConfig represents the kubernetes configuration file structure
type K8sConfig struct {
	Hosts []K8sHost `yaml:"hosts"`
}

// k8sMutex protects K8s config file operations from race conditions
var k8sMutex sync.Mutex

// GetK8sConfigPath returns the path to the k8s config file
func GetK8sConfigPath() (string, error) {
	configDir, err := GetSSHMConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "k8s.yaml"), nil
}

// K8sConfigExists checks if the k8s config file exists
func K8sConfigExists() bool {
	configPath, err := GetK8sConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(configPath)
	return err == nil
}

// ParseK8sConfig parses the k8s config file and returns the list of hosts
func ParseK8sConfig() ([]K8sHost, error) {
	configPath, err := GetK8sConfigPath()
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty list (feature is off)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return []K8sHost{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read k8s config: %w", err)
	}

	var config K8sConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse k8s config: %w", err)
	}

	// Apply defaults to each host
	for i := range config.Hosts {
		if config.Hosts[i].Shell == "" {
			config.Hosts[i].Shell = "/bin/bash"
		}
	}

	return config.Hosts, nil
}

// SaveK8sConfig saves the k8s configuration to file
func SaveK8sConfig(hosts []K8sHost) error {
	k8sMutex.Lock()
	defer k8sMutex.Unlock()

	configPath, err := GetK8sConfigPath()
	if err != nil {
		return err
	}

	// Ensure the config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	config := K8sConfig{Hosts: hosts}
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal k8s config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write k8s config: %w", err)
	}

	return nil
}

// AddK8sHost adds a new k8s host to the config
func AddK8sHost(host K8sHost) error {
	hosts, err := ParseK8sConfig()
	if err != nil {
		return err
	}

	// Check if host already exists
	for _, h := range hosts {
		if h.Name == host.Name {
			return fmt.Errorf("k8s host '%s' already exists", host.Name)
		}
	}

	// Apply defaults
	if host.Shell == "" {
		host.Shell = "/bin/bash"
	}

	hosts = append(hosts, host)
	return SaveK8sConfig(hosts)
}

// UpdateK8sHost updates an existing k8s host
func UpdateK8sHost(oldName string, newHost K8sHost) error {
	hosts, err := ParseK8sConfig()
	if err != nil {
		return err
	}

	found := false
	for i, h := range hosts {
		if h.Name == oldName {
			// Apply defaults
			if newHost.Shell == "" {
				newHost.Shell = "/bin/bash"
			}
			hosts[i] = newHost
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("k8s host '%s' not found", oldName)
	}

	return SaveK8sConfig(hosts)
}

// DeleteK8sHost removes a k8s host from the config
func DeleteK8sHost(name string) error {
	hosts, err := ParseK8sConfig()
	if err != nil {
		return err
	}

	var newHosts []K8sHost
	found := false
	for _, h := range hosts {
		if h.Name == name {
			found = true
			continue
		}
		newHosts = append(newHosts, h)
	}

	if !found {
		return fmt.Errorf("k8s host '%s' not found", name)
	}

	return SaveK8sConfig(newHosts)
}

// GetK8sHost retrieves a specific k8s host by name
func GetK8sHost(name string) (*K8sHost, error) {
	hosts, err := ParseK8sConfig()
	if err != nil {
		return nil, err
	}

	for _, h := range hosts {
		if h.Name == name {
			return &h, nil
		}
	}

	return nil, fmt.Errorf("k8s host '%s' not found", name)
}

// BuildKubectlCommand builds the kubectl exec command for a k8s host
func (h *K8sHost) BuildKubectlCommand() *exec.Cmd {
	args := []string{}

	// Add kubeconfig if specified
	if h.Kubeconfig != "" {
		args = append(args, "--kubeconfig", h.Kubeconfig)
	}

	// Add context if specified
	if h.Context != "" {
		args = append(args, "--context", h.Context)
	}

	// Add exec command with namespace and pod
	args = append(args, "exec", "-n", h.Namespace, "-it", h.Pod)

	// Add container if specified
	if h.Container != "" {
		args = append(args, "-c", h.Container)
	}

	// Add shell command
	shell := h.Shell
	if shell == "" {
		shell = "/bin/bash"
	}
	args = append(args, "--", shell)

	return exec.Command("kubectl", args...)
}

// K8sHostExists checks if a k8s host with the given name exists
func K8sHostExists(name string) (bool, error) {
	hosts, err := ParseK8sConfig()
	if err != nil {
		return false, err
	}

	for _, h := range hosts {
		if h.Name == name {
			return true, nil
		}
	}

	return false, nil
}
