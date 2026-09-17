package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"emicastro.com/gitview/internal/stats"
)

// blocks indexes eighth-width block glyphs by eighths filled, so a short track
// can still express a single percentage point.
var blocks = []string{" ", "▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}

const (
	fullBlock  = "█"
	emptyBlock = "░"
)

// bar renders pct (0..100) across cells columns. The partial glyph is painted
// in the fill style, not the track style: a ▎ in the track color reads as a
// gap rather than as progress.
func bar(pct float64, cells int, fill, track lipgloss.Style) string {
	if cells <= 0 {
		return ""
	}
	if pct < 0 {
		pct = 0
	}

	exact := pct / 100 * float64(cells)
	full := int(exact)
	eighths := int(math.Round((exact - float64(full)) * 8))
	if eighths == 8 {
		full++
		eighths = 0
	}
	if full > cells {
		full, eighths = cells, 0
	}

	var b strings.Builder
	b.WriteString(fill.Render(strings.Repeat(fullBlock, full)))
	used := full
	if eighths > 0 && used < cells {
		b.WriteString(fill.Render(blocks[eighths]))
		used++
	}
	b.WriteString(track.Render(strings.Repeat(emptyBlock, cells-used)))
	return b.String()
}

// ribbon renders every language as one solid run across cells columns. The
// segments must sum to exactly cells or the panel's right border wobbles, so
// the rounding remainder goes to the largest segment.
func ribbon(langs []stats.Language, cells int) string {
	if cells <= 0 {
		return ""
	}
	if len(langs) == 0 {
		return trackStyle.Render(strings.Repeat(emptyBlock, cells))
	}

	widths := make([]int, len(langs))
	total, largest := 0, 0
	for i, l := range langs {
		pct := l.Percent
		if pct < 0 {
			pct = 0
		}
		widths[i] = int(pct / 100 * float64(cells))
		total += widths[i]
		if langs[i].Percent > langs[largest].Percent {
			largest = i
		}
	}
	if total > cells {
		// Shave the overshoot off the largest segment rather than the tail.
		widths[largest] -= total - cells
		if widths[largest] < 0 {
			widths[largest] = 0
		}
		total = 0
		for _, w := range widths {
			total += w
		}
	}

	var b strings.Builder
	for i, l := range langs {
		w := widths[i]
		if i == largest {
			w += cells - total
		}
		if w <= 0 {
			continue
		}
		b.WriteString(fillStyle(l.Name).Render(strings.Repeat(fullBlock, w)))
	}
	return b.String()
}
