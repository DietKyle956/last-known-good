package tui

import "github.com/charmbracelet/lipgloss"

const (
	ColorBackground   = "#00172e"
	ColorPrimaryText  = "#f6dcac"
	ColorAccent       = "#faa968"
	ColorTeal         = "#3f8f8a"
	ColorCyan         = "#8cbfb8"
	ColorMuted        = "#a7c9c6"
	ColorGreen        = "#028391"
	ColorYellow       = "#e97b3c"
	ColorRed          = "#f85525"
	ColorStatusBg     = "#134e5a"
)

var (
	HeaderStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(ColorBackground)).
			Foreground(lipgloss.Color(ColorAccent)).
			Bold(true).
			Padding(0, 1)

	HeaderBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(ColorBackground)).
			Foreground(lipgloss.Color(ColorTeal))

	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(ColorStatusBg)).
			Foreground(lipgloss.Color(ColorMuted))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorRed)).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGreen))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorYellow))

	UserBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorAccent)).
			Padding(0, 1)

	UserLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorAccent)).
			Bold(true)

	AssistantBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(ColorTeal)).
				Padding(0, 1)

	AssistantLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorTeal)).
				Bold(true)

	ToolBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorTeal)).
			Padding(0, 1)

	ToolHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTeal)).
			Italic(true)

	ToolResultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorCyan))

	ThinkingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted)).
			Italic(true)

	InputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorAccent))

	InputTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimaryText))

	ViewportStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(ColorBackground))

	DividerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTeal))

	KeyShortcutStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTeal))

	KeyDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))

	KeySepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTeal))

	KeyEllipsisStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorCyan))

	WelcomeContainer = lipgloss.NewStyle().
				Background(lipgloss.Color(ColorBackground)).
				Align(lipgloss.Center)

	WelcomeTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorAccent)).
				Bold(true)

	WelcomeSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorMuted))

	WelcomeInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorPrimaryText)).
				Width(60).
				Align(lipgloss.Center)
)

var LKGAsciiArt = []string{
	`██     ██   ██████`,
	`██    ███  ██   ██`,
	`██  ██ ██  ██████`,
	`██ ██  ██  ██`,
	`████   ██  ██`,
}
