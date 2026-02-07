package theme

import "github.com/charmbracelet/lipgloss"

// Color palette
const (
	ColorPrimary   = lipgloss.Color("205") // Pink/magenta - headers, focused elements
	ColorMuted     = lipgloss.Color("240") // Gray - borders, unfocused elements
	ColorAccent    = lipgloss.Color("42")  // Green - cursor, selected items
	ColorText      = lipgloss.Color("255") // White - normal text
	ColorError     = lipgloss.Color("196") // Red - errors
	ColorMethod    = lipgloss.Color("51")  // Cyan - HTTP methods
	ColorStatus2xx = lipgloss.Color("42")  // Green - success responses
	ColorStatus3xx = lipgloss.Color("69")  // Blue - redirects
	ColorStatus4xx = lipgloss.Color("219") // Pink - client errors
	ColorStatus5xx = lipgloss.Color("197") // Red - server errors
	ColorDetailKey = lipgloss.Color("245") // Light gray - detail labels
	ColorBorder    = lipgloss.Color("63")  // Purple - detail borders
)

// Common styles
var (
	Header = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	Text = lipgloss.NewStyle().
		Foreground(ColorText)

	Muted = lipgloss.NewStyle().
		Foreground(ColorMuted)

	Accent = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	Error = lipgloss.NewStyle().
		Foreground(ColorError)

	Focused = lipgloss.NewStyle().
		Foreground(ColorPrimary)

	Unfocused = lipgloss.NewStyle().
			Foreground(ColorMuted)

	Button = lipgloss.NewStyle().
		Foreground(ColorMuted)

	ButtonActive = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	ButtonBordered = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 2)

	ButtonBorderedActive = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				Border(lipgloss.RoundedBorder()).
				Padding(0, 2)

	Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Padding(1, 2)

	Modal = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2)

	DetailBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)
)

// HTTP status styles
var (
	StatusMethod = lipgloss.NewStyle().
			Foreground(ColorMethod).
			Bold(true)

	Status2xx = lipgloss.NewStyle().
			Foreground(ColorStatus2xx)

	Status3xx = lipgloss.NewStyle().
			Foreground(ColorStatus3xx)

	Status4xx = lipgloss.NewStyle().
			Foreground(ColorStatus4xx)

	Status5xx = lipgloss.NewStyle().
			Foreground(ColorStatus5xx).
			Bold(true)
)

// StatusStyle returns the appropriate style for an HTTP status code
func StatusStyle(code int) lipgloss.Style {
	switch {
	case code >= 500:
		return Status5xx
	case code >= 400:
		return Status4xx
	case code >= 300:
		return Status3xx
	default:
		return Status2xx
	}
}
