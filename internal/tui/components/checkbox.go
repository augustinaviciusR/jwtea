package components

import (
	"fmt"
	"strings"

	"jwtea/internal/tui/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CheckboxOption struct {
	Label   string
	Value   string
	Checked bool
}

type Checkbox struct {
	Label   string
	Options []CheckboxOption
	Cursor  int
	Focused bool

	styleFocused   lipgloss.Style
	styleUnfocused lipgloss.Style
	styleChecked   lipgloss.Style
}

func NewCheckbox(label string, options []CheckboxOption) *Checkbox {
	return &Checkbox{
		Label:          label,
		Options:        options,
		Cursor:         0,
		Focused:        false,
		styleFocused:   theme.Focused,
		styleUnfocused: theme.Unfocused,
		styleChecked:   lipgloss.NewStyle().Foreground(theme.ColorAccent),
	}
}

func (c *Checkbox) Init() tea.Cmd {
	return nil
}

func (c *Checkbox) Update(msg tea.Msg) (*Checkbox, tea.Cmd) {
	if !c.Focused {
		return c, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k":
			if c.Cursor > 0 {
				c.Cursor--
			}
		case "down", "j":
			if c.Cursor < len(c.Options)-1 {
				c.Cursor++
			}
		case " ":
			if c.Cursor >= 0 && c.Cursor < len(c.Options) {
				c.Options[c.Cursor].Checked = !c.Options[c.Cursor].Checked
			}
		}
	}
	return c, nil
}

func (c *Checkbox) View() string {
	var b strings.Builder

	if c.Focused {
		b.WriteString(c.styleFocused.Render(c.Label))
	} else {
		b.WriteString(c.Label)
	}
	b.WriteString("\n")

	for i, opt := range c.Options {
		indicator := "☐"
		if opt.Checked {
			indicator = "☑"
		}

		line := fmt.Sprintf("  %s %s", indicator, opt.Label)

		if c.Focused && i == c.Cursor {
			b.WriteString(c.styleFocused.Copy().Bold(true).Render(line))
		} else if opt.Checked {
			b.WriteString(c.styleChecked.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (c *Checkbox) GetValues() []string {
	var values []string
	for _, opt := range c.Options {
		if opt.Checked {
			values = append(values, opt.Value)
		}
	}
	return values
}

func (c *Checkbox) SetFocused(focused bool) {
	c.Focused = focused
}
