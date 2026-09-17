package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"emicastro.com/gitview/internal/stats"
)

// Panel geometry. The natural width is the spec's: two grid cells of
// name + track + gutter + percentage, separated by colGap, inside 2 cells of
// padding and the border. Wider terminals do not stretch the panel; the bars
// stay comparable between runs.
const (
	panelPadding = 2
	panelChrome  = 2 + 2*panelPadding // border plus padding, both sides
	panelMax     = 85

	nameW     = 11
	barGutter = 2
	pctW      = 6
	colGap    = 3
	minTrack  = 10

	// Width breakpoints, measured on the terminal, not on the panel.
	twoColMin = 85
	barsMin   = 60
	// Below minRows the panel is dropped for the ribbon alone.
	minRows = 8

	// Chrome the repo panel costs on top of its rows.
	repoChrome = 2 + 1 + 1 // border, heading, column header
)

// termWidth is the terminal width, falling back to the 80-column floor the
// requirements guarantee.
func (m Model) termWidth() int {
	if m.width < 1 {
		return defaultWidth
	}
	return m.width
}

// termHeight is the terminal height, with the same 24-row fallback.
func (m Model) termHeight() int {
	if m.height < 1 {
		return defaultHeight
	}
	return m.height
}

func (m Model) panelWidth() int {
	if w := m.termWidth(); w < panelMax {
		return w
	}
	return panelMax
}

// innerWidth is the usable content width inside the panel border and padding.
func (m Model) innerWidth() int {
	n := m.panelWidth() - panelChrome
	if n < 1 {
		return 1
	}
	return n
}

func (m Model) panel(body string) string {
	inner := m.innerWidth()
	return panelStyle.Width(inner + 2*panelPadding).Render(body)
}

func (m Model) View() string {
	w := m.termWidth()
	// Too narrow for a border: the ribbon still fits any width, and the
	// loading and error states degrade to bare text.
	if w < panelChrome+4 {
		if m.state == stateReady {
			return ribbon(m.snap.Languages, w)
		}
		return trunc(m.plainState(), w)
	}

	var body string
	switch m.state {
	case stateLoading:
		body = spinnerStyle.Render(trunc(fmt.Sprintf("fetching %s…", m.user), m.innerWidth()))
	case stateFailed:
		body = errStyle.Render(trunc(m.plainState(), m.innerWidth()))
	default:
		return m.readyView()
	}
	return m.panel(body) + "\n" + helpLine(m.expanded, w)
}

func (m Model) plainState() string {
	switch m.state {
	case stateLoading:
		return fmt.Sprintf("fetching %s…", m.user)
	case stateFailed:
		err := "load failed"
		if m.err != nil {
			err = m.err.Error()
		}
		return "error: " + err
	default:
		return ""
	}
}

func (m Model) readyView() string {
	inner := m.innerWidth()
	w := m.termWidth()

	// Too short for the full panel: the ribbon alone still answers the
	// at-a-glance question. How short that is depends on how many languages
	// the grid has to place, not on a fixed row count alone.
	if m.termHeight() < minRows || (!m.expanded && m.termHeight() < m.compactHeight()+1) {
		strip := ribbon(m.snap.Languages, inner)
		if m.termHeight() < 2 {
			return strip
		}
		return strip + "\n" + helpLine(m.expanded, w)
	}

	var b strings.Builder
	if m.showCompact() {
		b.WriteString(m.panel(m.compactBody(inner)))
	}
	if m.expanded {
		if m.showCompact() {
			b.WriteString("\n")
		}
		b.WriteString(m.panel(m.repoBody(inner)))
	}
	b.WriteString("\n")
	b.WriteString(helpLine(m.expanded, w))
	return b.String()
}

func (m Model) compactBody(inner int) string {
	rows := []string{
		m.headerRow(inner),
		ruleStyle.Render(strings.Repeat("─", inner)),
		ribbon(m.snap.Languages, inner),
		"",
	}
	rows = append(rows, m.langRows(inner)...)
	if m.snap.Skipped > 0 {
		note := fmt.Sprintf("%d repos omitted (no language data)", m.snap.Skipped)
		rows = append(rows, "", mutedStyle.Render(trunc(note, inner)))
	}
	return strings.Join(rows, "\n")
}

// headerRow is the app name and handle on the left, the counts on the right,
// padded apart to exactly inner columns.
func (m Model) headerRow(inner int) string {
	user := m.snap.User
	if user == "" {
		user = m.user
	}
	src := "live"
	if m.snap.Cached {
		src = "cached"
	}
	right := fmt.Sprintf("%d repos · %d stars · %d langs · %s",
		m.snap.ReposCount, m.snap.Stars, len(m.snap.Languages), src)

	leftPlain := "gitview  ·  @" + user
	if lipgloss.Width(leftPlain)+1+lipgloss.Width(right) > inner {
		right = fmt.Sprintf("%d repos · %d langs", m.snap.ReposCount, len(m.snap.Languages))
	}
	if lipgloss.Width(leftPlain)+1+lipgloss.Width(right) > inner {
		right = ""
	}

	left := appNameStyle.Render("gitview") + dotStyle.Render("  ·  ") + handleStyle.Render("@"+user)
	gap := inner - lipgloss.Width(leftPlain) - lipgloss.Width(right)
	if gap < 1 {
		return trunc(leftPlain, inner)
	}
	return left + strings.Repeat(" ", gap) + countStyle.Render(right)
}

// langRows lays the languages out as one or two columns of
// `name │ bar │ percent`, or as a bare percentage list on a narrow terminal.
func (m Model) langRows(inner int) []string {
	langs := m.snap.Languages
	if len(langs) == 0 {
		return []string{mutedStyle.Render("no language data")}
	}

	if m.termWidth() < barsMin {
		out := make([]string, 0, len(langs))
		for _, l := range langs {
			name := displayName(l.Name)
			pct := fmt.Sprintf("%.1f", l.Percent)
			gap := inner - lipgloss.Width(name) - lipgloss.Width(pct)
			if gap < 1 {
				gap = 1
			}
			out = append(out, styleFor(l.Name).Render(name)+strings.Repeat(" ", gap)+pctStyle.Render(pct))
		}
		return out
	}

	if m.termWidth() >= twoColMin {
		return gridRows(langs, trackWidth((inner-colGap)/2), 2, inner)
	}
	// Single column: the track takes whatever the name and percentage columns
	// leave, so the row fills the panel instead of trailing off into padding.
	return gridRows(langs, trackWidth(inner), 1, inner)
}

// trackWidth is what is left of a cell once the name, gutter and percentage
// have taken their fixed columns.
func trackWidth(cell int) int {
	n := cell - (nameW + barGutter + pctW)
	if n < minTrack {
		return minTrack
	}
	return n
}

// gridRows fills cols columns row-wise: with two columns the order is
// 1|2 / 3|4, matching the spec's Rust|JavaScript, Go|C, other|.
func gridRows(langs []stats.Language, track, cols, inner int) []string {
	cellW := nameW + track + barGutter + pctW
	var out []string
	for i := 0; i < len(langs); i += cols {
		cells := make([]string, 0, cols)
		for c := 0; c < cols; c++ {
			if i+c < len(langs) {
				cells = append(cells, langCell(langs[i+c], track))
				continue
			}
			cells = append(cells, strings.Repeat(" ", cellW))
		}
		out = append(out, trimToWidth(strings.Join(cells, strings.Repeat(" ", colGap)), inner))
	}
	return out
}

func langCell(l stats.Language, track int) string {
	name := displayName(l.Name)
	name = pad(name, nameW)
	pct := fmt.Sprintf("%*.1f", pctW, l.Percent)
	return styleFor(l.Name).Render(name) +
		bar(l.Percent, track, fillStyle(l.Name), trackStyle) +
		strings.Repeat(" ", barGutter) +
		pctStyle.Render(pct)
}

// displayName lowercases the folded bucket so it reads as a group, not a
// language, and keeps every real language's own spelling.
func displayName(name string) string {
	if name == otherBucket {
		return "other"
	}
	return name
}

func styleFor(name string) lipgloss.Style {
	if name == otherBucket {
		return mutedStyle
	}
	return nameStyle
}

// repoBody is the expanded view's second panel: the ADR 0005 repo list.
func (m Model) repoBody(inner int) string {
	repos := m.listed()
	colW := inner - 2 - 1 - 5 - 1 - 8 - 1 - 10
	if colW < 4 {
		colW = 4
	}

	rows := []string{
		headingStyle.Render(trunc("Recently updated", inner)),
		colStyle.Render(trimToWidth(
			fmt.Sprintf("  %s %5s %s %s", pad("NAME", colW), "STARS", pad("LANG", 8), "UPDATED"), inner)),
	}

	vis := m.repoRows()
	end := m.offset + vis
	if end > len(repos) {
		end = len(repos)
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
		line := fmt.Sprintf("%s%s %s %s %s",
			mark, pad(r.Name, colW), stars, pad(r.Language, 8), r.UpdatedAt)
		line = trimToWidth(line, inner)
		if i == m.cursor {
			line = selStyle.Render(line)
		}
		rows = append(rows, line)
	}
	return strings.Join(rows, "\n")
}

// trimToWidth cuts a plain (unstyled) row to n cells, measuring display width.
func trimToWidth(s string, n int) string {
	if lipgloss.Width(s) <= n {
		return s
	}
	return trunc(s, n)
}
