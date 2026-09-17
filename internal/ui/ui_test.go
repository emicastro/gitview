package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"emicastro.com/gitview/internal/stats"
)

func TestInitAndLoadingView(t *testing.T) {
	t.Parallel()

	m := New("octocat", func(fresh bool) (stats.Snapshot, error) {
		t.Fatal("load should not run until the cmd is invoked")
		return stats.Snapshot{}, nil
	})
	if m.State() != "loading" {
		t.Fatalf("state = %s", m.State())
	}
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init should return a load cmd")
	}
	view := m.View()
	if !strings.Contains(view, "fetching octocat…") {
		t.Fatalf("view = %q", view)
	}
}

func TestQuitKeys(t *testing.T) {
	t.Parallel()

	m := New("octocat", nil)
	msgs := []tea.Msg{
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
		tea.KeyMsg{Type: tea.KeyCtrlC},
	}
	for _, msg := range msgs {
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Fatalf("%#v: expected quit cmd", msg)
		}
		got := cmd()
		if _, ok := got.(tea.QuitMsg); !ok {
			t.Fatalf("%#v: cmd() = %T", msg, got)
		}
	}
}

func TestRefreshAndFailed(t *testing.T) {
	t.Parallel()

	var freshFlag bool
	var calls int
	m := New("octocat", func(fresh bool) (stats.Snapshot, error) {
		calls++
		freshFlag = fresh
		if calls == 1 {
			return stats.Snapshot{}, errors.New("user not found: octocat")
		}
		return stats.Snapshot{User: "octocat", ReposCount: 1, Stars: 2}, nil
	})

	cmd := m.Init()
	next, _ := m.Update(cmd())
	m = next.(Model)
	if m.State() != "failed" {
		t.Fatalf("state = %s", m.State())
	}
	if strings.Contains(m.View(), "fetching") {
		t.Fatalf("spinner hung: %q", m.View())
	}
	if !strings.Contains(m.View(), "user not found: octocat") {
		t.Fatalf("view = %q", m.View())
	}
	if strings.Contains(m.View(), "test-github-token") {
		t.Fatal("token leaked")
	}

	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = next.(Model)
	if m.State() != "loading" {
		t.Fatalf("after r state = %s", m.State())
	}
	if cmd == nil {
		t.Fatal("r should refetch")
	}
	next, _ = m.Update(cmd())
	m = next.(Model)
	if !freshFlag {
		t.Fatal("r should load with fresh=true")
	}
	if m.State() != "ready" {
		t.Fatalf("state = %s", m.State())
	}
	if !strings.Contains(m.View(), "@octocat") {
		t.Fatalf("ready view = %q", m.View())
	}
	if !strings.Contains(m.View(), "q quit  r refresh") {
		t.Fatalf("missing footer: %q", m.View())
	}
}
