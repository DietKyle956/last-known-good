package sandbox

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// Sandbox is the interface for sandbox lifecycle management.
type Sandbox interface {
	Start(projectDir string, cfg SandboxConfig) (*SessionHandle, error)
	Exec(h *SessionHandle, command string) (string, error)
	Stop(h *SessionHandle) error
}

// SandboxConfig controls sandbox network policy and resource limits.
type SandboxConfig struct {
	Network *NetworkConfig
	CPU     string
	Memory  string
}

// NetworkConfig controls outbound network access from the sandbox.
type NetworkConfig struct {
	Allow []string
}

type SessionHandle struct {
	id string
}

func Start(projectDir string, cfg SandboxConfig) (*SessionHandle, error) {
	args := []string{"run", "-d", "--rm"}
	args = append(args, "-v", projectDir+":/workspace", "-w", "/workspace")

	if cfg.Network == nil || len(cfg.Network.Allow) == 0 {
		args = append(args, "--network=none")
	}

	if cfg.CPU != "" {
		args = append(args, "--cpus", cfg.CPU)
	}
	if cfg.Memory != "" {
		args = append(args, "--memory", cfg.Memory)
	}

	args = append(args, "alpine", "sleep", "infinity")

	out, err := exec.Command("docker", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("docker run: %w", err)
	}
	id := strings.TrimSpace(string(out))
	h := &SessionHandle{id: id}

	if cfg.Network != nil && len(cfg.Network.Allow) > 0 {
		if err := configureAllowlist(h, cfg.Network.Allow); err != nil {
			stopErr := Stop(h)
			if stopErr != nil {
				return nil, fmt.Errorf("configure allowlist failed: %w; stop also failed: %v", err, stopErr)
			}
			return nil, fmt.Errorf("configure allowlist: %w", err)
		}
	}

	return h, nil
}

func lookupIP(domain string) string {
	ips, err := net.LookupIP(domain)
	if err != nil || len(ips) == 0 {
		return ""
	}
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}
	return ips[0].String()
}

func configureAllowlist(h *SessionHandle, domains []string) error {
	for _, domain := range domains {
		ip := lookupIP(domain)
		if ip == "" {
			return fmt.Errorf("could not resolve %s", domain)
		}
		_, err := Exec(h, fmt.Sprintf("echo '%s %s' >> /etc/hosts", ip, domain))
		if err != nil {
			return fmt.Errorf("add host entry %s: %w", domain, err)
		}
	}
	_, err := Exec(h, "echo 'nameserver 127.0.0.1' > /etc/resolv.conf")
	if err != nil {
		return fmt.Errorf("block dns: %w", err)
	}
	return nil
}

func Exec(h *SessionHandle, command string) (string, error) {
	args := append([]string{"exec", "-w", "/workspace", h.id, "sh", "-c"}, command)
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("docker exec: %w", err)
	}
	return string(out), nil
}

func Stop(h *SessionHandle) error {
	out, err := exec.Command("docker", "rm", "-f", h.id).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm: %w: %s", err, string(out))
	}
	return nil
}
