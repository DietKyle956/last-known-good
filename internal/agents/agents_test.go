package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseValidFrontmatter(t *testing.T) {
	content := `---
full_name: Mild Executive Function
short_name: The Conductor
role: Orchestrates, routes
---
This is the system prompt for the conductor.
It can span multiple lines.
`
	agent, err := parseAgentFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected an agent, got nil")
	}
	if agent.Name != "Mild Executive Function" {
		t.Errorf("expected Name %q, got %q", "Mild Executive Function", agent.Name)
	}
	if agent.ShortName != "The Conductor" {
		t.Errorf("expected ShortName %q, got %q", "The Conductor", agent.ShortName)
	}
	if agent.Role != "Orchestrates, routes" {
		t.Errorf("expected Role %q, got %q", "Orchestrates, routes", agent.Role)
	}
	if strings.TrimSpace(agent.SystemPrompt) != "This is the system prompt for the conductor.\nIt can span multiple lines." {
		t.Errorf("unexpected system prompt: %q", agent.SystemPrompt)
	}
}

func TestParseFrontmatterWithOptionalFields(t *testing.T) {
	content := `---
full_name: Engineer-Initial
short_name: Engineer-Initial
role: First-pass planning
allowed_tools: read_file, grep, glob, git_diff
model_preference: deepseek-v4-pro
---
You are the engineer initial.
`
	agent, err := parseAgentFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected an agent, got nil")
	}
	if agent.Name != "Engineer-Initial" {
		t.Errorf("expected Name %q, got %q", "Engineer-Initial", agent.Name)
	}
	if agent.ShortName != "Engineer-Initial" {
		t.Errorf("expected ShortName %q, got %q", "Engineer-Initial", agent.ShortName)
	}
	if agent.Role != "First-pass planning" {
		t.Errorf("expected Role %q, got %q", "First-pass planning", agent.Role)
	}
	if agent.ModelPreference != "deepseek-v4-pro" {
		t.Errorf("expected ModelPreference %q, got %q", "deepseek-v4-pro", agent.ModelPreference)
	}
	if len(agent.AllowedTools) != 4 {
		t.Fatalf("expected 4 allowed tools, got %d", len(agent.AllowedTools))
	}
	expected := []string{"read_file", "grep", "glob", "git_diff"}
	for i, v := range expected {
		if agent.AllowedTools[i] != v {
			t.Errorf("allowed_tools[%d] = %q, want %q", i, agent.AllowedTools[i], v)
		}
	}
	if strings.TrimSpace(agent.SystemPrompt) != "You are the engineer initial." {
		t.Errorf("unexpected system prompt: %q", agent.SystemPrompt)
	}
}

func TestParseMissingFrontmatter(t *testing.T) {
	content := `# Just a regular markdown file
No frontmatter here.
`
	agent, err := parseAgentFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent != nil {
		t.Fatalf("expected nil for missing frontmatter, got %+v", agent)
	}
}

func TestParseMalformedFrontmatter(t *testing.T) {
	content := `---
full_name: "unclosed
short_name: broken
role: test
---
Body
`
	agent, err := parseAgentFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent != nil {
		t.Fatalf("expected nil for malformed frontmatter, got %+v", agent)
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	content := `---
full_name: only-name
short_name: only-short
---
Body
`
	agent, err := parseAgentFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent != nil {
		t.Fatalf("expected nil for frontmatter without role, got %+v", agent)
	}
}

func TestParseEmptyBodyFallsBackToDefaultPersona(t *testing.T) {
	content := `---
full_name: No Body
short_name: nobody
role: A role with no body
---
`
	agent, err := parseAgentFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected an agent, got nil")
	}
	if agent.SystemPrompt != defaultPersona {
		t.Errorf("expected default persona, got %q", agent.SystemPrompt)
	}
}

func TestLoaderDiscoversAgentsFromDirectory(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "conductor.md", `---
full_name: Mild Executive Function
short_name: The Conductor
role: Orchestrates, routes
---
Conductor prompt.
`)
	writeAgentFile(t, dir, "engineer.md", `---
full_name: Experiencing Significant Enthusiasm
short_name: Engineer-Initial
role: First-pass planning
---
Engineer prompt.
`)

	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agents := l.List()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
}

func TestLoaderEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agents := l.List()
	if len(agents) != 0 {
		t.Fatalf("expected 0 agents, got %d", len(agents))
	}
}

func TestLoaderDuplicateShortNameError(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "one.md", `---
full_name: Agent One
short_name: duplicate
role: First agent
---
One.
`)
	writeAgentFile(t, dir, "two.md", `---
full_name: Agent Two
short_name: duplicate
role: Second agent
---
Two.
`)

	l := NewLoader(dir)
	err := l.Load()
	if err == nil {
		t.Fatal("expected error for duplicate short name, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected error mentioning duplicate, got: %v", err)
	}
}

func TestRegistryGetByShortName(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "conductor.md", `---
full_name: Mild Executive Function
short_name: The Conductor
role: Orchestrates, routes
---
Conductor system prompt.
`)

	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agent, err := l.Get("The Conductor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Name != "Mild Executive Function" {
		t.Errorf("expected Name %q, got %q", "Mild Executive Function", agent.Name)
	}
	if agent.Role != "Orchestrates, routes" {
		t.Errorf("expected Role %q, got %q", "Orchestrates, routes", agent.Role)
	}
}

func TestRegistryGetUnknownShortNameError(t *testing.T) {
	dir := t.TempDir()
	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agent, err := l.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown short name, got nil")
	}
	if agent != nil {
		t.Errorf("expected nil agent on error, got %+v", agent)
	}
}

func TestLoaderSkipsNonMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "agent.md", `---
full_name: Valid Agent
short_name: valid
role: A valid agent
---
Valid.
`)
	// Non-markdown file should be skipped
	if err := writeFile(dir, "notes.txt", "this is not an agent file"); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agents := l.List()
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
}

func TestLoaderSkipsMalformedAgentFile(t *testing.T) {
	dir := t.TempDir()
	// Valid agent
	writeAgentFile(t, dir, "valid.md", `---
full_name: Valid
short_name: valid
role: Valid agent
---
Body.
`)
	// Malformed agent (no frontmatter)
	if err := writeFile(dir, "bad.md", "# Just a heading\nNo frontmatter here."); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agents := l.List()
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
}

func TestValidateAllowedTools(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "agent.md", `---
full_name: Restricted Agent
short_name: restricted
role: Agent with restricted tools
allowed_tools: read_file, grep, nonexistent_tool
---
Body.
`)

	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err := l.ValidateTools([]string{"read_file", "write_file", "grep", "glob", "git_diff"})
	if err == nil {
		t.Fatal("expected error for nonexistent tool, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent_tool") {
		t.Errorf("expected error mentioning nonexistent_tool, got: %v", err)
	}
}

func TestValidateAllowedToolsAllValid(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "agent.md", `---
full_name: Restricted Agent
short_name: restricted
role: Agent with restricted tools
allowed_tools: read_file, grep, glob
---
Body.
`)

	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := l.ValidateTools([]string{"read_file", "write_file", "grep", "glob", "git_diff"}); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidateAllowedToolsNoRestrictionPasses(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "agent.md", `---
full_name: Unrestricted Agent
short_name: unrestricted
role: Agent with no tool restrictions
---
Body.
`)

	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := l.ValidateTools([]string{"read_file", "write_file"}); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestDefaultRosterLoads(t *testing.T) {
	// Load from the project's agents/ directory
	l := NewLoader("../../agents")
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error loading default agents: %v", err)
	}
	agents := l.List()
	if len(agents) != 11 {
		t.Fatalf("expected 11 default agents, got %d", len(agents))
	}
	// Verify each expected agent by short name
	expected := map[string]string{
		"The Conductor":     "Mild Executive Function",
		"Engineer-Initial":  "Experiencing Significant Enthusiasm",
		"Engineer-Replan":   "Reconsidering Previous Position",
		"The Validator":     "Resistance Remains Warranted",
		"Infrastructure":    "Sleeping Until Needed",
		"Test Writer":       "Killing It Softly",
		"Test Reviewer":     "Prepared To Be Disappointed",
		"Coder":             "Acting On Assumptions",
		"The Auditor":       "Assuming The Worst",
		"Scribe":            "Quietly Remaining Confident",
		"Historian":         "As Previously Discussed",
	}
	for shortName, fullName := range expected {
		agent, err := l.Get(shortName)
		if err != nil {
			t.Errorf("agent %q not found: %v", shortName, err)
			continue
		}
		if agent.Name != fullName {
			t.Errorf("agent %q has Name %q, want %q", shortName, agent.Name, fullName)
		}
	}
}

func writeAgentFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := writeFile(dir, name, content); err != nil {
		t.Fatalf("failed to write agent file: %v", err)
	}
}

func writeFile(dir, name, content string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
}
