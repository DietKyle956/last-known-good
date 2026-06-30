package tui

import (
	"fmt"
	"strings"
)

type toolCallCard struct {
	name      string
	arguments string
	result    string
	isError   bool
	collapsed bool
}

func (m *Model) renderToolCallCard(tc toolCallCard) string {
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	indicator := SuccessStyle.Render("✓")
	if tc.isError {
		indicator = ErrorStyle.Render("✗")
	}

	summary := fmt.Sprintf("%s %s(%s)", indicator, tc.name, tc.arguments)

	if tc.collapsed {
		return ToolBorderStyle.Width(width - 4).Render(summary)
	}

	var b strings.Builder
	b.WriteString(summary)
	b.WriteString("\n")

	if tc.result != "" {
		style := ToolResultStyle
		if tc.isError {
			style = ErrorStyle
		}
		b.WriteString(style.Render("  " + tc.result))
	}

	return ToolBorderStyle.Width(width - 4).Render(b.String())
}
