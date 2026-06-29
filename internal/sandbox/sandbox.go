package sandbox

import (
	"fmt"
	"os/exec"
	"strings"
)

// Sandbox is the interface for sandbox lifecycle management.
type Sandbox interface {
	Start(projectDir string) (*SessionHandle, error)
	Exec(h *SessionHandle, command string) (string, error)
	Stop(h *SessionHandle) error
}

type SessionHandle struct {
	id string
}

func Start(projectDir string) (*SessionHandle, error) {
	out, err := exec.Command(
		"docker", "run", "-d",
		"--rm",
		"-v", projectDir+":/workspace",
		"-w", "/workspace",
		"alpine", "sleep", "infinity",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("docker run: %w", err)
	}
	id := strings.TrimSpace(string(out))
	return &SessionHandle{id: id}, nil
}

func Exec(h *SessionHandle, command string) (string, error) {
	args := append([]string{"exec", h.id, "sh", "-c"}, command)
	out, err := exec.Command("docker", args...).Output()
	if err != nil {
		return "", fmt.Errorf("docker exec: %w", err)
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
