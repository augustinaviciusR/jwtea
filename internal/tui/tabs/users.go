package tabs

import (
	"fmt"
	"jwtea/internal/core"
	"sort"
	"strings"

	"jwtea/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type UsersTab struct {
	ctx    *tui.Context
	cursor int
	users  []core.User

	width  int
	height int

	showModal   bool
	modalMode   string
	editingUser *core.User

	formEmail      string
	formRole       string
	formDept       string
	formFieldIndex int

	focusedButton int

	errorMsg string

	styleHeader       lipgloss.Style
	styleUser         lipgloss.Style
	styleCursor       lipgloss.Style
	styleButton       lipgloss.Style
	styleButtonActive lipgloss.Style
	styleBorder       lipgloss.Style
	styleModal        lipgloss.Style
	styleError        lipgloss.Style
}

func NewUsersTab(ctx *tui.Context) *UsersTab {
	tab := &UsersTab{
		ctx:               ctx,
		cursor:            0,
		styleHeader:       lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		styleUser:         lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
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
	tab.refreshUsers()
	return tab
}

func (t *UsersTab) Init() tea.Cmd {
	return nil
}

func (t *UsersTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			} else if len(t.users) > 0 && t.cursor < len(t.users)-1 {
				t.cursor++
			}
		case "g":
			t.cursor = 0
			t.focusedButton = 0
		case "G":
			if len(t.users) > 0 {
				t.cursor = len(t.users) - 1
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
			if t.focusedButton == 0 && len(t.users) > 0 {
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
			if t.focusedButton == 1 && t.cursor < len(t.users) {
				t.openEditModal(t.users[t.cursor])
			}
		case "d":
			if t.focusedButton == 2 && len(t.users) > 0 && t.cursor < len(t.users) {
				t.deleteUser(t.users[t.cursor].Email)
			}
		case "enter", " ":
			return t, t.handleButtonPress()
		}
	}
	return t, nil
}

func (t *UsersTab) View() string {
	if t.showModal {
		return t.viewModal()
	}

	var b strings.Builder

	headerLeft := t.styleHeader.Render("Users")
	addButtonStyle := t.styleButton
	if t.focusedButton == 3 {
		addButtonStyle = t.styleButtonActive
	}
	headerRight := addButtonStyle.Render("[Add User]")

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

	if len(t.users) == 0 {
		b.WriteString(lipgloss.NewStyle().Faint(true).Render("No users. Press 'a' or select [Add User] to add one."))
		b.WriteString("\n")
	} else {
		for i, user := range t.users {
			prefix := " "
			style := t.styleUser
			if i == t.cursor && t.focusedButton == 0 {
				prefix = "●"
				style = t.styleCursor
			}

			email := fmt.Sprintf("%-25s", user.Email)
			info := ""
			if user.Role != "" {
				info = fmt.Sprintf("role: %s", user.Role)
			}
			if user.Dept != "" {
				if info != "" {
					info += ", "
				}
				info += fmt.Sprintf("dept: %s", user.Dept)
			}
			if info == "" {
				info = "-"
			}
			info = fmt.Sprintf("%-35s", info)

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
				style.Render(email),
				style.Render(info),
				editStyle.Render("[Edit]"),
				delStyle.Render("[Del]"),
			)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	if t.errorMsg != "" {
		b.WriteString("\n")
		b.WriteString(t.styleError.Render("⚠ " + t.errorMsg))
	}

	b.WriteString("\n")
	footer := lipgloss.NewStyle().Faint(true).Render("j/k navigate • enter activate • a add • e edit • d delete • g/G jump")
	b.WriteString(footer)

	return t.styleBorder.Render(b.String())
}

func (t *UsersTab) Help() []string {
	return []string{
		"Users Tab:",
		"  j/k, ↑/↓    navigate users",
		"  enter/space activate button",
		"  a           add user",
		"  e           edit user (when Edit focused)",
		"  d           delete user (when Del focused)",
		"  g / G       jump to top/bottom",
		"",
		"Modal (Add/Edit):",
		"  tab         next field",
		"  shift+tab   previous field",
		"  enter       save",
		"  esc         cancel",
	}
}

func (t *UsersTab) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return t, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if t.cursor > 0 {
			t.cursor--
		}
	case tea.MouseButtonWheelDown:
		if len(t.users) > 0 && t.cursor < len(t.users)-1 {
			t.cursor++
		}
	}
	return t, nil
}

func (t *UsersTab) handleButtonPress() tea.Cmd {
	switch t.focusedButton {
	case 1:
		if len(t.users) > 0 && t.cursor < len(t.users) {
			t.openEditModal(t.users[t.cursor])
		}
	case 2:
		if len(t.users) > 0 && t.cursor < len(t.users) {
			t.deleteUser(t.users[t.cursor].Email)
		}
	case 3:
		t.openAddModal()
	}
	return nil
}

func (t *UsersTab) openAddModal() {
	t.showModal = true
	t.modalMode = "add"
	t.formEmail = ""
	t.formRole = ""
	t.formDept = ""
	t.formFieldIndex = 0
	t.errorMsg = ""
}

func (t *UsersTab) openEditModal(user core.User) {
	t.showModal = true
	t.modalMode = "edit"
	t.editingUser = &user
	t.formEmail = user.Email
	t.formRole = user.Role
	t.formDept = user.Dept
	t.formFieldIndex = 0
	t.errorMsg = ""
}

func (t *UsersTab) handleModalKeys(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		t.showModal = false
		return t, nil
	case "tab":
		t.formFieldIndex = (t.formFieldIndex + 1) % 3
	case "shift+tab":
		t.formFieldIndex--
		if t.formFieldIndex < 0 {
			t.formFieldIndex = 2
		}
	case "enter":
		return t, t.saveUser()
	case "backspace":
		t.deleteChar()
	default:
		if len(key.String()) == 1 {
			t.addChar(key.String())
		}
	}
	return t, nil
}

func (t *UsersTab) addChar(char string) {
	switch t.formFieldIndex {
	case 0:
		t.formEmail += char
	case 1:
		t.formRole += char
	case 2:
		t.formDept += char
	}
}

func (t *UsersTab) deleteChar() {
	switch t.formFieldIndex {
	case 0:
		if len(t.formEmail) > 0 {
			t.formEmail = t.formEmail[:len(t.formEmail)-1]
		}
	case 1:
		if len(t.formRole) > 0 {
			t.formRole = t.formRole[:len(t.formRole)-1]
		}
	case 2:
		if len(t.formDept) > 0 {
			t.formDept = t.formDept[:len(t.formDept)-1]
		}
	}
}

func (t *UsersTab) saveUser() tea.Cmd {
	if t.formEmail == "" {
		return nil
	}

	if t.ctx.Store == nil {
		t.showModal = false
		return nil
	}

	user := core.User{
		Email: t.formEmail,
		Role:  t.formRole,
		Dept:  t.formDept,
	}

	switch t.modalMode {
	case "add":
		t.ctx.Store.AddUser(user)
	case "edit":
		t.ctx.Store.UpdateUser(user)
	}

	if err := t.ctx.AutoSave(); err != nil {
		t.errorMsg = fmt.Sprintf("Failed to save: %v", err)
		return nil
	}

	t.showModal = false
	t.errorMsg = ""
	t.refreshUsers()
	return nil
}

func (t *UsersTab) deleteUser(email string) {
	if t.ctx.Store == nil {
		return
	}

	t.ctx.Store.DeleteUser(email)

	if err := t.ctx.AutoSave(); err != nil {
		t.errorMsg = fmt.Sprintf("Save failed: %v", err)
	} else {
		t.errorMsg = ""
	}

	t.refreshUsers()
	t.focusedButton = 0
}

func (t *UsersTab) refreshUsers() {
	if t.ctx.Store == nil {
		t.users = []core.User{}
		t.cursor = 0
		return
	}

	t.users = t.ctx.Store.ListUsers()

	sort.Slice(t.users, func(i, j int) bool {
		return t.users[i].Email < t.users[j].Email
	})

	if len(t.users) == 0 {
		t.cursor = 0
	} else if t.cursor >= len(t.users) {
		t.cursor = len(t.users) - 1
	}
}

func (t *UsersTab) IsTextInputActive() bool {
	return t.showModal
}

func (t *UsersTab) viewModal() string {
	var b strings.Builder

	title := "Add User"
	if t.modalMode == "edit" {
		title = "Edit User"
	}
	b.WriteString(t.styleHeader.Render(title))
	b.WriteString("\n\n")

	emailLabel := "Email:"
	emailStyle := t.styleUser
	if t.formFieldIndex == 0 {
		emailLabel = "Email: ▶"
		emailStyle = t.styleCursor
	}
	emailValue := t.formEmail
	if emailValue == "" {
		emailValue = "_"
	}
	b.WriteString(emailLabel + " " + emailStyle.Render(emailValue))
	b.WriteString("\n")

	roleLabel := "Role:"
	roleStyle := t.styleUser
	if t.formFieldIndex == 1 {
		roleLabel = "Role:  ▶"
		roleStyle = t.styleCursor
	}
	roleValue := t.formRole
	if roleValue == "" {
		roleValue = "_"
	}
	b.WriteString(roleLabel + "  " + roleStyle.Render(roleValue))
	b.WriteString("\n")

	deptLabel := "Dept:"
	deptStyle := t.styleUser
	if t.formFieldIndex == 2 {
		deptLabel = "Dept:  ▶"
		deptStyle = t.styleCursor
	}
	deptValue := t.formDept
	if deptValue == "" {
		deptValue = "_"
	}
	b.WriteString(deptLabel + "  " + deptStyle.Render(deptValue))
	b.WriteString("\n\n")

	if t.errorMsg != "" {
		b.WriteString(t.styleError.Render(t.errorMsg))
		b.WriteString("\n")
	}

	if t.formEmail == "" {
		b.WriteString(t.styleError.Render("Email is required"))
		b.WriteString("\n")
	}

	b.WriteString(lipgloss.NewStyle().Faint(true).Render("tab/shift+tab navigate • enter save • esc cancel"))

	return t.styleModal.Render(b.String())
}
