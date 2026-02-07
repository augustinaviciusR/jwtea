package components

import (
	"fmt"
	"strings"

	"jwtea/internal/tui/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type RadioOption struct {
	Label string
	Value string
}

type Radio struct {
	Label    string
	Options  []RadioOption
	Selected int
	Focused  bool

	styleFocused   lipgloss.Style
	styleUnfocused lipgloss.Style
	styleSelected  lipgloss.Style
}

func NewRadio(label string, options []RadioOption, defaultIndex int) *Radio {
	if defaultIndex < 0 || defaultIndex >= len(options) {
		defaultIndex = 0
	}
	return &Radio{
		Label:          label,
		Options:        options,
		Selected:       defaultIndex,
		Focused:        false,
		styleFocused:   theme.Focused,
		styleUnfocused: theme.Unfocused,
		styleSelected:  lipgloss.NewStyle().Foreground(theme.ColorAccent),
	}
}

func (r *Radio) Init() tea.Cmd {
	return nil
}

func (r *Radio) Update(msg tea.Msg) (*Radio, tea.Cmd) {
	if !r.Focused {
		return r, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k":
			if r.Selected > 0 {
				r.Selected--
			}
		case "down", "j":
			if r.Selected < len(r.Options)-1 {
				r.Selected++
			}
		case " ", "enter":
			return r, nil
		}
	}
	return r, nil
}

func (r *Radio) View() string {
	var b strings.Builder

	if r.Focused {
		b.WriteString(r.styleFocused.Render(r.Label))
	} else {
		b.WriteString(r.Label)
	}
	b.WriteString("\n")

	for i, opt := range r.Options {
		indicator := "○"
		if i == r.Selected {
			indicator = "●"
		}

		line := fmt.Sprintf("  %s %s", indicator, opt.Label)

		if r.Focused && i == r.Selected {
			b.WriteString(r.styleFocused.Bold(true).Render(line))
		} else if i == r.Selected {
			b.WriteString(r.styleSelected.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (r *Radio) GetValue() string {
	if r.Selected >= 0 && r.Selected < len(r.Options) {
		return r.Options[r.Selected].Value
	}
	return ""
}

func (r *Radio) SetFocused(focused bool) {
	r.Focused = focused
}
