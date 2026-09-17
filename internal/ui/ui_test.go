package ui

import (
	"errors"
	"fmt"
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

func TestLanguageRowsLeftAligned(t *testing.T) {
	t.Parallel()

	m := New("octocat", func(fresh bool) (stats.Snapshot, error) {
		return stats.Snapshot{
			User: "octocat",
			Languages: []stats.Language{
				{Name: "JavaScript", Bytes: 100, Percent: 25.6},
				{Name: "HTML", Bytes: 80, Percent: 22.3},
			},
			Skipped: 4,
		}, nil
	})
	next, _ := m.Update(m.Init()())
	m = next.(Model)
	next, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	view := m.View()
	if strings.Contains(view, "skip ") {
		t.Fatalf("skip warnings in TUI: %q", view)
	}
	if !strings.Contains(stripANSI(view), "4 repos omitted") {
		t.Fatalf("missing skip note: %q", view)
	}
	js := 0
	for _, line := range strings.Split(stripANSI(view), "\n") {
		if !strings.Contains(line, "JavaScript") {
			continue
		}
		js++
		if strings.Index(line, "JavaScript") > 2 {
			t.Fatalf("JavaScript not left-aligned: %q", line)
		}
	}
	if js != 1 {
		t.Fatalf("JavaScript appeared %d times in %q", js, view)
	}
}

func TestListNavigationAndLayout(t *testing.T) {
	t.Parallel()

	const n = 40
	var loads int
	repos := make([]stats.Repo, n)
	for i := range repos {
		repos[i] = stats.Repo{
			Name:      fmt.Sprintf("repo-%02d-with-a-very-long-name-that-must-truncate-XXXXXXXXXXXX", i),
			Stars:     i,
			Language:  "Go",
			UpdatedAt: "2026-01-02",
		}
	}
	m := New("octocat", func(fresh bool) (stats.Snapshot, error) {
		loads++
		return stats.Snapshot{User: "octocat", ReposCount: n, Stars: 99, Repos: repos}, nil
	})
	next, _ := m.Update(m.Init()())
	m = next.(Model)
	next, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Fatal("resize must not fetch")
	}
	m = next.(Model)

	view := m.View()
	if !strings.Contains(view, "> ") {
		t.Fatalf("missing selection: %q", view)
	}
	if strings.Contains(view, "XXXXXXXXXXXX") {
		t.Fatalf("name not truncated: %q", view)
	}
	if !strings.Contains(view, "UPDATED") || !strings.Contains(view, "STARS") {
		t.Fatalf("missing column headers: %q", view)
	}
	if !strings.Contains(view, "2026-01-02") {
		t.Fatalf("missing updated: %q", view)
	}
	if strings.Contains(stripANSI(view), "    0 ") {
		t.Fatalf("zero stars should not be a raw 0 column: %q", view)
	}
	if strings.Contains(view, "repo-39") {
		t.Fatalf("viewport did not clip: %q", view)
	}
	for _, line := range strings.Split(view, "\n") {
		plain := stripANSI(line)
		if len([]rune(plain)) > 80 {
			t.Fatalf("line wider than 80: %d %q", len([]rune(plain)), plain)
		}
	}

	before := loads
	for _, key := range []tea.Msg{
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")},
		tea.KeyMsg{Type: tea.KeyUp},
	} {
		next, cmd = m.Update(key)
		if cmd != nil {
			t.Fatalf("select %T started a fetch", key)
		}
		m = next.(Model)
	}
	if loads != before {
		t.Fatalf("select fetched %d extra times", loads-before)
	}

	for i := 0; i < 30; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = next.(Model)
	}
	view = m.View()
	if strings.Contains(view, "repo-00") {
		t.Fatalf("did not scroll off first repo: %q", view)
	}
	if !strings.Contains(view, "> ") {
		t.Fatalf("lost selection after scroll: %q", view)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
