package components

import (
	"encoding/json"

	"jwtea/internal/tui/theme"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TextArea struct {
	Label        string
	textarea     textarea.Model
	Focused      bool
	ValidateJSON bool
	err          error

	styleFocused   lipgloss.Style
	styleUnfocused lipgloss.Style
	styleError     lipgloss.Style
}

func NewTextArea(label, placeholder string, validateJSON bool) *TextArea {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.SetHeight(3)
	ta.SetWidth(60)
	ta.ShowLineNumbers = false

	return &TextArea{
		Label:          label,
		textarea:       ta,
		Focused:        false,
		ValidateJSON:   validateJSON,
		styleFocused:   theme.Focused,
		styleUnfocused: theme.Unfocused,
		styleError:     theme.Error,
	}
}

func (t *TextArea) Init() tea.Cmd {
	return nil
}

func (t *TextArea) Update(msg tea.Msg) (*TextArea, tea.Cmd) {
	if !t.Focused {
		return t, nil
	}

	var cmd tea.Cmd
	t.textarea, cmd = t.textarea.Update(msg)

	if t.ValidateJSON {
		val := t.textarea.Value()
		if val != "" {
			var js map[string]any
			if err := json.Unmarshal([]byte(val), &js); err != nil {
				t.err = err
			} else {
				t.err = nil
			}
		} else {
			t.err = nil
		}
	}

	return t, cmd
}

func (t *TextArea) View() string {
	labelStyle := t.styleUnfocused
	if t.Focused {
		labelStyle = t.styleFocused
		t.textarea.Focus()
	} else {
		t.textarea.Blur()
	}

	view := labelStyle.Render(t.Label) + "\n"
	view += t.textarea.View()

	if t.err != nil {
		view += "\n" + t.styleError.Render("  Invalid JSON: "+t.err.Error())
	}

	if t.Focused {
		hint := lipgloss.NewStyle().Faint(true)
		view += "\n" + hint.Render("  [esc] done editing  [tab] next field")
	}

	return view
}

func (t *TextArea) GetValue() string {
	return t.textarea.Value()
}

func (t *TextArea) SetValue(val string) {
	t.textarea.SetValue(val)
}

func (t *TextArea) SetFocused(focused bool) {
	t.Focused = focused
	if focused {
		t.textarea.Focus()
	} else {
		t.textarea.Blur()
	}
}

func (t *TextArea) HasError() bool {
	return t.err != nil
}
