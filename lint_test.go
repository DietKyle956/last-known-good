package main

import (
	"os"
	"testing"
)

func TestGolangciLintConfigExists(t *testing.T) {
	paths := []string{".golangci.yml", ".golangci.yaml"}
	var found bool
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no .golangci.yml or .golangci.yaml found")
	}
}

func TestGolangciLintConfigHasContent(t *testing.T) {
	path := ".golangci.yml"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = ".golangci.yaml"
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("golangci config is empty")
	}
}
