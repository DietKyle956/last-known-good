package main

import (
	"os"
	"strings"
	"testing"
)

func TestCIWorkflowExists(t *testing.T) {
	path := ".github/workflows/ci.yml"
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Fatal("ci.yml not found")
	}
	if err != nil {
		t.Fatalf("failed to stat ci.yml: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("ci.yml is empty")
	}
}

func TestCIWorkflowHasBuildJob(t *testing.T) {
	path := ".github/workflows/ci.yml"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "build:") && !strings.Contains(content, "build") {
		t.Fatal("ci.yml should contain a build step")
	}
}

func TestCIWorkflowHasLintCheck(t *testing.T) {
	path := ".github/workflows/ci.yml"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "Vet") && !strings.Contains(content, "lint") {
		t.Fatal("ci.yml should contain a lint or vet step")
	}
}

func TestCIWorkflowRunsOnPushAndPR(t *testing.T) {
	path := ".github/workflows/ci.yml"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "push:") && !strings.Contains(content, "push") {
		t.Fatal("ci.yml should trigger on push")
	}
	if !strings.Contains(content, "pull_request:") && !strings.Contains(content, "pull_request") {
		t.Fatal("ci.yml should trigger on pull_request")
	}
}
