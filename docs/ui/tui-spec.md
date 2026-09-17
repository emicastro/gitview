# langstat — TUI spec (compact strip layout)

Terminal UI showing the language breakdown of a GitHub account.
Go + Bubble Tea + Lip Gloss. Palette: Catppuccin Mocha.

## Layout

Everything lives in one rounded-border panel, with a help line below it.

```
╭──────────────────────────────────────────────────────────────────────╮
│ langstat  ·  @handle                             12 repos · 9 langs  │
│ ──────────────────────────────────────────────────────────────────── │
│ ████████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ │
│                                                                      │
│ Rust        ██████▎         34.0   JavaScript  ██▏            10.0   │
│ Go          ████            20.0   C           ██▏            10.0   │
│ other       █████▏          26.0                                     │
╰──────────────────────────────────────────────────────────────────────╯
  tab expand   p period 12 months   r refresh   q quit
```

Vertical order inside the panel:

1. **Header row.** Left: app name in mauve bold, a middot separator in
   surface1, then the handle in subtext0. Right: repo count and language
   count in overlay2. One blank line of separation via a horizontal rule.
2. **Rule.** A full-width run of `─` in surface0.
3. **Ribbon.** One row, full panel width, segmented by share. This is the
   at-a-glance summary — no legend of its own, the grid below serves that
   role.
4. **Blank row.**
5. **Mini-bar grid.** Two columns, filled row-wise: Rust | JavaScript,
   Go | C, other | (empty). Each cell is `name │ bar │ pct`.

Column widths inside a grid cell:

| part | width | alignment |
|---|---|---|
| name | 11 cells | left |
| bar track | 20 cells | left |
| gutter | 2 cells | — |
| percentage | 6 cells | right |

Gutter between the two grid columns: 3 cells. Panel horizontal padding:
2 cells each side. So the natural width is
`2 + (11+20+2+6)*2 + 3 + 2 = 85` columns including borders.

## Colors (Catppuccin Mocha)

Define these as `lipgloss.AdaptiveColor` with Latte values as the light
counterpart so the TUI doesn't glow on a light terminal.

| role | hex | token |
|---|---|---|
| panel background | `#1e1e2e` | base |
| header strip | `#181825` | mantle |
| empty bar track | `#313244` | surface0 |
| border, rules | `#45475a` | surface1 |
| panel border (focused) | `#585b70` | surface2 |
| dim text, help | `#7f849c` | overlay1 |
| secondary text | `#9399b2` | overlay2 |
| body text | `#a6adc8` | subtext0 |
| primary text | `#cdd6f4` | text |
| app name | `#cba6f7` | mauve |
| key hints | `#b4befe` | lavender |

Language accents — assign from a fixed map, fall back to a rotating list
for anything unmapped:

| language | hex | token |
|---|---|---|
| Rust | `#fab387` | peach |
| Go | `#89dceb` | sky |
| JavaScript | `#f9e2af` | yellow |
| TypeScript | `#89b4fa` | blue |
| C | `#b4befe` | lavender |
| Python | `#a6e3a1` | green |
| Ruby | `#f38ba8` | red |
| Shell | `#94e2d5` | teal |
| other | `#6c7086` | overlay0 |

`other` is always the neutral grey and always sorts last, regardless of
its share.

## Bar rendering

Use eighth-width block glyphs so a 20-cell track can still express a
single percentage point.

```go
var blocks = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

// bar renders pct (0..100) across `cells` columns.
func bar(pct float64, cells int, fill, track lipgloss.Style) string {
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
    b.WriteString(fill.Render(strings.Repeat("█", full)))
    used := full
    if eighths > 0 && used < cells {
        b.WriteString(fill.Render(string(blocks[eighths])))
        used++
    }
    b.WriteString(track.Render(strings.Repeat("░", cells-used)))
    return b.String()
}
```

Two things that bite:

- The partial glyph must be rendered in the **fill** color on the **track**
  background, not the reverse. A `▎` painted in the track color reads as a
  gap.
- `strings.Repeat` counts bytes, not cells. These glyphs are multi-byte but
  single-width, so repeat the *string* not the rune count, and measure
  finished rows with `lipgloss.Width`, never `len`.

The ribbon uses the same function but with the full panel width, and it
must sum to exactly that width — accumulate the rounded cell counts and
give the rounding remainder to the largest segment, otherwise the ribbon
comes up a cell short and the border wobbles.

## Sorting and grouping

- Sort by share descending. Ties break alphabetically.
- Fold every language under 5% into `other`. Without this you get a dozen
  one-cell bars and the palette runs out of distinguishable hues.
- If the fold leaves `other` at 0%, drop the row entirely rather than
  rendering an empty bar.

## Responsive behaviour

Recompute on every `tea.WindowSizeMsg` — nothing below should be constant.

- **≥ 85 columns:** the two-column grid above.
- **60–84 columns:** single column, bar track shrinks to
  `width - nameWidth - pctWidth - gutters - padding`, minimum 10 cells.
- **< 60 columns:** ribbon plus a percentage-only list, no bars.
- **< 8 rows:** ribbon and help line only.

## Keys

| key | action |
|---|---|
| `tab` | expand to the full dashboard view |
| `p` | cycle period: all time → 12 months → 90 days |
| `r` | refetch from the GitHub API |
| `q`, `ctrl+c` | quit |

Build the help line with `bubbles/help` and a `key.Map` rather than
hand-rendering it, so the short and full forms stay in sync.

## Data

Source is the GitHub REST `/repos/{owner}/{repo}/languages` endpoint per
repo, which returns bytes per language. Sum across repos, then convert to
percentages. Notes:

- Exclude forks and archived repos by default; make it a flag.
- Byte counts are heavily skewed by vendored and generated files. Consider
  honouring `.gitattributes` `linguist-vendored` if you want the numbers to
  match GitHub's own bars.
- Cache the per-repo response on disk keyed by the repo's `pushed_at`, so
  `r` only refetches what actually moved.

## Structure

```
cmd/langstat/main.go
internal/github/     client, language aggregation, cache
internal/ui/
    model.go         tea.Model, Update, Init
    view.go          View, layout selection by width
    bars.go          bar(), ribbon()
    theme.go         palette, language color map, lipgloss styles
    keys.go          key.Map + help bindings
```

Keep `theme.go` free of layout and `bars.go` free of styling decisions
beyond the two styles passed in — that's what makes a Latte variant or an
alternate palette a one-file change.
