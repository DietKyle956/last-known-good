package tui

import (
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

type messageType int

const (
	msgUser messageType = iota + 1
	msgAssistant
	msgToolCall
	msgToolResult
	msgError
)

type messageBubble struct {
	msgType messageType
	content string
	label   string
	model   string
}

func (m *Model) renderMessageBubble(msg messageBubble) string {
	switch msg.msgType {
	case msgUser:
		return m.renderUserBubble(msg)
	case msgAssistant:
		return m.renderAssistantBubble(msg)
	case msgToolCall:
		return m.renderToolCallBubble(msg)
	case msgToolResult:
		return m.renderToolResultBubble(msg)
	case msgError:
		return ErrorStyle.Render("  ✗ Error: " + msg.content)
	default:
		return msg.content
	}
}

func (m *Model) renderUserBubble(msg messageBubble) string {
	label := UserLabelStyle.Render("  You")
	if msg.model != "" {
		label = UserLabelStyle.Render("  You (" + msg.model + ")")
	}
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}
	content := UserBorderStyle.Width(width - 4).Render(msg.content)
	return lipgloss.JoinVertical(lipgloss.Top, label, content)
}

func (m *Model) renderAssistantBubble(msg messageBubble) string {
	label := AssistantLabelStyle.Render("  Assistant")
	if msg.model != "" {
		label = AssistantLabelStyle.Render("  Assistant (" + msg.model + ")")
	}
	rendered := m.renderMarkdown(msg.content)
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}
	content := AssistantBorderStyle.Width(width - 4).Render(rendered)
	return lipgloss.JoinVertical(lipgloss.Top, label, content)
}

func (m *Model) renderToolCallBubble(msg messageBubble) string {
	style := ToolHeaderStyle
	return style.Render("  ⚙ " + msg.content)
}

func (m *Model) renderToolResultBubble(msg messageBubble) string {
	return ToolResultStyle.Render("  " + msg.content)
}

func (m *Model) renderMarkdown(text string) string {
	if text == "" {
		return text
	}

	var b strings.Builder
	lines := strings.Split(text, "\n")

	inCodeBlock := false
	var codeBlock strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inCodeBlock {
			if strings.HasPrefix(trimmed, "```") {
				highlighted := highlightCode(codeBlock.String())
				b.WriteString(highlighted)
				codeBlock.Reset()
				inCodeBlock = false
			} else {
				codeBlock.WriteString(line)
				codeBlock.WriteString("\n")
			}
		} else if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = true
			codeBlock.Reset()
		} else {
			b.WriteString(renderInlineMarkdown(line))
			b.WriteString("\n")
		}
	}

	if inCodeBlock && codeBlock.Len() > 0 {
		highlighted := highlightCode(codeBlock.String())
		b.WriteString(highlighted)
	}

	return strings.TrimSuffix(b.String(), "\n")
}

func highlightCode(code string) string {
	lexer := lexers.Analyse(code)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.TTY16m

	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	var b strings.Builder
	err = formatter.Format(&b, style, it)
	if err != nil {
		return code
	}
	return b.String()
}

func renderInlineMarkdown(line string) string {
	if line == "" {
		return ""
	}

	result := line

	// Bold
	for {
		start := strings.Index(result, "**")
		if start == -1 {
			break
		}
		end := strings.Index(result[start+2:], "**")
		if end == -1 {
			break
		}
		end = start + 2 + end + 2
		bold := lipgloss.NewStyle().Bold(true).Render(result[start+2 : end-2])
		result = result[:start] + bold + result[end:]
	}

	// Italic
	for {
		start := strings.Index(result, "_")
		if start == -1 {
			break
		}
		end := strings.Index(result[start+1:], "_")
		if end == -1 {
			break
		}
		end = start + 1 + end + 1
		italic := lipgloss.NewStyle().Italic(true).Render(result[start+1 : end-1])
		result = result[:start] + italic + result[end:]
	}

	return result
}
