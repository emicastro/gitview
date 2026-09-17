package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"emicastro.com/gitview/internal/render"
	"emicastro.com/gitview/internal/stats"
)

type state int

const (
	stateLoading state = iota
	stateReady
	stateFailed
)

// LoadFunc fetches a snapshot. fresh is true on `r` (same as -fresh).
type LoadFunc func(fresh bool) (stats.Snapshot, error)

type Model struct {
	user  string
	load  LoadFunc
	state state
	snap  stats.Snapshot
	err   error
}

type resultMsg struct {
	snap stats.Snapshot
	err  error
}

func New(user string, load LoadFunc) Model {
	return Model{user: user, load: load, state: stateLoading}
}

func (m Model) Init() tea.Cmd {
	return m.fetchCmd(false)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	switch m.state {
	case stateLoading:
		fmt.Fprintf(&b, "fetching %s…", m.user)
	case stateFailed:
		err := "load failed"
		if m.err != nil {
			err = m.err.Error()
		}
		fmt.Fprintf(&b, "error: %s", err)
	default:
		_ = render.Text(&b, m.snap)
	}
	fmt.Fprintf(&b, "\n%s", footerStyle.Render("q quit  r refresh"))
	return b.String()
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

var footerStyle = lipgloss.NewStyle().Faint(true)
