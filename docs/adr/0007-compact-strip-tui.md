# 0007. Compact strip TUI

Status: accepted
Date: 2026-09-17

## Context

The TUI was a flat, unbordered stack of lines: header, language bars, rule, `Recently updated`, footer. It has no visual hierarchy, no at-a-glance summary of the whole account, and its colors are GitHub-linguist hexes inlined in `internal/ui/ui.go`. `AGENTS.md` requires every color to go through `internal/ui/theme.go`, which does not exist. `docs/ui/tui-spec.md` and `docs/ui/layout.html` specify a compact bordered panel in Catppuccin Mocha: header row, full-width language ribbon, two-column mini-bar grid, help line below.

## Options

- **Option A** — Bordered compact strip panel as the default view, with `tab` expanding to a second panel holding the ADR 0005 repo list. Palette and language accents behind `theme.go`; bar geometry behind `bars.go`.
- **Option B** — Keep the flat layout, only move the colors into `theme.go`.
- **Option C** — Full-screen multi-pane dashboard with the repo list always visible beside the bars.

## Decision

Option A. The ribbon answers "what is this account written in" in one row, and the grid is the legend for it; the repo list is a second question and earns a keystroke rather than permanent screen space. Option C does not fit the 80x24 floor that requirements mandate.

## Consequences

Extends ADR 0003: Lip Gloss styling is no longer "minimal" and a palette now exists, so a Latte/light variant is a one-file change. The ADR 0005 repo list moves behind `tab` with no change to what data it shows or to JSON. The spec's `p` period key is not adopted — the languages API has no time dimension. The help line is hand-rendered from a local key map rather than `bubbles/help`, so no new module dependency. Panel width caps at 85 columns (the spec's natural width) and is left-aligned, so bars stay comparable between runs.
