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
}

type resultMsg struct {
	snap stats.Snapshot
	err  error
}

func New(user string, load LoadFunc) Model {
	return Model{
		user:   user,
		load:   load,
		state:  stateLoading,
		width:  defaultWidth,
		height: defaultHeight,
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

func (m *Model) move(delta int) {
	n := len(m.snap.Repos)
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
	n := len(m.snap.Repos)
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
	// header + languages + footer
	used := 1 + len(m.snap.Languages) + 1
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
	fmt.Fprintf(b, "%s\n", trunc(fmt.Sprintf("@%s  repos=%d  stars=%d  %s", m.snap.User, m.snap.ReposCount, m.snap.Stars, src), w))
	for _, l := range m.snap.Languages {
		fmt.Fprintf(b, "%s\n", trunc(fmt.Sprintf("%s %s %5.1f%%", pad(l.Name, 12), bar(l.Percent), l.Percent), w))
	}

	repos := m.snap.Repos
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
		nameW := w - 2 - 1 - 5 - 1 - 8 - 1 - 10
		if nameW < 4 {
			nameW = 4
		}
		line := fmt.Sprintf("%s%s %5d %-8s %s", mark, pad(trunc(r.Name, nameW), nameW), r.Stars, trunc(r.Language, 8), r.UpdatedAt)
		fmt.Fprintf(b, "%s\n", trunc(line, w))
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

func bar(pct float64) string {
	n := int(pct/100*barWidth + 0.5)
	if n < 0 {
		n = 0
	}
	if n > barWidth {
		n = barWidth
	}
	return strings.Repeat("█", n) + strings.Repeat(" ", barWidth-n)
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

var footerStyle = lipgloss.NewStyle().Faint(true)
