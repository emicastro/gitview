package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"emicastro.com/gitview/internal/stats"
)

type state int

const (
	stateLoading state = iota
	stateReady
	stateFailed
)

const (
	defaultWidth  = 80
	defaultHeight = 24
)

// LoadFunc fetches a snapshot. fresh is true on `r` (same as -fresh).
type LoadFunc func(fresh bool) (stats.Snapshot, error)

type Model struct {
	user   string
	load   LoadFunc
	state  state
	snap   stats.Snapshot
	err    error
	width  int
	height int
	cursor int
	offset int
	all    bool
}

type resultMsg struct {
	snap stats.Snapshot
	err  error
}

func New(user string, load LoadFunc, all bool) Model {
	return Model{
		user:   user,
		load:   load,
		state:  stateLoading,
		width:  defaultWidth,
		height: defaultHeight,
		all:    all,
	}
}

func (m Model) Init() tea.Cmd {
	return m.fetchCmd(false)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width < 1 {
			m.width = defaultWidth
		}
		if m.height < 1 {
			m.height = defaultHeight
		}
		m.clampScroll()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			if m.state == stateLoading {
				return m, nil
			}
			m.state = stateLoading
			m.err = nil
			return m, m.fetchCmd(true)
		case "j", "down":
			if m.state != stateReady {
				return m, nil
			}
			m.move(1)
			return m, nil
		case "k", "up":
			if m.state != stateReady {
				return m, nil
			}
			m.move(-1)
			return m, nil
		}
	case resultMsg:
		if msg.err != nil {
			m.state = stateFailed
			m.err = msg.err
			return m, nil
		}
		m.state = stateReady
		m.snap = msg.snap
		m.err = nil
		m.cursor = 0
		m.offset = 0
		return m, nil
	}
	return m, nil
}

func (m Model) listed() []stats.Repo {
	return stats.Visible(m.snap.Repos, m.all)
}

func (m *Model) move(delta int) {
	n := len(m.listed())
	if n == 0 {
		m.cursor = 0
		m.offset = 0
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
	m.clampScroll()
}

func (m *Model) clampScroll() {
	n := len(m.listed())
	vis := m.repoRows()
	if vis < 1 {
		vis = 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vis {
		m.offset = m.cursor - vis + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
	maxOff := n - vis
	if maxOff < 0 {
		maxOff = 0
	}
	if m.offset > maxOff {
		m.offset = maxOff
	}
}

func (m Model) repoRows() int {
	h := m.height
	if h < 1 {
		h = defaultHeight
	}
	// header + languages + skipped + blank + rule + title + columns + footer
	used := 1 + len(m.snap.Languages) + 1 + 1 + 1 + 1 + 1
	if m.snap.Skipped > 0 {
		used++
	}
	rows := h - used
	if rows < 1 {
		return 1
	}
	return rows
}

func (m Model) View() string {
	w := m.width
	if w < 1 {
		w = defaultWidth
	}
	var b strings.Builder
	switch m.state {
	case stateLoading:
		fmt.Fprintf(&b, "fetching %s…", m.user)
	case stateFailed:
		err := "load failed"
		if m.err != nil {
			err = m.err.Error()
		}
		fmt.Fprintf(&b, "error: %s", trunc(err, w))
	default:
		m.writeReady(&b, w)
	}
	fmt.Fprintf(&b, "\n%s", footerStyle.Render("q quit  r refresh"))
	return b.String()
}

func (m Model) writeReady(b *strings.Builder, w int) {
	src := "live"
	if m.snap.Cached {
		src = "cached"
	}
	head := fmt.Sprintf("@%s   %d repos   %d stars   %s", m.snap.User, m.snap.ReposCount, m.snap.Stars, src)
	fmt.Fprintf(b, "%s\n", headerStyle.Render(trunc(head, w)))
	for _, l := range m.snap.Languages {
		name := lipgloss.NewStyle().Width(12).MaxWidth(12).Foreground(langColor(l.Name)).Render(trunc(l.Name, 12))
		pct := lipgloss.NewStyle().Width(7).Align(lipgloss.Right).Faint(true).Render(fmt.Sprintf("%.1f%%", l.Percent))
		row := lipgloss.JoinHorizontal(lipgloss.Center, name, " ", colorBar(l.Percent, l.Name), " ", pct)
		fmt.Fprintf(b, "%s\n", row)
	}
	if m.snap.Skipped > 0 {
		note := fmt.Sprintf("%d repos omitted (no language data)", m.snap.Skipped)
		fmt.Fprintf(b, "%s\n", colStyle.Render(trunc(note, w)))
	}

	fmt.Fprintln(b)
	ruleW := w
	if ruleW > 40 {
		ruleW = 40
	}
	if ruleW < 8 {
		ruleW = 8
	}
	fmt.Fprintf(b, "%s\n", colStyle.Render(strings.Repeat("─", ruleW)))
	fmt.Fprintf(b, "%s\n", headerStyle.Render(trunc("Recently updated", w)))

	nameW := w - 2 - 1 - 5 - 1 - 8 - 1 - 10
	if nameW < 4 {
		nameW = 4
	}
	fmt.Fprintf(b, "%s\n", colStyle.Render(trunc(
		fmt.Sprintf("  %s %5s %-8s %s", pad("NAME", nameW), "STARS", "LANG", "UPDATED"),
		w,
	)))

	repos := stats.Visible(m.snap.Repos, m.all)
	vis := m.repoRows()
	end := m.offset + vis
	if end > len(repos) {
		end = len(repos)
	}
	if m.offset > len(repos) {
		return
	}
	for i := m.offset; i < end; i++ {
		r := repos[i]
		mark := "  "
		if i == m.cursor {
			mark = "> "
		}
		stars := "    —"
		if r.Stars > 0 {
			stars = fmt.Sprintf("%5d", r.Stars)
		}
		line := fmt.Sprintf("%s%s %s %-8s %s", mark, pad(trunc(r.Name, nameW), nameW), stars, trunc(r.Language, 8), r.UpdatedAt)
		line = trunc(line, w)
		if i == m.cursor {
			line = selStyle.Render(line)
		}
		fmt.Fprintf(b, "%s\n", line)
	}
}

func (m Model) fetchCmd(fresh bool) tea.Cmd {
	load := m.load
	return func() tea.Msg {
		if load == nil {
			return resultMsg{err: fmt.Errorf("load failed")}
		}
		snap, err := load(fresh)
		return resultMsg{snap: snap, err: err}
	}
}

func (m Model) State() string {
	switch m.state {
	case stateLoading:
		return "loading"
	case stateReady:
		return "ready"
	case stateFailed:
		return "failed"
	default:
		return "loading"
	}
}

const barWidth = 20

func colorBar(pct float64, lang string) string {
	n := int(pct/100*barWidth + 0.5)
	if n < 0 {
		n = 0
	}
	if n > barWidth {
		n = barWidth
	}
	fill := lipgloss.NewStyle().Foreground(langColor(lang)).Render(strings.Repeat("█", n))
	empty := lipgloss.NewStyle().Faint(true).Render(strings.Repeat("░", barWidth-n))
	return fill + empty
}

func langColor(name string) lipgloss.Color {
	switch name {
	case "Go":
		return "#00ADD8"
	case "Python":
		return "#3572A5"
	case "JavaScript":
		return "#f1e05a"
	case "TypeScript":
		return "#3178c6"
	case "Rust":
		return "#dea584"
	case "C":
		return "#555555"
	case "C++":
		return "#f34b7d"
	case "Java":
		return "#b07219"
	case "Ruby":
		return "#701516"
	case "HTML":
		return "#e34c26"
	case "CSS":
		return "#563d7c"
	case "Shell":
		return "#89e051"
	case "Other":
		return "#6e7681"
	default:
		return "#58a6ff"
	}
}

func pad(s string, n int) string {
	r := []rune(s)
	if len(r) >= n {
		return string(r[:n])
	}
	return s + strings.Repeat(" ", n-len(r))
}

func trunc(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

var (
	footerStyle = lipgloss.NewStyle().Faint(true)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7ee787"))
	colStyle    = lipgloss.NewStyle().Faint(true)
	selStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0d1117")).Background(lipgloss.Color("#58a6ff"))
)
