package main

import (
	"os/exec"
	"testing"
)

func TestBinaryBuilds(t *testing.T) {
	build := exec.Command("go", "build", "-o", "/dev/null", ".")
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
}

func TestBinaryPrintsHelp(t *testing.T) {
	build := exec.Command("go", "build", "-o", "/tmp/lkg-agent-test", ".")
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	help := exec.Command("/tmp/lkg-agent-test", "--help")
	out, err = help.CombinedOutput()
	if err != nil {
		t.Fatalf("help failed: %v\n%s", err, out)
	}
	if len(out) == 0 {
		t.Fatal("help output is empty")
	}
}
