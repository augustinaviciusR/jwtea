package tabs

import (
	"encoding/json"
	"jwtea/internal/core"
	"sort"
	"strings"
	"time"

	"jwtea/internal/tui"
	"jwtea/internal/tui/components"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GenerateTab struct {
	ctx *tui.Context

	userRadio   *components.Radio
	scopeCheck  *components.Checkbox
	expiryRadio *components.Radio
	claimsArea  *components.TextArea
	tokenView   *components.TokenView

	focusIndex int
	focusables []focusable

	buttonIndex    int
	buttonLabels   []string
	generatedToken string

	scrollOffset int
	viewHeight   int
	width        int

	lastKey          string
	waitingForLeader bool

	styleHeader   lipgloss.Style
	styleButton   lipgloss.Style
	styleButtonOn lipgloss.Style
	styleError    lipgloss.Style
}

type focusable interface {
	SetFocused(bool)
}

func NewGenerateTab(ctx *tui.Context) *GenerateTab {
	userOptions := []components.RadioOption{}
	if ctx.Store != nil {
		users := ctx.Store.ListUsers()
		sort.Slice(users, func(i, j int) bool {
			return users[i].Email < users[j].Email
		})
		for _, user := range users {
			userOptions = append(userOptions, components.RadioOption{
				Label: user.Email,
				Value: user.Email,
			})
		}
	}
	if len(userOptions) == 0 {
		userOptions = []components.RadioOption{
			{Label: "alice@test.com", Value: "alice@test.com"},
			{Label: "bob@test.com", Value: "bob@test.com"},
			{Label: "admin@test.com", Value: "admin@test.com"},
		}
	}
	userRadio := components.NewRadio("Select User:", userOptions, 0)

	scopeOptions := []components.CheckboxOption{}
	if ctx.Config != nil {
		supportedScopes := ctx.Config.OAuth.SupportedScopes
		defaultScopes := make(map[string]bool)
		for _, s := range ctx.Config.OAuth.DefaultScopes {
			defaultScopes[s] = true
		}
		for _, scope := range supportedScopes {
			scopeOptions = append(scopeOptions, components.CheckboxOption{
				Label:   scope,
				Value:   scope,
				Checked: defaultScopes[scope],
			})
		}
	}
	if len(scopeOptions) == 0 {
		scopeOptions = []components.CheckboxOption{
			{Label: "openid", Value: "openid", Checked: true},
			{Label: "profile", Value: "profile", Checked: true},
			{Label: "email", Value: "email", Checked: true},
			{Label: "offline_access", Value: "offline_access", Checked: false},
		}
	}
	scopeCheck := components.NewCheckbox("Scopes:", scopeOptions)

	expiryOptions := []components.RadioOption{
		{Label: "5 minutes", Value: "5m"},
		{Label: "1 hour", Value: "1h"},
		{Label: "24 hours", Value: "24h"},
		{Label: "7 days", Value: "168h"},
	}
	defaultIndex := 1
	if ctx.Config != nil {
		configExpiry := ctx.Config.Tokens.AccessTokenExpiry.Duration
		for i, opt := range expiryOptions {
			dur, _ := time.ParseDuration(opt.Value)
			if dur == configExpiry {
				defaultIndex = i
				break
			}
		}
	}
	expiryRadio := components.NewRadio("Expiration:", expiryOptions, defaultIndex)

	claimsArea := components.NewTextArea("Custom Claims (JSON):", `{"role":"user"}`, true)
	claimsArea.SetValue(`{"role":"user"}`)

	tokenView := components.NewTokenView("")

	tab := &GenerateTab{
		ctx:           ctx,
		userRadio:     userRadio,
		scopeCheck:    scopeCheck,
		expiryRadio:   expiryRadio,
		claimsArea:    claimsArea,
		tokenView:     tokenView,
		focusIndex:    0,
		buttonIndex:   0,
		buttonLabels:  []string{"[C] Generate"},
		styleHeader:   lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		styleButton:   lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Border(lipgloss.RoundedBorder()).Padding(0, 2),
		styleButtonOn: lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Border(lipgloss.RoundedBorder()).Padding(0, 2),
		styleError:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
	}

	tab.focusables = []focusable{
		tab.userRadio,
		tab.scopeCheck,
		tab.expiryRadio,
		tab.claimsArea,
	}

	tab.updateFocus()

	return tab
}

func (t *GenerateTab) Init() tea.Cmd {
	return nil
}

func (t *GenerateTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		t.viewHeight = v.Height - 8
		if t.viewHeight < 10 {
			t.viewHeight = 10
		}
		t.width = v.Width
		t.tokenView.SetWidth(v.Width - 10)
		return t, nil
	case tea.MouseMsg:
		return t.handleMouse(v)
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		keyStr := key.String()

		// When claims textarea is focused, forward all keys to it
		// except esc (to leave) and tab/shift+tab (to navigate)
		if t.focusIndex == 3 {
			switch keyStr {
			case "esc":
				t.focusIndex++
				t.updateFocus()
				t.scrollToFocusedField()
				return t, nil
			case "tab":
				maxFocus := len(t.focusables) + 1
				if t.generatedToken != "" {
					maxFocus = len(t.focusables) + 2
				}
				t.focusIndex = (t.focusIndex + 1) % maxFocus
				t.updateFocus()
				t.scrollToFocusedField()
				return t, nil
			case "shift+tab":
				t.focusIndex--
				if t.focusIndex < 0 {
					maxFocus := len(t.focusables)
					if t.generatedToken != "" {
						maxFocus = len(t.focusables) + 1
					}
					t.focusIndex = maxFocus
				}
				t.updateFocus()
				t.scrollToFocusedField()
				return t, nil
			default:
				updated, cmd := t.claimsArea.Update(msg)
				t.claimsArea = updated
				return t, cmd
			}
		}

		if t.waitingForLeader {
			t.waitingForLeader = false
			switch keyStr {
			case "c":
				t.buttonIndex = 0
				return t, t.handleButtonPress()
			case "i":
				t.buttonIndex = 1
				return t, t.handleButtonPress()
			default:
				return t, nil
			}
		}

		if keyStr == "g" && t.lastKey == "g" {
			t.focusIndex = 0
			t.updateFocus()
			t.scrollOffset = 0
			t.lastKey = ""
			return t, nil
		}
		if keyStr == "g" {
			t.lastKey = "g"
			return t, nil
		}
		t.lastKey = ""

		switch keyStr {
		case "tab":
			maxFocus := len(t.focusables) + 1
			if t.generatedToken != "" {
				maxFocus = len(t.focusables) + 2
			}
			t.focusIndex = (t.focusIndex + 1) % maxFocus
			t.updateFocus()
			t.scrollToFocusedField()
			return t, nil
		case "shift+tab":
			t.focusIndex--
			if t.focusIndex < 0 {
				maxFocus := len(t.focusables)
				if t.generatedToken != "" {
					maxFocus = len(t.focusables) + 1
				}
				t.focusIndex = maxFocus
			}
			t.updateFocus()
			t.scrollToFocusedField()
			return t, nil

		case "j", "down":
			if t.focusIndex < len(t.focusables) {
				break
			}
			return t, t.handleVimDown()
		case "k", "up":
			if t.focusIndex < len(t.focusables) {
				break
			}
			return t, t.handleVimUp()
		case "h", "left":
			return t, t.handleVimLeft()
		case "l", "right":
			return t, t.handleVimRight()

		case "G":
			t.focusIndex = len(t.focusables)
			t.updateFocus()
			t.scrollToFocusedField()
			return t, nil

		case "ctrl+d":
			if t.viewHeight > 0 {
				t.scrollOffset += t.viewHeight / 2
			}
			return t, nil
		case "ctrl+u":
			if t.viewHeight > 0 {
				t.scrollOffset -= t.viewHeight / 2
			}
			if t.scrollOffset < 0 {
				t.scrollOffset = 0
			}
			return t, nil
		case "ctrl+f", "pgdown":
			if t.viewHeight > 0 {
				t.scrollOffset += t.viewHeight
			}
			return t, nil
		case "ctrl+b", "pgup":
			if t.viewHeight > 0 {
				t.scrollOffset -= t.viewHeight
			}
			if t.scrollOffset < 0 {
				t.scrollOffset = 0
			}
			return t, nil
		case "home":
			t.scrollOffset = 0
			return t, nil
		case "end":
			t.scrollOffset = 999999
			return t, nil

		case ",":
			t.waitingForLeader = true
			return t, nil

		case "c":
			if t.focusIndex == len(t.focusables)+1 && t.generatedToken != "" {
				return t, nil
			}
			if t.focusIndex != 3 {
				t.buttonIndex = 0
				return t, t.handleButtonPress()
			}
		case "i":
			if t.focusIndex != 3 {
				t.buttonIndex = 1
				return t, t.handleButtonPress()
			}

		case "d":
			if t.generatedToken != "" {
				t.tokenView.ToggleDecoded()
			}
			return t, nil

		case "enter":
			if t.focusIndex == len(t.focusables) {
				return t, t.handleButtonPress()
			}
			return t, nil

		case " ":
			if t.focusIndex == len(t.focusables) {
				return t, t.handleButtonPress()
			}
		}
	}

	if t.focusIndex < len(t.focusables) {
		switch t.focusIndex {
		case 0:
			updated, cmd := t.userRadio.Update(msg)
			t.userRadio = updated
			cmds = append(cmds, cmd)
		case 1:
			updated, cmd := t.scopeCheck.Update(msg)
			t.scopeCheck = updated
			cmds = append(cmds, cmd)
		case 2:
			updated, cmd := t.expiryRadio.Update(msg)
			t.expiryRadio = updated
			cmds = append(cmds, cmd)
		case 3:
			updated, cmd := t.claimsArea.Update(msg)
			t.claimsArea = updated
			cmds = append(cmds, cmd)
		}
	}

	return t, tea.Batch(cmds...)
}

func (t *GenerateTab) View() string {
	var b strings.Builder

	if t.ctx.PrivKey == nil {
		b.WriteString(t.styleError.Render("⚠ Token generation requires server mode"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Faint(true).Render("Start with: jwtea serve --dashboard"))
		b.WriteString("\n\n")
	}

	b.WriteString(t.userRadio.View())
	b.WriteString("\n")
	b.WriteString(t.scopeCheck.View())
	b.WriteString("\n")
	b.WriteString(t.expiryRadio.View())
	b.WriteString("\n")
	b.WriteString(t.claimsArea.View())
	b.WriteString("\n\n")

	var buttons []string
	for i, label := range t.buttonLabels {
		style := t.styleButton
		if t.focusIndex == len(t.focusables) && i == t.buttonIndex {
			style = t.styleButtonOn
		}
		buttons = append(buttons, style.Render(label))
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, buttons...))
	b.WriteString("\n\n")

	if t.generatedToken != "" {
		b.WriteString(t.tokenView.View())
	}

	footerText := "j/k fields • gg/G jump • ,c/,i generate • space/enter select"
	if t.waitingForLeader {
		footerText = "Leader: , (waiting for command: c=copy, i=invalid)"
	}
	footer := lipgloss.NewStyle().Faint(true).Render(footerText)

	content := b.String()
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	viewHeight := t.viewHeight
	if viewHeight <= 0 {
		viewHeight = 20
	}

	availableHeight := viewHeight - 2

	if availableHeight > 0 && totalLines > availableHeight {
		start := t.scrollOffset
		end := t.scrollOffset + availableHeight

		if start < 0 {
			start = 0
		}
		if end > totalLines {
			end = totalLines
			start = end - availableHeight
			if start < 0 {
				start = 0
			}
		}

		lines = lines[start:end]

		scrollInfo := ""
		if start > 0 {
			scrollInfo += "↑ "
		}
		if end < totalLines {
			scrollInfo += "↓"
		}
		if scrollInfo != "" {
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().Faint(true).Render(scrollInfo))
		}
	}

	return strings.Join(lines, "\n") + "\n\n" + footer
}

func (t *GenerateTab) Help() []string {
	return []string{
		"Generate Tab (Vim-Style Navigation):",
		"",
		"Field Navigation:",
		"  j / ↓           next field (or scroll down if at bottom)",
		"  k / ↑           previous field (or scroll up if at top)",
		"  h / ←           previous option/button",
		"  l / →           next option/button",
		"  gg              jump to first field + scroll to top",
		"  G               jump to last field (buttons)",
		"  tab/shift+tab   cycle through fields, buttons, and token view",
		"",
		"Selection:",
		"  space           toggle checkbox / select radio",
		"  enter           activate button",
		"",
		"Scrolling:",
		"  ctrl+d          scroll half page down",
		"  ctrl+u          scroll half page up",
		"  ctrl+f / PgDn   scroll full page down",
		"  ctrl+b / PgUp   scroll full page up",
		"  Home / End      scroll to top/bottom",
		"",
		"Token Actions:",
		"  ,c              generate token + copy to clipboard",
		"  ,i              generate invalid token (chaos mode)",
		"  d               toggle decoded token view",
		"  c (on token)    copy token to clipboard",
	}
}

func (t *GenerateTab) IsTextInputActive() bool {
	return t.focusIndex == 3
}

func (t *GenerateTab) updateFocus() {
	for _, f := range t.focusables {
		f.SetFocused(false)
	}
	t.tokenView.SetFocused(false)

	if t.focusIndex < len(t.focusables) {
		t.focusables[t.focusIndex].SetFocused(true)
	} else if t.focusIndex == len(t.focusables)+1 {
		t.tokenView.SetFocused(true)
	}
}

func (t *GenerateTab) handleVimDown() tea.Cmd {
	if t.focusIndex >= len(t.focusables) {
		t.scrollOffset++
		return nil
	}
	t.focusIndex++
	if t.focusIndex > len(t.focusables) {
		t.focusIndex = len(t.focusables)
	}
	t.updateFocus()
	t.scrollToFocusedField()
	return nil
}

func (t *GenerateTab) handleVimUp() tea.Cmd {
	if t.focusIndex == 0 {
		t.scrollOffset--
		if t.scrollOffset < 0 {
			t.scrollOffset = 0
		}
		return nil
	}
	t.focusIndex--
	if t.focusIndex < 0 {
		t.focusIndex = 0
	}
	t.updateFocus()
	t.scrollToFocusedField()
	return nil
}

func (t *GenerateTab) handleVimLeft() tea.Cmd {
	if t.focusIndex == len(t.focusables) {
		t.buttonIndex--
		if t.buttonIndex < 0 {
			t.buttonIndex = len(t.buttonLabels) - 1
		}
		return nil
	}
	return nil
}

func (t *GenerateTab) handleVimRight() tea.Cmd {
	if t.focusIndex == len(t.focusables) {
		t.buttonIndex = (t.buttonIndex + 1) % len(t.buttonLabels)
		return nil
	}
	return nil
}

func (t *GenerateTab) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return t, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		t.scrollOffset -= 3
		if t.scrollOffset < 0 {
			t.scrollOffset = 0
		}
	case tea.MouseButtonWheelDown:
		t.scrollOffset += 3
	}
	return t, nil
}

func (t *GenerateTab) scrollToFocusedField() {
	var fieldLine int
	switch t.focusIndex {
	case 0:
		fieldLine = 0
	case 1:
		fieldLine = 5
	case 2:
		fieldLine = 12
	case 3:
		fieldLine = 18
	case 4:
		fieldLine = 24
	case 5:
		fieldLine = 28
	default:
		return
	}

	viewHeight := t.viewHeight
	if viewHeight <= 0 {
		viewHeight = 20
	}
	availableHeight := viewHeight - 2

	if fieldLine < t.scrollOffset {
		t.scrollOffset = fieldLine
	}
	if fieldLine > t.scrollOffset+availableHeight-5 {
		t.scrollOffset = fieldLine - availableHeight + 5
		if t.scrollOffset < 0 {
			t.scrollOffset = 0
		}
	}
}

func (t *GenerateTab) handleButtonPress() tea.Cmd {
	if t.ctx.PrivKey == nil {
		return nil
	}

	user := t.userRadio.GetValue()
	scopes := strings.Join(t.scopeCheck.GetValues(), " ")
	expiryStr := t.expiryRadio.GetValue()
	expiry, _ := time.ParseDuration(expiryStr)

	customClaims := make(map[string]any)
	claimsJSON := t.claimsArea.GetValue()
	if claimsJSON != "" {
		if err := json.Unmarshal([]byte(claimsJSON), &customClaims); err != nil {
			return nil
		}
	}

	clientID := "demo-client"
	if t.ctx.Store != nil {
		clients := t.ctx.Store.ListClients()
		if len(clients) > 0 {
			clientID = clients[0].ID
		}
	}

	req := core.TokenRequest{
		Subject:      user,
		Audience:     clientID,
		Scope:        scopes,
		ExpiresIn:    expiry,
		CustomClaims: customClaims,
	}

	switch t.buttonIndex {
	case 1:
		req.ChaosInvalidSignature = true
	}

	gen := core.NewTokenGenerator(t.ctx.PrivKey, t.ctx.Kid, t.ctx.Issuer)
	result, err := gen.Generate(req)
	if err != nil {
		return nil
	}

	t.generatedToken = result.AccessToken
	t.tokenView.SetToken(result.AccessToken)
	t.scrollOffset = 15

	return nil
}
