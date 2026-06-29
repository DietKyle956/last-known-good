package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestDockerComposeConfigIsValid(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}
	if _, err := os.Stat("docker-compose.yml"); os.IsNotExist(err) {
		t.Fatal("docker-compose.yml not found at project root")
	}
	cmd := exec.Command("docker", "compose", "config", "--quiet")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docker compose config failed: %v\n%s", err, out)
	}
}
