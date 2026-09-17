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
	user     string
	load     LoadFunc
	state    state
	snap     stats.Snapshot
	err      error
	width    int
	height   int
	cursor   int
	offset   int
	all      bool
	expanded bool
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
		case "tab":
			m.expanded = !m.expanded
			m.clampScroll()
			return m, nil
		case "j", "down":
			if m.state != stateReady || !m.expanded {
				return m, nil
			}
			m.move(1)
			return m, nil
		case "k", "up":
			if m.state != stateReady || !m.expanded {
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
	if !m.expanded {
		return 0
	}
	// Everything the repo panel's rows have to share the screen with: this
	// panel's own border/heading/columns, the help line, and the compact
	// panel when there is room for both.
	used := repoChrome + 1
	if m.showCompact() {
		used += m.compactHeight()
	}
	rows := m.termHeight() - used
	if rows < 1 {
		return 1
	}
	return rows
}

// showCompact reports whether the language panel fits alongside the repo list.
// On a short terminal the list the user just asked for wins the space.
func (m Model) showCompact() bool {
	if !m.expanded {
		return true
	}
	// The repo panel needs its chrome plus at least one row, and the help
	// line needs the last one.
	return m.termHeight() >= m.compactHeight()+repoChrome+1+1
}

// compactHeight is the rendered line count of the compact panel: border,
// header, rule, ribbon, blank, one row per grid row, plus the skip note.
func (m Model) compactHeight() int {
	n := len(m.snap.Languages)
	if n == 0 {
		n = 1
	} else if m.termWidth() >= twoColMin {
		n = (n + 1) / 2
	}
	h := 2 + 4 + n
	if m.snap.Skipped > 0 {
		h += 2
	}
	return h
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

// pad fits s to exactly n display cells, truncating when it is too wide.
// Truncation can land short of n — an ellipsis replacing a double-width
// glyph gives back a column — so the result is padded out either way.
func pad(s string, n int) string {
	if n <= 0 {
		return ""
	}
	out := trunc(s, n)
	w := lipgloss.Width(out)
	if w >= n {
		return out
	}
	return out + strings.Repeat(" ", n-w)
}

// trunc cuts s to at most n display cells, marking the cut with an ellipsis.
// Width is measured in cells, not runes: a double-width glyph costs two.
func trunc(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > n-1 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + "…"
}
