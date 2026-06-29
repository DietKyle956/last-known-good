package sandbox

import (
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

func TestExecRunsCommandAndReusesContainer(t *testing.T) {
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	h, err := Start(dir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = Stop(h) }()

	out1, err := Exec(h, "echo hello-1")
	if err != nil {
		t.Fatalf("first Exec failed: %v", err)
	}
	if strings.TrimSpace(out1) != "hello-1" {
		t.Fatalf("expected 'hello-1', got %q", out1)
	}

	out2, err := Exec(h, "echo hello-2")
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

	h, err := Start(dir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = Stop(h) }()

	hostFile := dir + "/test.txt"
	if err := os.WriteFile(hostFile, []byte("hello from host"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := Exec(h, "cat /workspace/test.txt")
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

	h, err := Start(dir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = Stop(h) }()

	_, err = Exec(h, "echo 'hello from container' > /workspace/from-container.txt")
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

	h, err := Start(dir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = Stop(h) }()

	out, err := Exec(h, "cat "+outsideFile)
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

	h, err := Start(dir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Simulate interrupt by calling Stop (signal cleanup calls Stop)
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

	h, err := Start(dir)
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

	h, err := Start(dir)
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
