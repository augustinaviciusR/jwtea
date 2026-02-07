package tabs

import (
	"fmt"
	"jwtea/internal/tui/theme"
	"strings"
	"time"

	"jwtea/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SettingsTab struct {
	ctx *tui.Context

	width  int
	height int

	chaosExpired   bool
	chaosSignature bool
	chaos500       bool

	editMode       bool
	focusedSetting int

	accessTokenExpiry  string
	idTokenExpiry      string
	refreshTokenExpiry string
	supportedScopes    string

	errorMsg string

	styleHeader   lipgloss.Style
	styleKey      lipgloss.Style
	styleVal      lipgloss.Style
	styleCursor   lipgloss.Style
	styleChaosOn  lipgloss.Style
	styleChaosOff lipgloss.Style
	styleError    lipgloss.Style
	styleBorder   lipgloss.Style
	styleModal    lipgloss.Style
}

func NewSettingsTab(ctx *tui.Context) *SettingsTab {
	return &SettingsTab{
		ctx:           ctx,
		styleHeader:   theme.Header,
		styleKey:      theme.Muted,
		styleVal:      theme.Text,
		styleCursor:   theme.Accent,
		styleChaosOn:  theme.Error,
		styleChaosOff: theme.Muted,
		styleError:    theme.Error,
		styleBorder:   theme.Border,
		styleModal:    theme.Modal,
	}
}

func (t *SettingsTab) Init() tea.Cmd {
	return nil
}

func (t *SettingsTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = v.Width
		t.height = v.Height
		return t, nil
	case tea.KeyMsg:
		if t.editMode {
			return t.handleEditKeys(v)
		}

		switch v.String() {
		case "e":
			t.enterEditMode()
		case "x":
			if t.ctx.Chaos != nil {
				t.chaosExpired = t.ctx.Chaos.ToggleNextTokenExpired()
			}
		case "s":
			if t.ctx.Chaos != nil {
				t.chaosSignature = t.ctx.Chaos.ToggleInvalidSignature()
			}
		case "5":
			if t.ctx.Chaos != nil {
				t.chaos500 = t.ctx.Chaos.ToggleSimulate500()
			}
		}
	}
	return t, nil
}

func (t *SettingsTab) View() string {
	if t.editMode {
		return t.viewEditMode()
	}

	var b strings.Builder

	b.WriteString(t.styleHeader.Render("Server Information"))
	b.WriteString("\n\n")

	kv := func(k, v string) string {
		return fmt.Sprintf("%s %s", t.styleKey.Render(fmt.Sprintf("%-16s", k+":")), t.styleVal.Render(v))
	}

	if t.ctx.ServerRunning {
		b.WriteString(kv("Status", "Running"))
		b.WriteString("\n")
		b.WriteString(kv("Issuer", t.ctx.Issuer))
		b.WriteString("\n")
		b.WriteString(kv("Key ID", t.ctx.Kid))
		b.WriteString("\n")
	} else {
		b.WriteString(t.styleVal.Render("  Not running (standalone mode)"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(t.styleHeader.Render("Token Configuration"))
	b.WriteString("\n\n")

	if t.ctx.Config != nil {
		b.WriteString(kv("Access Token", t.ctx.Config.Tokens.AccessTokenExpiry.String()))
		b.WriteString("\n")
		b.WriteString(kv("ID Token", t.ctx.Config.Tokens.IDTokenExpiry.String()))
		b.WriteString("\n")
		b.WriteString(kv("Refresh Token", t.ctx.Config.Tokens.RefreshTokenExpiry.String()))
		b.WriteString("\n")
		b.WriteString(kv("Algorithm", t.ctx.Config.Tokens.Algorithm))
		b.WriteString("\n\n")

		b.WriteString(t.styleKey.Render("Supported Scopes:"))
		b.WriteString("\n")
		for _, scope := range t.ctx.Config.OAuth.SupportedScopes {
			b.WriteString("  - " + t.styleVal.Render(scope))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(lipgloss.NewStyle().Faint(true).Render("  Configuration not available"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(t.styleHeader.Render("Chaos Mode Controls"))
	b.WriteString("\n\n")

	if t.ctx.Chaos == nil {
		b.WriteString(lipgloss.NewStyle().Faint(true).Render("  Chaos controls unavailable in standalone mode"))
		b.WriteString("\n")
	} else {
		t.chaosExpired = t.ctx.Chaos.NextTokenExpired
		t.chaosSignature = t.ctx.Chaos.IsInvalidSignature()
		t.chaos500 = t.ctx.Chaos.IsSimulate500()

		chaosItem := func(key, label string, active bool) string {
			status := "OFF"
			style := t.styleChaosOff
			if active {
				status = "ON "
				style = t.styleChaosOn
			}
			return fmt.Sprintf("  [%s] %s  %s", key, style.Render(status), label)
		}

		b.WriteString(chaosItem("x", "Expire Next Token", t.chaosExpired))
		b.WriteString("\n")
		b.WriteString(chaosItem("s", "Invalid Signature", t.chaosSignature))
		b.WriteString("\n")
		b.WriteString(chaosItem("5", "Simulate 500 Errors", t.chaos500))
		b.WriteString("\n")

		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Faint(true).Render("  Press the key to toggle each chaos mode"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	footer := lipgloss.NewStyle().Faint(true).Render("e edit config • x expire token • s invalid sig • 5 500 errors")
	b.WriteString(footer)

	return b.String()
}

func (t *SettingsTab) Help() []string {
	return []string{
		"Settings Tab:",
		"  e    edit token configuration (changes auto-saved)",
		"  x    toggle expired token chaos (one-time)",
		"  s    toggle invalid signature chaos",
		"  5    toggle 500 error chaos",
		"",
		"Edit Mode:",
		"  tab         next field",
		"  shift+tab   previous field",
		"  enter       save changes",
		"  esc         cancel",
	}
}

func (t *SettingsTab) enterEditMode() {
	if t.ctx.Config == nil {
		return
	}

	t.editMode = true
	t.focusedSetting = 0
	t.errorMsg = ""

	t.accessTokenExpiry = t.ctx.Config.Tokens.AccessTokenExpiry.String()
	t.idTokenExpiry = t.ctx.Config.Tokens.IDTokenExpiry.String()
	t.refreshTokenExpiry = t.ctx.Config.Tokens.RefreshTokenExpiry.String()
	t.supportedScopes = strings.Join(t.ctx.Config.OAuth.SupportedScopes, ",")
}

func (t *SettingsTab) handleEditKeys(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		t.editMode = false
		t.errorMsg = ""
		return t, nil
	case "tab":
		t.focusedSetting = (t.focusedSetting + 1) % 4
	case "shift+tab":
		t.focusedSetting--
		if t.focusedSetting < 0 {
			t.focusedSetting = 3
		}
	case "enter":
		return t, t.saveSettings()
	case "backspace":
		t.deleteCharFromField()
	default:
		if len(key.String()) == 1 {
			t.addCharToField(key.String())
		}
	}
	return t, nil
}

func (t *SettingsTab) addCharToField(char string) {
	switch t.focusedSetting {
	case 0:
		t.accessTokenExpiry += char
	case 1:
		t.idTokenExpiry += char
	case 2:
		t.refreshTokenExpiry += char
	case 3:
		t.supportedScopes += char
	}
}

func (t *SettingsTab) deleteCharFromField() {
	switch t.focusedSetting {
	case 0:
		if len(t.accessTokenExpiry) > 0 {
			t.accessTokenExpiry = t.accessTokenExpiry[:len(t.accessTokenExpiry)-1]
		}
	case 1:
		if len(t.idTokenExpiry) > 0 {
			t.idTokenExpiry = t.idTokenExpiry[:len(t.idTokenExpiry)-1]
		}
	case 2:
		if len(t.refreshTokenExpiry) > 0 {
			t.refreshTokenExpiry = t.refreshTokenExpiry[:len(t.refreshTokenExpiry)-1]
		}
	case 3:
		if len(t.supportedScopes) > 0 {
			t.supportedScopes = t.supportedScopes[:len(t.supportedScopes)-1]
		}
	}
}

func (t *SettingsTab) saveSettings() tea.Cmd {
	if t.ctx.Config == nil {
		t.editMode = false
		return nil
	}

	accessExpiry, err := time.ParseDuration(t.accessTokenExpiry)
	if err != nil {
		t.errorMsg = "Invalid access token expiry format (e.g., 5m, 1h)"
		return nil
	}

	idExpiry, err := time.ParseDuration(t.idTokenExpiry)
	if err != nil {
		t.errorMsg = "Invalid ID token expiry format (e.g., 5m, 1h)"
		return nil
	}

	refreshExpiry, err := time.ParseDuration(t.refreshTokenExpiry)
	if err != nil {
		t.errorMsg = "Invalid refresh token expiry format (e.g., 24h, 7d)"
		return nil
	}

	var scopes []string
	if t.supportedScopes != "" {
		for _, scope := range strings.Split(t.supportedScopes, ",") {
			trimmed := strings.TrimSpace(scope)
			if trimmed != "" {
				scopes = append(scopes, trimmed)
			}
		}
	}

	if len(scopes) == 0 {
		t.errorMsg = "At least one scope is required"
		return nil
	}

	t.ctx.Config.Tokens.AccessTokenExpiry.Duration = accessExpiry
	t.ctx.Config.Tokens.IDTokenExpiry.Duration = idExpiry
	t.ctx.Config.Tokens.RefreshTokenExpiry.Duration = refreshExpiry
	t.ctx.Config.OAuth.SupportedScopes = scopes

	if err := t.ctx.AutoSave(); err != nil {
		t.errorMsg = fmt.Sprintf("Failed to save: %v", err)
		return nil
	}

	t.editMode = false
	t.errorMsg = ""
	return nil
}

func (t *SettingsTab) IsTextInputActive() bool {
	return t.editMode
}

func (t *SettingsTab) viewEditMode() string {
	var b strings.Builder

	b.WriteString(t.styleHeader.Render("Edit Token Configuration"))
	b.WriteString("\n\n")

	accessLabel := "Access Token Expiry:"
	accessStyle := t.styleVal
	if t.focusedSetting == 0 {
		accessLabel = "Access Token Expiry: ▶"
		accessStyle = t.styleCursor
	}
	accessValue := t.accessTokenExpiry
	if accessValue == "" {
		accessValue = "_"
	}
	b.WriteString(accessLabel + " " + accessStyle.Render(accessValue))
	b.WriteString("\n")

	idLabel := "ID Token Expiry:"
	idStyle := t.styleVal
	if t.focusedSetting == 1 {
		idLabel = "ID Token Expiry:     ▶"
		idStyle = t.styleCursor
	}
	idValue := t.idTokenExpiry
	if idValue == "" {
		idValue = "_"
	}
	b.WriteString(idLabel + "     " + idStyle.Render(idValue))
	b.WriteString("\n")

	refreshLabel := "Refresh Token Expiry:"
	refreshStyle := t.styleVal
	if t.focusedSetting == 2 {
		refreshLabel = "Refresh Token Expiry:▶"
		refreshStyle = t.styleCursor
	}
	refreshValue := t.refreshTokenExpiry
	if refreshValue == "" {
		refreshValue = "_"
	}
	b.WriteString(refreshLabel + " " + refreshStyle.Render(refreshValue))
	b.WriteString("\n")

	scopesLabel := "Supported Scopes:"
	scopesStyle := t.styleVal
	if t.focusedSetting == 3 {
		scopesLabel = "Supported Scopes:    ▶"
		scopesStyle = t.styleCursor
	}
	scopesValue := t.supportedScopes
	if scopesValue == "" {
		scopesValue = "_"
	}
	b.WriteString(scopesLabel + "    " + scopesStyle.Render(scopesValue))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("                     (comma-separated, e.g., openid,profile,email)"))
	b.WriteString("\n\n")

	if t.errorMsg != "" {
		b.WriteString(t.styleError.Render(t.errorMsg))
		b.WriteString("\n\n")
	}

	b.WriteString(lipgloss.NewStyle().Faint(true).Render("Examples: 5m, 1h, 24h, 7d"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("tab/shift+tab navigate • enter save • esc cancel"))

	return t.styleModal.Render(b.String())
}
