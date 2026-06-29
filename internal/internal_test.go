package internal

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAllInternalPackagesCompile(t *testing.T) {
	pkgs := []string{
		"agent",
		"llm",
		"sandbox",
		"store",
	}
	root, _ := filepath.Abs("..")
	for _, name := range pkgs {
		build := exec.Command("go", "build", filepath.Join(root, "internal", name))
		out, err := build.CombinedOutput()
		if err != nil {
			t.Errorf("internal/%s failed to compile: %v\n%s", name, err, out)
		}
	}
}
