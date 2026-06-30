package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

func newTextarea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Prompt = "> "
	ta.CharLimit = 0
	ta.Focus()

	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Background(lipgloss.Color(ColorBackground)).
		Foreground(lipgloss.Color(ColorPrimaryText))
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorAccent))
	ta.FocusedStyle.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPrimaryText))
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Background(lipgloss.Color(ColorBackground))

	ta.BlurredStyle.Base = lipgloss.NewStyle().
		Background(lipgloss.Color(ColorBackground)).
		Foreground(lipgloss.Color(ColorPrimaryText))
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorAccent))
	ta.BlurredStyle.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPrimaryText))

	ta.ShowLineNumbers = false

	return ta
}
