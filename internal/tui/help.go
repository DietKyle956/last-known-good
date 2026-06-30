package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Send     key.Binding
	Quit     key.Binding
	Help     key.Binding
	Collapse key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Send, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Send, k.Collapse},
		{k.Help, k.Quit},
	}
}

var keys = keyMap{
	Send: key.NewBinding(
		key.WithKeys("alt+enter", "ctrl+j"),
		key.WithHelp("alt+enter", "send message"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "ctrl+d"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("ctrl+h"),
		key.WithHelp("ctrl+h", "toggle help"),
	),
	Collapse: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "toggle tool collapse"),
	),
}

func newHelpModel() help.Model {
	h := help.New()
	h.Styles = help.Styles{
		ShortKey:       KeyShortcutStyle,
		ShortDesc:      KeyDescStyle,
		ShortSeparator: KeySepStyle,
		Ellipsis:       KeyEllipsisStyle,
		FullKey:        KeyShortcutStyle,
		FullDesc:       KeyDescStyle,
		FullSeparator:  KeySepStyle,
	}
	return h
}
