package tabs

import (
	"fmt"
	"jwtea/internal/core"
	"jwtea/internal/tui"
	"jwtea/internal/tui/theme"
	"net/url"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LogsTab struct {
	ctx         *tui.Context
	list        list.Model
	follow      bool
	errorOnly   bool
	showDetails bool
	detailItem  *core.LogEntry
	subCh       chan core.LogEntry
	width       int
	height      int

	styleMethod    lipgloss.Style
	styleStatus2   lipgloss.Style
	styleStatus3   lipgloss.Style
	styleStatus4   lipgloss.Style
	styleStatus5   lipgloss.Style
	styleDetailBox lipgloss.Style
	styleDetailKey lipgloss.Style
	styleDetailVal lipgloss.Style
}

type logListItem struct {
	title string
	desc  string
	entry core.LogEntry
}

func (i logListItem) Title() string       { return i.title }
func (i logListItem) Description() string { return i.desc }
func (i logListItem) FilterValue() string { return i.title + " " + i.desc }

func NewLogsTab(ctx *tui.Context) *LogsTab {
	items := []list.Item{}

	if ctx.LogHub != nil {
		snapshot := ctx.LogHub.Snapshot()
		for i := len(snapshot) - 1; i >= 0; i-- {
			items = append(items, toLogItem(snapshot[i]))
		}
	}

	d := list.NewDefaultDelegate()
	d.SetSpacing(0)
	l := list.New(items, d, 0, 0)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.SetFilteringEnabled(true)

	var ch chan core.LogEntry
	if ctx.LogHub != nil {
		ch = ctx.LogHub.Subscribe()
	}

	return &LogsTab{
		ctx:          ctx,
		list:         l,
		follow:       true,
		errorOnly:    false,
		subCh:        ch,
		styleMethod:  lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true),
		styleStatus2: lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		styleStatus3: lipgloss.NewStyle().Foreground(lipgloss.Color("69")),
		styleStatus4: lipgloss.NewStyle().Foreground(lipgloss.Color("219")),
		styleStatus5: lipgloss.NewStyle().Foreground(lipgloss.Color("197")).Bold(true),
		styleDetailBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			Margin(1, 2),
		styleDetailKey: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		styleDetailVal: lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
	}
}

func (t *LogsTab) Init() tea.Cmd {
	return waitForLog(t.subCh)
}

func (t *LogsTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = v.Width
		t.height = v.Height
		t.list.SetWidth(v.Width)
		t.list.SetHeight(v.Height - 8)
		return t, nil

	case tea.KeyMsg:
		if t.showDetails {
			switch v.String() {
			case "esc", "enter", "q":
				t.showDetails = false
				t.detailItem = nil
				return t, nil
			case "c":
				if t.detailItem != nil {
					_ = clipboard.WriteAll(fmt.Sprintf("%s %s", t.detailItem.Method, t.detailItem.Path))
				}
				return t, nil
			}
			return t, nil
		}

		switch v.String() {
		case "j", "down":
			t.follow = false
			t.list.CursorDown()
			return t, nil
		case "k", "up":
			t.follow = false
			t.list.CursorUp()
			return t, nil
		case "enter":
			if i, ok := t.list.SelectedItem().(logListItem); ok {
				t.detailItem = &i.entry
				t.showDetails = true
			}
			return t, nil
		case "c":
			if i, ok := t.list.SelectedItem().(logListItem); ok {
				summary := fmt.Sprintf("%s %s", i.entry.Method, i.entry.Path)
				_ = clipboard.WriteAll(summary)
			}
			return t, nil
		case "f":
			t.follow = !t.follow
			return t, nil
		case "e":
			t.errorOnly = !t.errorOnly
			t.rebuildList()
			return t, nil
		case "g":
			if len(t.list.VisibleItems()) > 0 {
				t.list.Select(0)
			}
			return t, nil
		case "G":
			n := len(t.list.VisibleItems())
			if n > 0 {
				t.list.Select(n - 1)
			}
			return t, nil
		}

	case logMsg:
		if core.LogEntry(v).Path != "/favicon.ico" {
			le := core.LogEntry(v)
			if !t.errorOnly || le.Status >= 400 {
				t.list.InsertItem(0, toLogItem(le))
			}
			if t.follow {
				t.list.Select(0)
			}
		}
		return t, waitForLog(t.subCh)
	}

	var cmd tea.Cmd
	t.list, cmd = t.list.Update(msg)
	return t, cmd
}

func (t *LogsTab) View() string {
	if t.showDetails && t.detailItem != nil {
		return t.viewDetails()
	}

	body := t.list.View()

	followState := "Off"
	if t.follow {
		followState = "On"
	}
	errorState := "Off"
	if t.errorOnly {
		errorState = "On"
	}
	count := len(t.list.Items())

	status := lipgloss.NewStyle().Faint(true).Render(
		fmt.Sprintf("Follow: %s  |  Errors: %s  |  Items: %d", followState, errorState, count),
	)

	footer := lipgloss.NewStyle().Faint(true).Render("enter details • c copy • j/k move • f follow • e errors • g/G jump • / filter")

	return lipgloss.JoinVertical(lipgloss.Left, status, body, footer)
}

func (t *LogsTab) Help() []string {
	return []string{
		"Logs Tab:",
		"  enter       show details",
		"  c           copy path to clipboard",
		"  f           toggle follow (auto-jump to newest)",
		"  e           toggle errors-only view (status >= 400)",
		"  g / G       jump to top/bottom",
		"  /           filter (Esc to clear)",
		"  j/k, ↑/↓    navigate",
	}
}

func (t *LogsTab) IsTextInputActive() bool {
	return t.list.FilterState() == list.Filtering
}

func (t *LogsTab) viewDetails() string {
	e := t.detailItem
	kv := func(k, v string) string {
		return fmt.Sprintf("%s %s", t.styleDetailKey.Render(fmt.Sprintf("%-12s", k+":")), t.styleDetailVal.Render(v))
	}
	decodedPath, _ := url.QueryUnescape(e.Path)
	if decodedPath == "" {
		decodedPath = e.Path
	}
	content := []string{
		theme.Header.Render("Request Details"),
		"",
		kv("Time", e.Time.Format(time.RFC3339)),
		kv("Method", e.Method),
		kv("Status", fmt.Sprintf("%d", e.Status)),
		kv("Duration", e.Duration.String()),
		kv("Remote IP", e.RemoteIP),
		kv("Bytes", fmt.Sprintf("%d", e.Bytes)),
		"",
		kv("Path", decodedPath),
		kv("User-Agent", e.UserAgent),
	}

	box := t.styleDetailBox.Render(strings.Join(content, "\n"))
	help := lipgloss.NewStyle().Faint(true).Render("\n  esc/enter back • c copy path")
	return lipgloss.JoinVertical(lipgloss.Left, box, help)
}

func (t *LogsTab) rebuildList() {
	if t.ctx.LogHub == nil {
		return
	}

	items := []list.Item{}
	snapshot := t.ctx.LogHub.Snapshot()
	for i := len(snapshot) - 1; i >= 0; i-- {
		e := snapshot[i]
		if t.errorOnly && e.Status < 400 {
			continue
		}
		if e.Path == "/favicon.ico" {
			continue
		}
		items = append(items, toLogItem(e))
	}
	t.list.SetItems(items)
	if t.follow && len(items) > 0 {
		t.list.Select(0)
	}
}

type logMsg core.LogEntry

func waitForLog(subCh chan core.LogEntry) tea.Cmd {
	if subCh == nil {
		return nil
	}
	return func() tea.Msg {
		e, ok := <-subCh
		if !ok {
			return nil
		}
		return logMsg(e)
	}
}

func toLogItem(e core.LogEntry) list.Item {
	styleMethod := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	styleStatus2 := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleStatus3 := lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	styleStatus4 := lipgloss.NewStyle().Foreground(lipgloss.Color("219"))
	styleStatus5 := lipgloss.NewStyle().Foreground(lipgloss.Color("197")).Bold(true)

	ts := e.Time.Format("15:04:05.000")
	method := styleMethod.Render(fmt.Sprintf("%-6s", e.Method))
	statusStyled := fmt.Sprintf("%3d", e.Status)
	switch e.Status / 100 {
	case 2:
		statusStyled = styleStatus2.Render(statusStyled)
	case 3:
		statusStyled = styleStatus3.Render(statusStyled)
	case 4:
		statusStyled = styleStatus4.Render(statusStyled)
	default:
		if e.Status >= 500 {
			statusStyled = styleStatus5.Render(statusStyled)
		}
	}

	dur := formatDuration(e.Duration)
	bytes := formatBytes(int64(e.Bytes))

	path := e.Path
	if len(path) > 50 {
		path = path[:47] + "…"
	}
	ua := e.UserAgent
	if len(ua) > 50 {
		ua = ua[:47] + "…"
	}

	title := fmt.Sprintf("%s  %s %-50s  %s  %6s", ts, method, path, statusStyled, dur)
	meta := e.RemoteIP
	if meta == "" {
		meta = "-"
	}
	desc := lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("%s • %s • %s", meta, ua, bytes))
	return logListItem{title: title, desc: desc, entry: e}
}

func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return "0ms"
	}
	if d < time.Millisecond {
		us := d.Round(10 * time.Microsecond).Microseconds()
		return fmt.Sprintf("%dµs", us)
	}
	if d < time.Second {
		ms := float64(d.Microseconds()) / 1000.0
		return fmt.Sprintf("%.1fms", ms)
	}
	if d < time.Minute {
		s := float64(d.Milliseconds()) / 1000.0
		return fmt.Sprintf("%.1fs", s)
	}
	m := int(d / time.Minute)
	s := int((d % time.Minute) / time.Second)
	return fmt.Sprintf("%dm%02ds", m, s)
}

func formatBytes(n int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	if n < kb {
		return fmt.Sprintf("%dB", n)
	}
	if n < mb {
		return fmt.Sprintf("%.1fKB", float64(n)/kb)
	}
	if n < gb {
		return fmt.Sprintf("%.1fMB", float64(n)/mb)
	}
	return fmt.Sprintf("%.1fGB", float64(n)/gb)
}
