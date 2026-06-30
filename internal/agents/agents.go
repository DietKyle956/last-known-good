package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultPersona = "You are a helpful software development assistant."

type Agent struct {
	Name            string
	ShortName       string
	Role            string
	AllowedTools    []string
	ModelPreference string
	SystemPrompt    string
}

type Loader struct {
	baseDir string
	agents  []Agent
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
	seen := make(map[string]string)
	l.agents = nil
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(l.baseDir, entry.Name()))
		if err != nil {
			continue
		}
		agent, err := parseAgentFile(string(data))
		if err != nil || agent == nil {
			continue
		}
		if existing, ok := seen[agent.ShortName]; ok {
			return fmt.Errorf("duplicate short_name %q in files %q and %q", agent.ShortName, existing, entry.Name())
		}
		seen[agent.ShortName] = entry.Name()
		l.agents = append(l.agents, *agent)
	}
	return nil
}

func (l *Loader) List() []Agent {
	result := make([]Agent, len(l.agents))
	copy(result, l.agents)
	return result
}

func (l *Loader) ValidateTools(validTools []string) error {
	valid := make(map[string]bool, len(validTools))
	for _, t := range validTools {
		valid[t] = true
	}
	for _, a := range l.agents {
		for _, t := range a.AllowedTools {
			if !valid[t] {
				return fmt.Errorf("agent %q lists tool %q which does not exist in the tool registry", a.ShortName, t)
			}
		}
	}
	return nil
}

func (l *Loader) Get(shortName string) (*Agent, error) {
	for _, a := range l.agents {
		if a.ShortName == shortName {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("agent %q not found", shortName)
}

func parseAgentFile(content string) (*Agent, error) {
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
	body := strings.TrimSpace(rest[endIdx+4:])

	var name, shortName, role, modelPref string
	var allowedTools []string
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
		case "full_name":
			name = value
		case "short_name":
			shortName = value
		case "role":
			role = value
		case "allowed_tools":
			for _, t := range strings.Split(value, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					allowedTools = append(allowedTools, t)
				}
			}
		case "model_preference":
			modelPref = value
		}
	}
	if name == "" || shortName == "" || role == "" {
		return nil, nil
	}
	systemPrompt := body
	if systemPrompt == "" {
		systemPrompt = defaultPersona
	}
	return &Agent{
		Name:            name,
		ShortName:       shortName,
		Role:            role,
		AllowedTools:    allowedTools,
		ModelPreference: modelPref,
		SystemPrompt:    systemPrompt,
	}, nil
}
