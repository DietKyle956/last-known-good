package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Skill struct {
	Name        string
	Description string
	Body        string
}

type skillEntry struct {
	Name        string
	Description string
	path        string
}

type Loader struct {
	baseDir string
	entries []skillEntry
}

func NewLoader(baseDir string) *Loader {
	return &Loader{baseDir: baseDir}
}

func (l *Loader) Load() error {
	entries, err := os.ReadDir(l.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := filepath.Join(l.baseDir, entry.Name())
		mdFiles, err := filepath.Glob(filepath.Join(skillDir, "*.md"))
		if err != nil || len(mdFiles) == 0 {
			continue
		}
		data, err := os.ReadFile(mdFiles[0])
		if err != nil {
			continue
		}
		skill, err := parseSkillFile(string(data))
		if err != nil || skill == nil {
			continue
		}
		l.entries = append(l.entries, skillEntry{
			Name:        skill.Name,
			Description: skill.Description,
			path:        mdFiles[0],
		})
	}
	return nil
}

func (l *Loader) Summaries() []Skill {
	result := make([]Skill, len(l.entries))
	for i, e := range l.entries {
		result[i] = Skill{Name: e.Name, Description: e.Description}
	}
	return result
}

func (l *Loader) ReadBody(name string) (string, error) {
	for _, e := range l.entries {
		if e.Name == name {
			data, err := os.ReadFile(e.path)
			if err != nil {
				return "", err
			}
			body := extractBody(string(data))
			return body, nil
		}
	}
	return "", fmt.Errorf("skill %q not found", name)
}

func parseSkillFile(content string) (*Skill, error) {
	body := extractBody(content)
	trimmed := strings.TrimLeft(content, "\n\r\t ")
	if !strings.HasPrefix(trimmed, "---") {
		return nil, nil
	}
	rest := trimmed[3:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx == -1 {
		return nil, nil
	}
	frontmatter := rest[:endIdx]
	var name, description string
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		if strings.HasPrefix(value, "\"") && !strings.HasSuffix(value, "\"") {
			return nil, nil
		}
		value = strings.Trim(value, "\"")
		switch key {
		case "name":
			name = value
		case "description":
			description = value
		}
	}
	if name == "" || description == "" {
		return nil, nil
	}
	return &Skill{
		Name:        name,
		Description: description,
		Body:        body,
	}, nil
}

func extractBody(content string) string {
	trimmed := strings.TrimLeft(content, "\n\r\t ")
	if !strings.HasPrefix(trimmed, "---") {
		return ""
	}
	rest := trimmed[3:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx == -1 {
		return ""
	}
	body := rest[endIdx+4:]
	body = strings.TrimLeft(body, "\n\r\t ")
	return strings.TrimSpace(body)
}
