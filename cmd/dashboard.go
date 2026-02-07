package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jwtea/internal/tui"
	"jwtea/internal/tui/tabs"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultTickInterval = 1000 * time.Millisecond
	appVersion          = "v0.1.0"
	appName             = "JWTea"

	tabGenerate = 0
	tabUsers    = 1
	tabClients  = 2
	tabLogs     = 3
	tabSettings = 4
	totalTabs   = 5

	colorPrimary   = "205"
	colorSecondary = "240"
	colorSuccess   = "42"
)

type TabHelper interface {
	Help() []string
}

type TextInputChecker interface {
	IsTextInputActive() bool
}

type DashboardTheme struct {
	Header      lipgloss.Style
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style
	Success     lipgloss.Style
	Faint       lipgloss.Style
	HelpBox     lipgloss.Style
}

func NewDashboardTheme() DashboardTheme {
	return DashboardTheme{
		Header:      lipgloss.NewStyle().Foreground(lipgloss.Color(colorPrimary)).Bold(true),
		TabActive:   createTabStyle(colorPrimary, true),
		TabInactive: createTabStyle(colorSecondary, false),
		Success:     lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)),
		Faint:       lipgloss.NewStyle().Faint(true),
		HelpBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorPrimary)).
			Padding(1, 2),
	}
}

func createTabStyle(color string, bold bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(color)).
		Padding(0, 1)
	if bold {
		style = style.Bold(true)
	}
	return style
}

type (
	tickMsg time.Time
)

type dashModel struct {
	ctx          *tui.Context
	theme        DashboardTheme
	tickInterval time.Duration

	activeTab int
	tabNames  []string
	tabs      []tea.Model

	showHelp   bool
	pulseOn    bool
	pulseUntil time.Time
	width      int
	height     int
}

func newDashModelWithConfig(ctx *tui.Context, tickInterval time.Duration) dashModel {
	tabs := createAllTabs(ctx)
	tabNames := []string{"Generate", "Users", "Clients", "Logs", "Settings"}

	return dashModel{
		ctx:          ctx,
		theme:        NewDashboardTheme(),
		tickInterval: tickInterval,
		activeTab:    tabGenerate,
		tabNames:     tabNames,
		tabs:         tabs,
		showHelp:     false,
	}
}

func createAllTabs(ctx *tui.Context) []tea.Model {
	return []tea.Model{
		tabs.NewGenerateTab(ctx),
		tabs.NewUsersTab(ctx),
		tabs.NewClientsTab(ctx),
		tabs.NewLogsTab(ctx),
		tabs.NewSettingsTab(ctx),
	}
}

func (m dashModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.createTickCommand()}

	for _, tab := range m.tabs {
		cmds = append(cmds, tab.Init())
	}

	return tea.Batch(cmds...)
}

func (m dashModel) createTickCommand() tea.Cmd {
	return tea.Tick(m.tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m dashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowResize(v)
	case tea.KeyMsg:
		return m.handleKeyPress(v)
	case tea.MouseMsg:
		return m.routeToActiveTab(v)
	case tickMsg:
		return m.handleTick()
	}

	return m.routeToActiveTab(msg)
}

func (m dashModel) handleWindowResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	var cmds []tea.Cmd
	for i := range m.tabs {
		var cmd tea.Cmd
		m.tabs[i], cmd = m.tabs[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m dashModel) handleKeyPress(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		return m.handleHelpKeys(key)
	}

	return m.handleGlobalKeys(key)
}

func (m dashModel) handleHelpKeys(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "q", "?":
		m.showHelp = false
	}
	return m, nil
}

func (m dashModel) handleGlobalKeys(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "1", "2", "3", "4", "5":
		if checker, ok := m.tabs[m.activeTab].(TextInputChecker); ok && checker.IsTextInputActive() {
			return m.routeToActiveTab(key)
		}
		tabNum := int(key.String()[0] - '0')
		if m.isValidTabNumber(tabNum) && m.activeTab == tabNum-1 {
			return m.routeToActiveTab(key)
		}
		return m.switchToTabByNumber(key.String())
	}

	return m.routeToActiveTab(key)
}

func (m dashModel) switchToTabByNumber(keyStr string) (tea.Model, tea.Cmd) {
	tabNum := int(keyStr[0] - '0')
	if m.isValidTabNumber(tabNum) {
		m.activeTab = tabNum - 1
	}
	return m, nil
}

func (m dashModel) isValidTabNumber(tabNum int) bool {
	return tabNum > 0 && tabNum <= len(m.tabs)
}

func (m dashModel) handleTick() (tea.Model, tea.Cmd) {
	if m.pulseOn && time.Now().After(m.pulseUntil) {
		m.pulseOn = false
	}
	return m, m.createTickCommand()
}

func (m dashModel) routeToActiveTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.tabs[m.activeTab], cmd = m.tabs[m.activeTab].Update(msg)
	return m, cmd
}

func (m dashModel) View() string {
	if m.showHelp {
		return m.renderHelpView()
	}
	return m.renderMainView()
}

func (m dashModel) renderMainView() string {
	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString(m.renderTabNavigation())
	b.WriteString(m.renderActiveTabContent())
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m dashModel) renderHeader() string {
	pulse := m.renderPulse()
	appInfo := m.renderAppInfo()
	serverStatus := m.renderServerStatus()

	return pulse + appInfo + serverStatus + "\n"
}

func (m dashModel) renderPulse() string {
	if m.pulseOn {
		return m.theme.Header.Render("AUTH HIT • ")
	}
	return ""
}

func (m dashModel) renderAppInfo() string {
	return m.theme.Header.Render(fmt.Sprintf("%s %s  ", appName, appVersion))
}

func (m dashModel) renderServerStatus() string {
	if !m.ctx.ServerRunning {
		return m.theme.Faint.Render("Standalone Mode")
	}

	status := m.theme.Success.Render("● Server Running") + " "
	if m.ctx.Issuer != "" {
		status += m.theme.Faint.Render(m.ctx.Issuer)
	}
	return status
}

func (m dashModel) renderTabNavigation() string {
	tabViews := m.buildTabViews()
	tabs := lipgloss.JoinHorizontal(lipgloss.Left, tabViews...)
	return tabs + "\n\n"
}

func (m dashModel) buildTabViews() []string {
	tabViews := make([]string, 0, len(m.tabNames))
	for i, name := range m.tabNames {
		style := m.getTabStyle(i)
		label := fmt.Sprintf("%d:%s", i+1, name)
		tabViews = append(tabViews, style.Render(label))
	}
	return tabViews
}

func (m dashModel) getTabStyle(index int) lipgloss.Style {
	if index == m.activeTab {
		return m.theme.TabActive
	}
	return m.theme.TabInactive
}

func (m dashModel) renderActiveTabContent() string {
	return m.tabs[m.activeTab].View()
}

func (m dashModel) renderFooter() string {
	return "\n" + m.theme.Faint.Render("1-5 switch tabs • ? help • q quit")
}

func (m dashModel) renderHelpView() string {
	globalHelp := m.buildGlobalHelp()
	tabHelp := m.buildActiveTabHelp()

	sections := []string{globalHelp, tabHelp}
	content := strings.Join(sections, "\n")

	helpBox := m.theme.HelpBox.Render(content)
	footer := m.theme.Faint.Render("\nPress ? or esc to close help")

	return lipgloss.JoinVertical(lipgloss.Left, helpBox, footer)
}

func (m dashModel) buildGlobalHelp() string {
	helpLines := []string{
		"Global Keys:",
		"  1-5         switch to tab 1-5",
		"  ?           toggle help",
		"  q / Ctrl+C  quit",
		"",
	}
	return strings.Join(helpLines, "\n")
}

func (m dashModel) buildActiveTabHelp() string {
	if helper, ok := m.tabs[m.activeTab].(TabHelper); ok {
		return strings.Join(helper.Help(), "\n")
	}
	return ""
}

func startDashboardWithConfig(ctx context.Context, tuiCtx *tui.Context, tickInterval time.Duration) error {
	m := newDashModelWithConfig(tuiCtx, tickInterval)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	errCh := make(chan error, 1)
	go func() {
		_, err := p.Run()
		errCh <- err
	}()

	return waitForDashboardExit(ctx, p, errCh)
}

func waitForDashboardExit(ctx context.Context, p *tea.Program, errCh <-chan error) error {
	select {
	case <-ctx.Done():
		p.Quit()
		<-errCh
		return nil
	case err := <-errCh:
		return err
	}
}

func runDashboardWithContext(tuiCtx *tui.Context, stop <-chan struct{}) {
	runDashboardWithConfig(tuiCtx, stop, defaultTickInterval)
}

func runDashboardWithConfig(tuiCtx *tui.Context, stop <-chan struct{}, tickInterval time.Duration) {
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-stop
		cancel()
	}()

	_ = startDashboardWithConfig(cancelCtx, tuiCtx, tickInterval)
}
