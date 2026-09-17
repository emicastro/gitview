package ui

import "strings"

// binding is one entry in the help line. The same list drives the collapsed
// and expanded forms so the two cannot drift apart.
type binding struct {
	key   string
	label string
	// expandedLabel replaces label while the repo list is open.
	expandedLabel string
	// onlyExpanded hides the entry while the repo list is collapsed.
	onlyExpanded bool
}

var bindings = []binding{
	{key: "tab", label: "expand", expandedLabel: "collapse"},
	{key: "j/k", label: "select", onlyExpanded: true},
	{key: "r", label: "refresh"},
	{key: "q", label: "quit"},
}

func visibleBindings(expanded bool) []binding {
	out := make([]binding, 0, len(bindings))
	for _, b := range bindings {
		if b.onlyExpanded && !expanded {
			continue
		}
		if expanded && b.expandedLabel != "" {
			b.label = b.expandedLabel
		}
		out = append(out, b)
	}
	return out
}

// helpLine renders the key hints below the panel: keys in lavender, labels
// dim. It gives up the gaps, then the labels, rather than overflowing the
// terminal and wrapping onto a line the layout has not budgeted for.
func helpLine(expanded bool, width int) string {
	for _, f := range []struct {
		sep    string
		labels bool
	}{
		{sep: "   ", labels: true},
		{sep: "  ", labels: true},
		{sep: "  ", labels: false},
	} {
		if plainHelp(expanded, f.sep, f.labels) == trunc(plainHelp(expanded, f.sep, f.labels), width) {
			return renderHelp(expanded, f.sep, f.labels)
		}
	}
	return helpStyle.Render(trunc(plainHelp(expanded, " ", false), width))
}

func renderHelp(expanded bool, sep string, labels bool) string {
	parts := make([]string, 0, len(bindings))
	for _, b := range visibleBindings(expanded) {
		s := keyStyle.Render(b.key)
		if labels {
			s += helpStyle.Render(" " + b.label)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, helpStyle.Render(sep))
}

// plainHelp is the same line without styling, for width budgeting.
func plainHelp(expanded bool, sep string, labels bool) string {
	parts := make([]string, 0, len(bindings))
	for _, b := range visibleBindings(expanded) {
		s := b.key
		if labels {
			s += " " + b.label
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, sep)
}
