package ui

import "github.com/charmbracelet/lipgloss"

// Catppuccin, Mocha on a dark terminal and Latte on a light one. Every color
// in the package comes from here; nothing below this file names a hex.
var (
	base     = lipgloss.AdaptiveColor{Dark: "#1e1e2e", Light: "#eff1f5"}
	surface0 = lipgloss.AdaptiveColor{Dark: "#313244", Light: "#ccd0da"}
	surface1 = lipgloss.AdaptiveColor{Dark: "#45475a", Light: "#bcc0cc"}
	surface2 = lipgloss.AdaptiveColor{Dark: "#585b70", Light: "#acb0be"}
	overlay0 = lipgloss.AdaptiveColor{Dark: "#6c7086", Light: "#9ca0b0"}
	overlay1 = lipgloss.AdaptiveColor{Dark: "#7f849c", Light: "#8c8fa1"}
	overlay2 = lipgloss.AdaptiveColor{Dark: "#9399b2", Light: "#7c7f93"}
	subtext0 = lipgloss.AdaptiveColor{Dark: "#a6adc8", Light: "#6c6f85"}
	text     = lipgloss.AdaptiveColor{Dark: "#cdd6f4", Light: "#4c4f69"}
	mauve    = lipgloss.AdaptiveColor{Dark: "#cba6f7", Light: "#8839ef"}
	lavender = lipgloss.AdaptiveColor{Dark: "#b4befe", Light: "#7287fd"}

	peach     = lipgloss.AdaptiveColor{Dark: "#fab387", Light: "#fe640b"}
	sky       = lipgloss.AdaptiveColor{Dark: "#89dceb", Light: "#04a5e5"}
	yellow    = lipgloss.AdaptiveColor{Dark: "#f9e2af", Light: "#df8e1d"}
	blue      = lipgloss.AdaptiveColor{Dark: "#89b4fa", Light: "#1e66f5"}
	green     = lipgloss.AdaptiveColor{Dark: "#a6e3a1", Light: "#40a02b"}
	red       = lipgloss.AdaptiveColor{Dark: "#f38ba8", Light: "#d20f39"}
	teal      = lipgloss.AdaptiveColor{Dark: "#94e2d5", Light: "#179299"}
	pink      = lipgloss.AdaptiveColor{Dark: "#f5c2e7", Light: "#ea76cb"}
	maroon    = lipgloss.AdaptiveColor{Dark: "#eba0ac", Light: "#e64553"}
	flamingo  = lipgloss.AdaptiveColor{Dark: "#f2cdcd", Light: "#dd7878"}
	rosewater = lipgloss.AdaptiveColor{Dark: "#f5e0dc", Light: "#dc8a78"}
	sapphire  = lipgloss.AdaptiveColor{Dark: "#74c7ec", Light: "#209fb5"}
)

// otherBucket is the name stats.TopN gives the folded remainder. It always
// renders in the neutral grey and always sorts last, whatever its share.
const otherBucket = "Other"

var langAccents = map[string]lipgloss.AdaptiveColor{
	"Rust":       peach,
	"Go":         sky,
	"JavaScript": yellow,
	"TypeScript": blue,
	"C":          lavender,
	"Python":     green,
	"Ruby":       red,
	"Shell":      teal,
	otherBucket:  overlay0,
}

// fallbackAccents covers anything not in langAccents. Indexing by a hash of
// the name keeps a language the same hue between renders and between runs.
// These hues are deliberately disjoint from langAccents: a fallback that
// reused, say, teal would render Vue and Shell as the same ribbon segment.
var fallbackAccents = []lipgloss.AdaptiveColor{pink, mauve, maroon, sapphire, flamingo, rosewater}

func langColor(name string) lipgloss.AdaptiveColor {
	if c, ok := langAccents[name]; ok {
		return c
	}
	var sum uint32
	for _, r := range name {
		sum = sum*31 + uint32(r)
	}
	return fallbackAccents[int(sum%uint32(len(fallbackAccents)))]
}

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(surface2).
			Background(base).
			Padding(0, panelPadding)

	appNameStyle = lipgloss.NewStyle().Bold(true).Foreground(mauve)
	dotStyle     = lipgloss.NewStyle().Foreground(surface1)
	handleStyle  = lipgloss.NewStyle().Foreground(subtext0)
	countStyle   = lipgloss.NewStyle().Foreground(overlay2)
	ruleStyle    = lipgloss.NewStyle().Foreground(surface0)
	trackStyle   = lipgloss.NewStyle().Foreground(surface1)
	nameStyle    = lipgloss.NewStyle().Foreground(text)
	pctStyle     = lipgloss.NewStyle().Foreground(subtext0)
	mutedStyle   = lipgloss.NewStyle().Foreground(overlay1)
	headingStyle = lipgloss.NewStyle().Bold(true).Foreground(mauve)
	colStyle     = lipgloss.NewStyle().Foreground(overlay1)
	selStyle     = lipgloss.NewStyle().Bold(true).Foreground(base).Background(lavender)
	keyStyle     = lipgloss.NewStyle().Foreground(lavender)
	helpStyle    = lipgloss.NewStyle().Foreground(overlay1)
	errStyle     = lipgloss.NewStyle().Foreground(red)
	spinnerStyle = lipgloss.NewStyle().Foreground(overlay2)
)

// fillStyle paints a language's own share of a bar or ribbon segment.
func fillStyle(lang string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(langColor(lang))
}
