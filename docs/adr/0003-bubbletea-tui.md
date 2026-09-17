# 0003. Bubble Tea TUI

Status: accepted
Date: 2026-09-17

## Context

Default (non-`-json`) output is an interactive list: loading spinner, language bars, repo list, `q`/`r`/`j`/`k`. `-json` must not enter that mode. The domain load path is shared.

## Options

- **Option A** — Bubble Tea + Lip Gloss (`loading | ready | failed` model).
- **Option B** — Text-only stdout forever; no TUI.

## Decision

Option A, matching requirements. JSON mode calls the same `Load()` and prints to stdout without starting the program.

## Consequences

TUI tests drive `Update`/`View` with messages (including `tea.WindowSizeMsg` at 80×24); they do not talk to GitHub. Lip Gloss styling stays minimal (usable at 80×24, no theme system — a requirements non-goal).
