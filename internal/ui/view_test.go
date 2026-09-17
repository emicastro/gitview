package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"emicastro.com/gitview/internal/stats"
)

// fillMark and trackMark are the styles the panel itself paints bars with;
// the tests only care that the fill reaches the partial glyph.
var (
	fillMark  = fillStyle("Rust")
	trackMark = trackStyle
)

func TestBarWidthIsExact(t *testing.T) {
	t.Parallel()

	for _, cells := range []int{10, 19, 20, 60} {
		for _, pct := range []float64{0, 0.4, 1, 5, 12.5, 33.3, 50, 66.7, 99.9, 100, 120, -5} {
			got := bar(pct, cells, fillMark, trackMark)
			if w := lipgloss.Width(got); w != cells {
				t.Fatalf("bar(%v, %d) width = %d, want %d (%q)", pct, cells, w, cells, stripANSI(got))
			}
		}
	}
	if got := bar(50, 0, fillMark, trackMark); got != "" {
		t.Fatalf("bar with no cells = %q", got)
	}
}

// A single percentage point must still show on a 20-cell track, which is the
// whole reason for the eighth-width glyphs.
func TestBarShowsSinglePercent(t *testing.T) {
	t.Parallel()

	got := stripANSI(bar(1, 20, fillMark, trackMark))
	if strings.TrimRight(got, emptyBlock) == "" {
		t.Fatalf("1%% on a 20-cell track rendered as an empty track: %q", got)
	}
}

// The partial glyph carries the fill color; painted in the track color it
// reads as a gap rather than as progress.
func TestBarPartialGlyphUsesFill(t *testing.T) {
	t.Parallel()

	got := bar(33.3, 20, fillMark, trackMark)
	plain := stripANSI(got)
	idx := strings.IndexFunc(plain, func(r rune) bool {
		return r != []rune(fullBlock)[0]
	})
	if idx < 0 || string(plain[idx:idx+len(blocks[4])]) == emptyBlock {
		t.Fatalf("no partial glyph in %q", plain)
	}
	partial := plain[idx : idx+len(blocks[4])]
	if !strings.Contains(got, fillMark.Render(partial)) {
		t.Fatalf("partial glyph %q not painted in the fill style: %q", partial, got)
	}
}

// The ribbon must sum to exactly the panel width or the right border wobbles.
func TestRibbonWidthIsExact(t *testing.T) {
	t.Parallel()

	sets := map[string][]float64{
		"thirds":   {33.3, 33.3, 33.4},
		"sevenths": {14.3, 14.3, 14.3, 14.3, 14.3, 14.2, 14.3},
		"skewed":   {97.5, 1.0, 0.9, 0.6},
		"tiny":     {0.4, 0.3, 0.3},
		"single":   {100},
		"over":     {60, 60},
	}
	for name, pcts := range sets {
		langs := make([]stats.Language, len(pcts))
		for i, p := range pcts {
			langs[i] = stats.Language{Name: fmt.Sprintf("lang%d", i), Percent: p}
		}
		for _, cells := range []int{19, 34, 47, 79, 83} {
			if w := lipgloss.Width(ribbon(langs, cells)); w != cells {
				t.Fatalf("ribbon(%s, %d) width = %d", name, cells, w)
			}
		}
	}
	if w := lipgloss.Width(ribbon(nil, 40)); w != 40 {
		t.Fatalf("empty ribbon width = %d", w)
	}
	if got := ribbon(nil, 0); got != "" {
		t.Fatalf("ribbon with no cells = %q", got)
	}
}

func demoModel(t *testing.T, expanded bool) Model {
	t.Helper()
	repos := make([]stats.Repo, 12)
	for i := range repos {
		repos[i] = stats.Repo{
			Name:      fmt.Sprintf("repo-%02d", i),
			Stars:     i,
			Language:  "Go",
			UpdatedAt: "2026-09-17",
		}
	}
	m := New("emicastro", func(bool) (stats.Snapshot, error) {
		return stats.Snapshot{
			User:       "emicastro",
			ReposCount: 12,
			Stars:      48,
			Languages: []stats.Language{
				{Name: "Rust", Percent: 34.0},
				{Name: "Go", Percent: 20.0},
				{Name: "JavaScript", Percent: 10.0},
				{Name: "C", Percent: 10.0},
				{Name: "Other", Percent: 26.0},
			},
			Repos: repos,
		}, nil
	}, true)
	next, _ := m.Update(m.Init()())
	m = next.(Model)
	if expanded {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = next.(Model)
	}
	return m
}

// Nothing about the layout is constant: every size is recomputed from the
// window message, and no line may exceed the terminal.
func TestLayoutFitsEveryWidth(t *testing.T) {
	t.Parallel()

	for _, dim := range [][2]int{{200, 50}, {120, 30}, {85, 24}, {84, 24}, {80, 24}, {70, 24}, {60, 24}, {59, 24}, {50, 24}, {40, 20}, {30, 20}, {20, 10}, {10, 6}, {9, 6}, {6, 4}, {3, 3}, {1, 1}} {
		for _, expanded := range []bool{false, true} {
			m := demoModel(t, expanded)
			next, _ := m.Update(tea.WindowSizeMsg{Width: dim[0], Height: dim[1]})
			m = next.(Model)
			for _, line := range strings.Split(m.View(), "\n") {
				if w := lipgloss.Width(stripANSI(line)); w > dim[0] {
					t.Fatalf("%dx%d expanded=%v: line %d wide: %q", dim[0], dim[1], expanded, w, stripANSI(line))
				}
			}
		}
	}
}

// The view must never render more lines than the terminal has rows, or the
// alt screen scrolls and the panel borders come apart.
func TestLayoutFitsEveryHeight(t *testing.T) {
	t.Parallel()

	for _, dim := range [][2]int{{85, 60}, {85, 30}, {85, 24}, {85, 18}, {85, 16}, {85, 12}, {85, 9}, {85, 8}, {85, 7}, {85, 4}, {85, 2}, {85, 1}, {80, 24}, {80, 14}, {50, 24}, {50, 10}} {
		for _, expanded := range []bool{false, true} {
			m := demoModel(t, expanded)
			next, _ := m.Update(tea.WindowSizeMsg{Width: dim[0], Height: dim[1]})
			m = next.(Model)
			if got := len(strings.Split(m.View(), "\n")); got > dim[1] {
				t.Fatalf("%dx%d expanded=%v: %d lines\n%s", dim[0], dim[1], expanded, got, stripANSI(m.View()))
			}
		}
	}
}

// When both panels cannot fit, the list the user just asked for wins.
func TestShortTerminalPrefersTheRepoList(t *testing.T) {
	t.Parallel()

	m := demoModel(t, true)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 85, Height: 12})
	m = next.(Model)
	view := stripANSI(m.View())
	if !strings.Contains(view, "Recently updated") {
		t.Fatalf("repo list dropped on a short terminal: %q", view)
	}
	if strings.Contains(view, "gitview  ·  @") {
		t.Fatalf("both panels drawn with no room for both: %q", view)
	}
}

// However narrow the terminal, a loaded snapshot still shows something: the
// ribbon fits any width, and a blank screen would look like a hang.
func TestNarrowTerminalStillRenders(t *testing.T) {
	t.Parallel()

	for _, w := range []int{1, 3, 6, 9} {
		m := demoModel(t, false)
		next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: 20})
		m = next.(Model)
		view := m.View()
		if strings.TrimSpace(stripANSI(view)) == "" {
			t.Fatalf("blank view at %d columns", w)
		}
		if got := lipgloss.Width(stripANSI(strings.Split(view, "\n")[0])); got > w {
			t.Fatalf("%d columns: first line %d wide", w, got)
		}
	}
}

// pad and trunc are the panel's width discipline: a double-width glyph costs
// two columns, not one, or the border is pushed out.
func TestPadAndTruncCountCells(t *testing.T) {
	t.Parallel()

	for _, s := range []string{"gitview", "日本語のリポジトリ", "mixed-日本語", "", "…"} {
		for _, n := range []int{0, 1, 2, 5, 12, 40} {
			if got := lipgloss.Width(pad(s, n)); got != n {
				t.Fatalf("pad(%q, %d) is %d cells", s, n, got)
			}
			if got := lipgloss.Width(trunc(s, n)); got > n {
				t.Fatalf("trunc(%q, %d) is %d cells", s, n, got)
			}
		}
	}
}

// The panel keeps the spec's proportions instead of stretching, so bars stay
// comparable between runs.
func TestPanelCapsAtNaturalWidth(t *testing.T) {
	t.Parallel()

	m := demoModel(t, false)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	m = next.(Model)
	for _, line := range strings.Split(m.View(), "\n") {
		if strings.HasPrefix(stripANSI(line), "╭") && lipgloss.Width(stripANSI(line)) != panelMax {
			t.Fatalf("panel width = %d, want %d", lipgloss.Width(stripANSI(line)), panelMax)
		}
	}
}

func TestTwoColumnGridAboveBreakpoint(t *testing.T) {
	t.Parallel()

	m := demoModel(t, false)
	next, _ := m.Update(tea.WindowSizeMsg{Width: twoColMin, Height: 24})
	m = next.(Model)
	var paired int
	for _, line := range strings.Split(stripANSI(m.View()), "\n") {
		if strings.Contains(line, "Rust") && strings.Contains(line, "Go") {
			paired++
		}
	}
	if paired != 1 {
		t.Fatalf("expected Rust and Go on one row at %d cols: %q", twoColMin, stripANSI(m.View()))
	}

	next, _ = m.Update(tea.WindowSizeMsg{Width: twoColMin - 1, Height: 24})
	m = next.(Model)
	for _, line := range strings.Split(stripANSI(m.View()), "\n") {
		if strings.Contains(line, "Rust") && strings.Contains(line, "Go") {
			t.Fatalf("still two columns below the breakpoint: %q", line)
		}
	}
}

// Below the bars breakpoint the ribbon plus percentages is all that is left.
func TestNarrowDropsBars(t *testing.T) {
	t.Parallel()

	m := demoModel(t, false)
	next, _ := m.Update(tea.WindowSizeMsg{Width: barsMin - 10, Height: 24})
	m = next.(Model)
	view := stripANSI(m.View())
	if !strings.Contains(view, "34.0") {
		t.Fatalf("missing percentages: %q", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Rust") && strings.Contains(line, emptyBlock) {
			t.Fatalf("bar track survived below %d cols: %q", barsMin, line)
		}
	}
}

// On a very short terminal the ribbon still answers the at-a-glance question.
func TestShortTerminalKeepsRibbonAndHelp(t *testing.T) {
	t.Parallel()

	m := demoModel(t, false)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: minRows - 2})
	m = next.(Model)
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 2 {
		t.Fatalf("want ribbon and help only, got %d lines: %q", len(lines), stripANSI(view))
	}
	if lipgloss.Width(lines[0]) != m.innerWidth() {
		t.Fatalf("ribbon width = %d, want %d", lipgloss.Width(lines[0]), m.innerWidth())
	}
	if !strings.Contains(stripANSI(lines[1]), "q quit") {
		t.Fatalf("missing help line: %q", stripANSI(lines[1]))
	}
}

// The folded bucket reads as a group, not a language, and sorts last.
func TestOtherBucketRendersLast(t *testing.T) {
	t.Parallel()

	m := demoModel(t, false)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	view := stripANSI(m.View())
	if !strings.Contains(view, "other ") {
		t.Fatalf("Other bucket not lowercased: %q", view)
	}
	if strings.Index(view, "other ") < strings.Index(view, "Rust") {
		t.Fatalf("other should sort last: %q", view)
	}
}

func TestTabTogglesRepoList(t *testing.T) {
	t.Parallel()

	m := demoModel(t, false)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 85, Height: 30})
	m = next.(Model)
	if strings.Contains(m.View(), "Recently updated") {
		t.Fatal("repo list visible while collapsed")
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Fatal("tab must not fetch")
	}
	m = next.(Model)
	if !strings.Contains(m.View(), "Recently updated") {
		t.Fatalf("tab did not expand: %q", stripANSI(m.View()))
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(Model)
	if strings.Contains(m.View(), "Recently updated") {
		t.Fatalf("tab did not collapse: %q", stripANSI(m.View()))
	}
}

// A fallback hue that also appears in the fixed map would render two
// different languages as the same ribbon segment.
func TestFallbackAccentsAreDisjoint(t *testing.T) {
	t.Parallel()

	mapped := make(map[lipgloss.AdaptiveColor]string, len(langAccents))
	for name, c := range langAccents {
		mapped[c] = name
	}
	for _, c := range fallbackAccents {
		if name, ok := mapped[c]; ok {
			t.Fatalf("fallback accent %v collides with %s", c, name)
		}
	}
}

// An unmapped language keeps the same hue across renders and runs.
func TestLangColorIsStable(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"Vue", "Elixir", "GDScript", "Zig"} {
		first := langColor(name)
		for i := 0; i < 5; i++ {
			if langColor(name) != first {
				t.Fatalf("%s changed color between calls", name)
			}
		}
	}
	if langColor("Go") != sky {
		t.Fatal("mapped language lost its accent")
	}
	if langColor(otherBucket) != overlay0 {
		t.Fatal("the folded bucket must stay neutral grey")
	}
}
