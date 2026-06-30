package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSkillFileWithValidFrontmatter(t *testing.T) {
	content := `---
name: my-test-skill
description: A test skill for validating frontmatter parsing
---

This is the skill body content.
`
	skill, err := parseSkillFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected a skill, got nil")
	}
	if skill.Name != "my-test-skill" {
		t.Errorf("expected name %q, got %q", "my-test-skill", skill.Name)
	}
	if skill.Description != "A test skill for validating frontmatter parsing" {
		t.Errorf("expected description %q, got %q", "A test skill for validating frontmatter parsing", skill.Description)
	}
	if strings.TrimSpace(skill.Body) != "This is the skill body content." {
		t.Errorf("expected body %q, got %q", "This is the skill body content.", strings.TrimSpace(skill.Body))
	}
}

func TestParseSkillFileEmptyBody(t *testing.T) {
	content := `---
name: no-body-skill
description: A skill with no body
---
`
	skill, err := parseSkillFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected a skill, got nil")
	}
	if skill.Name != "no-body-skill" {
		t.Errorf("expected name %q, got %q", "no-body-skill", skill.Name)
	}
	if skill.Body != "" {
		t.Errorf("expected empty body, got %q", skill.Body)
	}
}

func TestParseSkillFileMissingFrontmatter(t *testing.T) {
	content := `# Just a regular markdown file
No frontmatter here.
`
	skill, err := parseSkillFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill != nil {
		t.Fatalf("expected nil skill for missing frontmatter, got %+v", skill)
	}
}

func TestParseSkillFileMalformedFrontmatter(t *testing.T) {
	content := `---
name: "unclosed
description: broken
---
Body text
`
	skill, err := parseSkillFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill != nil {
		t.Fatalf("expected nil skill for malformed frontmatter, got %+v", skill)
	}
}

func TestParseSkillFileFrontmatterMissingRequiredFields(t *testing.T) {
	content := `---
name: only-name
---
Body text
`
	skill, err := parseSkillFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill != nil {
		t.Fatalf("expected nil skill for frontmatter without description, got %+v", skill)
	}
}

func TestLoaderDiscoversSkillsFromDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create skill directories with markdown files
	skill1Dir := filepath.Join(dir, "skill-one")
	os.MkdirAll(skill1Dir, 0755)
	os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte(`---
name: skill-one
description: The first skill
---
Body of skill one
`), 0644)

	skill2Dir := filepath.Join(dir, "skill-two")
	os.MkdirAll(skill2Dir, 0755)
	os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte(`---
name: skill-two
description: The second skill
---
Body of skill two
`), 0644)

	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error loading skills: %v", err)
	}

	summaries := l.Summaries()
	if len(summaries) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(summaries))
	}

	got := map[string]string{}
	for _, s := range summaries {
		got[s.Name] = s.Description
	}
	if got["skill-one"] != "The first skill" {
		t.Errorf("expected skill-one description, got %q", got["skill-one"])
	}
	if got["skill-two"] != "The second skill" {
		t.Errorf("expected skill-two description, got %q", got["skill-two"])
	}

	// Body should NOT be loaded yet (lazy loading)
	for _, s := range summaries {
		if s.Body != "" {
			t.Errorf("expected empty body for %s during summaries, got %q", s.Name, s.Body)
		}
	}
}

func TestLoaderEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	summaries := l.Summaries()
	if len(summaries) != 0 {
		t.Fatalf("expected 0 skills, got %d", len(summaries))
	}
}

func TestLoaderReadSkillReturnsFullBody(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: my-skill
description: A lazy-loaded skill
---
# My Skill

This is the full body of my skill.
It spans multiple lines.
`), 0644)

	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify body is not loaded yet
	summaries := l.Summaries()
	if len(summaries) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(summaries))
	}
	if summaries[0].Body != "" {
		t.Fatal("body was loaded before ReadBody call")
	}

	// Now read the full body
	body, err := l.ReadBody("my-skill")
	if err != nil {
		t.Fatalf("unexpected error reading body: %v", err)
	}
	expectedBody := "# My Skill\n\nThis is the full body of my skill.\nIt spans multiple lines."
	if strings.TrimSpace(body) != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, strings.TrimSpace(body))
	}
}

func TestLoaderReadSkillUnknownName(t *testing.T) {
	dir := t.TempDir()
	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, err := l.ReadBody("nonexistent-skill")
	if err == nil {
		t.Fatal("expected error for unknown skill, got nil")
	}
	if body != "" {
		t.Errorf("expected empty body on error, got %q", body)
	}
}

func TestLoaderReadSkillNoBody(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "empty-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: empty-skill
description: A skill with no body
---
`), 0644)

	l := NewLoader(dir)
	if err := l.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := l.ReadBody("empty-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "" {
		t.Errorf("expected empty body, got %q", body)
	}
}
