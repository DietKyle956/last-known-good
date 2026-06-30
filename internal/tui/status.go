package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type statusData struct {
	modelName    string
	tokenCount   int
	sandboxState string
}

func (m *Model) renderStatusBar() string {
	if !m.ready {
		return ""
	}

	s := m.status

	modelText := ""
	if s.modelName != "" {
		modelText = fmt.Sprintf("  💬 %s", s.modelName)
	}

	tokenText := ""
	if s.tokenCount > 0 {
		tokenText = fmt.Sprintf("  ⚡ %d tok", s.tokenCount)
	}

	sandboxText := ""
	if s.sandboxState != "" {
		sandboxText = fmt.Sprintf("  🐳 %s", s.sandboxState)
	}

	parts := []string{}
	if modelText != "" {
		parts = append(parts, modelText)
	}
	if tokenText != "" {
		parts = append(parts, tokenText)
	}
	if sandboxText != "" {
		parts = append(parts, sandboxText)
	}

	content := strings.Join(parts, " │")
	statusWidth := m.width
	if statusWidth <= 0 {
		statusWidth = 80
	}

	rendered := StatusBarStyle.Width(statusWidth).Render(content)
	divider := DividerStyle.Render(strings.Repeat("─", statusWidth))
	return lipgloss.JoinVertical(lipgloss.Top, divider, rendered)
}
