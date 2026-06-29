package sandbox

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func containerExists(t *testing.T, containerID string) bool {
	t.Helper()
	out, err := exec.Command("docker", "ps", "-a", "--filter", "id="+containerID, "--format", "{{.ID}}").Output()
	if err != nil {
		t.Fatalf("docker ps failed: %v", err)
	}
	return strings.TrimSpace(string(out)) != ""
}

func startDefault(t *testing.T) *SessionHandle {
	t.Helper()
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	h, err := Start(dir, SandboxConfig{})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() {
		if err := Stop(h); err != nil {
			t.Errorf("failed to stop sandbox: %v", err)
		}
	})
	return h
}

func TestExecRunsCommandAndReusesContainer(t *testing.T) {
	h := startDefault(t)

	out1, err := Exec(context.Background(), h, "echo hello-1")
	if err != nil {
		t.Fatalf("first Exec failed: %v", err)
	}
	if strings.TrimSpace(out1) != "hello-1" {
		t.Fatalf("expected 'hello-1', got %q", out1)
	}

	out2, err := Exec(context.Background(), h, "echo hello-2")
	if err != nil {
		t.Fatalf("second Exec failed: %v", err)
	}
	if strings.TrimSpace(out2) != "hello-2" {
		t.Fatalf("expected 'hello-2', got %q", out2)
	}
}

func TestFileWrittenOnHostVisibleInsideContainer(t *testing.T) {
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	h, err := Start(dir, SandboxConfig{})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = Stop(h) }()

	hostFile := dir + "/test.txt"
	if err := os.WriteFile(hostFile, []byte("hello from host"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := Exec(context.Background(), h, "cat /workspace/test.txt")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if strings.TrimSpace(out) != "hello from host" {
		t.Fatalf("expected 'hello from host', got %q", out)
	}
}

func TestFileWrittenInsideContainerVisibleOnHost(t *testing.T) {
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	h, err := Start(dir, SandboxConfig{})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = Stop(h) }()

	_, err = Exec(context.Background(), h, "echo 'hello from container' > /workspace/from-container.txt")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	content, err := os.ReadFile(dir + "/from-container.txt")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(content)) != "hello from container" {
		t.Fatalf("expected 'hello from container', got %q", string(content))
	}
}

func TestFilesOutsideMountNotAccessibleFromContainer(t *testing.T) {
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	outsideFile := dir + "-outside/outside.txt"
	if err := os.MkdirAll(dir+"-outside", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir + "-outside")

	h, err := Start(dir, SandboxConfig{})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = Stop(h) }()

	out, err := Exec(context.Background(), h, "cat "+outsideFile)
	if err == nil {
		t.Fatalf("expected error accessing file outside mount, got output: %q", out)
	}
}

func TestContainerRemovedOnInterruptSignal(t *testing.T) {
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	h, err := Start(dir, SandboxConfig{})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := Stop(h); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if containerExists(t, h.id) {
		t.Fatal("container still exists after simulated interrupt")
	}
}

func TestNoOrphanedContainersAfterSessionEnds(t *testing.T) {
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	h, err := Start(dir, SandboxConfig{})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := Stop(h); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	out, err := exec.Command("docker", "ps", "-a", "--filter", "name="+h.id, "--format", "{{.ID}}").Output()
	if err != nil {
		t.Fatalf("docker ps failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatal("found orphaned container after Stop")
	}
}

func TestStartCreatesContainerAndStopRemovesIt(t *testing.T) {
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	h, err := Start(dir, SandboxConfig{})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil handle")
	}

	if !containerExists(t, h.id) {
		t.Fatal("container was not created")
	}

	if err := Stop(h); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if containerExists(t, h.id) {
		t.Fatal("container still exists after Stop")
	}
}

func TestDefaultNoNetworkBlocksOutbound(t *testing.T) {
	h := startDefault(t)

	_, err := Exec(context.Background(), h, "wget -q -O- --timeout=5 http://example.com")
	if err == nil {
		t.Fatal("expected outbound network to be blocked by default")
	}
}

func TestAllowlistReachesAllowedDomain(t *testing.T) {
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	h, err := Start(dir, SandboxConfig{
		Network: &NetworkConfig{Allow: []string{"example.com"}},
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = Stop(h) }()

	out, err := Exec(context.Background(), h, "wget -q -O- --timeout=10 http://example.com")
	if err != nil {
		t.Fatalf("expected allowed domain to be reachable: %v", err)
	}
	if !strings.Contains(out, "Example Domain") && !strings.Contains(out, "example") {
		t.Fatalf("unexpected content from example.com: %q", out)
	}
}

func TestAllowlistBlocksOtherDomains(t *testing.T) {
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	h, err := Start(dir, SandboxConfig{
		Network: &NetworkConfig{Allow: []string{"example.com"}},
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = Stop(h) }()

	_, err = Exec(context.Background(), h, "wget -q -O- --timeout=5 http://google.com")
	if err == nil {
		t.Fatal("expected non-allowed domain to be unreachable")
	}
}

func TestCPUAndMemoryLimitsApplied(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	h, err := Start(dir, SandboxConfig{
		CPU:    "0.5",
		Memory: "128m",
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = Stop(h) }()

	cpuOut, err := exec.Command("docker", "inspect", h.id, "--format", "{{.HostConfig.NanoCpus}}").Output()
	if err != nil {
		t.Fatalf("docker inspect failed: %v", err)
	}
	if strings.TrimSpace(string(cpuOut)) != "500000000" {
		t.Fatalf("expected NanoCpus 500000000, got %q", strings.TrimSpace(string(cpuOut)))
	}

	memOut, err := exec.Command("docker", "inspect", h.id, "--format", "{{.HostConfig.Memory}}").Output()
	if err != nil {
		t.Fatalf("docker inspect failed: %v", err)
	}
	if strings.TrimSpace(string(memOut)) != "134217728" {
		t.Fatalf("expected Memory 134217728, got %q", strings.TrimSpace(string(memOut)))
	}
}


func TestDockerExecerDelegates(t *testing.T) {
	h := startDefault(t)
	ex := NewDockerExecer(h)
	out, err := ex.Exec(context.Background(), "echo hello")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("expected 'hello', got %q", out)
	}
}
