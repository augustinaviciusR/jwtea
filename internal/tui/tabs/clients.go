package tabs

import (
	"fmt"
	"sort"
	"strings"

	"jwtea/internal/core"
	"jwtea/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ClientsTab struct {
	ctx     *tui.Context
	cursor  int
	clients []core.Client

	width  int
	height int

	showModal  bool
	modalMode  string
	originalID string

	formID           string
	formSecret       string
	formRedirectURIs string
	formFieldIndex   int

	focusedButton int

	errorMsg string

	styleHeader       lipgloss.Style
	styleClient       lipgloss.Style
	styleCursor       lipgloss.Style
	styleButton       lipgloss.Style
	styleButtonActive lipgloss.Style
	styleBorder       lipgloss.Style
	styleModal        lipgloss.Style
	styleError        lipgloss.Style
}

func NewClientsTab(ctx *tui.Context) *ClientsTab {
	tab := &ClientsTab{
		ctx:               ctx,
		cursor:            0,
		styleHeader:       lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		styleClient:       lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
		styleCursor:       lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		styleButton:       lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		styleButtonActive: lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		styleBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2),
		styleModal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2),
		styleError: lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
	}
	tab.refreshClients()
	return tab
}

func (t *ClientsTab) Init() tea.Cmd {
	return nil
}

func (t *ClientsTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = v.Width
		t.height = v.Height
		return t, nil
	case tea.MouseMsg:
		if !t.showModal {
			return t.handleMouse(v)
		}
		return t, nil
	case tea.KeyMsg:
		if t.showModal {
			return t.handleModalKeys(v)
		}

		switch v.String() {
		case "up", "k":
			if t.focusedButton > 0 {
				t.focusedButton = 0
			} else if t.cursor > 0 {
				t.cursor--
			}
		case "down", "j":
			if t.focusedButton > 0 {
				t.focusedButton = 0
			} else if len(t.clients) > 0 && t.cursor < len(t.clients)-1 {
				t.cursor++
			}
		case "g":
			t.cursor = 0
			t.focusedButton = 0
		case "G":
			if len(t.clients) > 0 {
				t.cursor = len(t.clients) - 1
			} else {
				t.cursor = 0
			}
			t.focusedButton = 0
		case "left", "h":
			if t.focusedButton > 0 {
				t.focusedButton--
				if t.focusedButton == 0 {
					t.focusedButton = 3
				}
			}
		case "right", "l":
			if t.focusedButton == 0 && len(t.clients) > 0 {
				t.focusedButton = 1
			} else if t.focusedButton > 0 && t.focusedButton < 3 {
				t.focusedButton++
			} else if t.focusedButton == 3 {
				t.focusedButton = 1
			}
		case "tab":
			if t.focusedButton == 0 {
				t.focusedButton = 3
			} else {
				t.focusedButton = 0
			}
		case "a":
			t.openAddModal()
		case "e":
			if t.focusedButton == 1 && len(t.clients) > 0 && t.cursor < len(t.clients) {
				t.openEditModal(t.clients[t.cursor])
			}
		case "d":
			if t.focusedButton == 2 && len(t.clients) > 0 && t.cursor < len(t.clients) {
				t.deleteClient(t.clients[t.cursor].ID)
			}
		case "enter", " ":
			return t, t.handleButtonPress()
		}
	}
	return t, nil
}

func (t *ClientsTab) View() string {
	if t.showModal {
		return t.viewModal()
	}

	var b strings.Builder

	headerLeft := t.styleHeader.Render("OAuth Clients")
	addButtonStyle := t.styleButton
	if t.focusedButton == 3 {
		addButtonStyle = t.styleButtonActive
	}
	headerRight := addButtonStyle.Render("[Add Client]")

	contentWidth := t.width - 10
	if contentWidth < 60 {
		contentWidth = 60
	}
	spacing := contentWidth - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	if spacing < 1 {
		spacing = 1
	}
	header := headerLeft + strings.Repeat(" ", spacing) + headerRight

	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", contentWidth))
	b.WriteString("\n")

	if len(t.clients) == 0 {
		b.WriteString(lipgloss.NewStyle().Faint(true).Render("No clients. Press 'a' or select [Add Client] to add one."))
		b.WriteString("\n")
	} else {
		for i, client := range t.clients {
			prefix := " "
			style := t.styleClient
			if i == t.cursor && t.focusedButton == 0 {
				prefix = "●"
				style = t.styleCursor
			}

			clientID := fmt.Sprintf("%-30s", client.ID)
			redirectInfo := fmt.Sprintf("%d URIs", len(client.RedirectURIs))
			if len(client.RedirectURIs) > 0 {
				redirectInfo = fmt.Sprintf("%s (%s...)", redirectInfo, truncate(client.RedirectURIs[0], 20))
			}
			redirectInfo = fmt.Sprintf("%-30s", redirectInfo)

			editStyle := t.styleButton
			delStyle := t.styleButton
			if i == t.cursor {
				switch t.focusedButton {
				case 1:
					editStyle = t.styleButtonActive
				case 2:
					delStyle = t.styleButtonActive
				}
			}

			line := fmt.Sprintf("%s %s %s %s%s",
				prefix,
				style.Render(clientID),
				style.Render(redirectInfo),
				editStyle.Render("[Edit]"),
				delStyle.Render("[Del]"),
			)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	footer := lipgloss.NewStyle().Faint(true).Render("j/k navigate • enter activate • a add • e edit • d delete • g/G jump")
	b.WriteString(footer)

	return t.styleBorder.Render(b.String())
}

func (t *ClientsTab) Help() []string {
	return []string{
		"Clients Tab:",
		"  j/k, ↑/↓    navigate clients",
		"  enter/space activate button",
		"  a           add client",
		"  e           edit client (when Edit focused)",
		"  d           delete client (when Del focused)",
		"  g / G       jump to top/bottom",
		"",
		"Modal (Add/Edit):",
		"  tab         next field",
		"  shift+tab   previous field",
		"  enter       save (changes auto-saved)",
		"  esc         cancel",
	}
}

func (t *ClientsTab) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return t, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if t.cursor > 0 {
			t.cursor--
		}
	case tea.MouseButtonWheelDown:
		if len(t.clients) > 0 && t.cursor < len(t.clients)-1 {
			t.cursor++
		}
	}
	return t, nil
}

func (t *ClientsTab) handleButtonPress() tea.Cmd {
	switch t.focusedButton {
	case 1:
		if len(t.clients) > 0 && t.cursor < len(t.clients) {
			t.openEditModal(t.clients[t.cursor])
		}
	case 2:
		if len(t.clients) > 0 && t.cursor < len(t.clients) {
			t.deleteClient(t.clients[t.cursor].ID)
		}
	case 3:
		t.openAddModal()
	}
	return nil
}

func (t *ClientsTab) openAddModal() {
	t.showModal = true
	t.modalMode = "add"
	t.formID = ""
	t.formSecret = ""
	t.formRedirectURIs = ""
	t.formFieldIndex = 0
	t.errorMsg = ""
}

func (t *ClientsTab) openEditModal(client core.Client) {
	t.showModal = true
	t.modalMode = "edit"
	t.originalID = client.ID
	t.formID = client.ID
	t.formSecret = client.Secret
	t.formRedirectURIs = strings.Join(client.RedirectURIs, ",")
	t.formFieldIndex = 0
	t.errorMsg = ""
}

func (t *ClientsTab) handleModalKeys(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		t.showModal = false
		t.errorMsg = ""
		return t, nil
	case "tab":
		t.formFieldIndex = (t.formFieldIndex + 1) % 3
	case "shift+tab":
		t.formFieldIndex--
		if t.formFieldIndex < 0 {
			t.formFieldIndex = 2
		}
	case "enter":
		return t, t.saveClient()
	case "backspace":
		t.deleteChar()
	default:
		if len(key.String()) == 1 {
			t.addChar(key.String())
		}
	}
	return t, nil
}

func (t *ClientsTab) addChar(char string) {
	switch t.formFieldIndex {
	case 0:
		t.formID += char
	case 1:
		t.formSecret += char
	case 2:
		t.formRedirectURIs += char
	}
}

func (t *ClientsTab) deleteChar() {
	switch t.formFieldIndex {
	case 0:
		if len(t.formID) > 0 {
			t.formID = t.formID[:len(t.formID)-1]
		}
	case 1:
		if len(t.formSecret) > 0 {
			t.formSecret = t.formSecret[:len(t.formSecret)-1]
		}
	case 2:
		if len(t.formRedirectURIs) > 0 {
			t.formRedirectURIs = t.formRedirectURIs[:len(t.formRedirectURIs)-1]
		}
	}
}

func (t *ClientsTab) saveClient() tea.Cmd {
	if t.formID == "" {
		t.errorMsg = "Client ID is required"
		return nil
	}
	if t.formSecret == "" {
		t.errorMsg = "Client Secret is required"
		return nil
	}

	if t.ctx.Store == nil {
		t.showModal = false
		return nil
	}

	var redirectURIs []string
	if t.formRedirectURIs != "" {
		for _, uri := range strings.Split(t.formRedirectURIs, ",") {
			trimmed := strings.TrimSpace(uri)
			if trimmed != "" {
				redirectURIs = append(redirectURIs, trimmed)
			}
		}
	}

	client := core.Client{
		ID:           t.formID,
		Secret:       t.formSecret,
		RedirectURIs: redirectURIs,
	}

	switch t.modalMode {
	case "add":
		t.ctx.Store.AddClient(client)
	case "edit":
		if t.originalID != t.formID {
			t.ctx.Store.DeleteClient(t.originalID)
			t.ctx.Store.AddClient(client)
		} else {
			t.ctx.Store.UpdateClient(client)
		}
	}

	if err := t.ctx.AutoSave(); err != nil {
		t.errorMsg = fmt.Sprintf("Failed to save: %v", err)
		return nil
	}

	t.showModal = false
	t.errorMsg = ""
	t.refreshClients()
	return nil
}

func (t *ClientsTab) deleteClient(id string) {
	if t.ctx.Store == nil {
		return
	}

	t.ctx.Store.DeleteClient(id)

	if err := t.ctx.AutoSave(); err != nil {
		t.errorMsg = fmt.Sprintf("Failed to save: %v", err)
		return
	}

	t.refreshClients()
	t.focusedButton = 0
}

func (t *ClientsTab) refreshClients() {
	if t.ctx.Store == nil {
		t.clients = []core.Client{}
		t.cursor = 0
		return
	}

	t.clients = t.ctx.Store.ListClients()

	sort.Slice(t.clients, func(i, j int) bool {
		return t.clients[i].ID < t.clients[j].ID
	})

	if len(t.clients) == 0 {
		t.cursor = 0
	} else if t.cursor >= len(t.clients) {
		t.cursor = len(t.clients) - 1
	}
}

func (t *ClientsTab) IsTextInputActive() bool {
	return t.showModal
}

func (t *ClientsTab) viewModal() string {
	var b strings.Builder

	title := "Add Client"
	if t.modalMode == "edit" {
		title = "Edit Client"
	}
	b.WriteString(t.styleHeader.Render(title))
	b.WriteString("\n\n")

	idLabel := "Client ID:"
	idStyle := t.styleClient
	if t.formFieldIndex == 0 {
		idLabel = "Client ID: ▶"
		idStyle = t.styleCursor
	}
	idValue := t.formID
	if idValue == "" {
		idValue = "_"
	}
	b.WriteString(idLabel + " " + idStyle.Render(idValue))
	b.WriteString("\n")

	secretLabel := "Secret:"
	secretStyle := t.styleClient
	if t.formFieldIndex == 1 {
		secretLabel = "Secret:    ▶"
		secretStyle = t.styleCursor
	}
	secretValue := t.formSecret
	if secretValue == "" {
		secretValue = "_"
	}
	b.WriteString(secretLabel + "    " + secretStyle.Render(secretValue))
	b.WriteString("\n")

	uriLabel := "Redirects:"
	uriStyle := t.styleClient
	if t.formFieldIndex == 2 {
		uriLabel = "Redirects: ▶"
		uriStyle = t.styleCursor
	}
	uriValue := t.formRedirectURIs
	if uriValue == "" {
		uriValue = "_"
	}
	b.WriteString(uriLabel + " " + uriStyle.Render(uriValue))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("           (comma-separated URLs)"))
	b.WriteString("\n\n")

	if t.errorMsg != "" {
		b.WriteString(t.styleError.Render(t.errorMsg))
		b.WriteString("\n")
	}

	if t.formID == "" {
		b.WriteString(t.styleError.Render("Client ID is required"))
		b.WriteString("\n")
	}
	if t.formSecret == "" {
		b.WriteString(t.styleError.Render("Client Secret is required"))
		b.WriteString("\n")
	}

	b.WriteString(lipgloss.NewStyle().Faint(true).Render("tab/shift+tab navigate • enter save • esc cancel"))

	return t.styleModal.Render(b.String())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
