# 0006. Exclude HTML and CSS from language bars

Status: accepted
Date: 2026-09-17

## Context

HTML and CSS often dominate byte counts and drown languages people care about. Folding them into `Other` would still inflate that bucket.

## Options

- **Option A** — Drop GitHub names `HTML` and `CSS` before TopN. Bytes are not in the total and not in `Other`.
- **Option B** — Fold HTML/CSS into `Other`.
- **Option C** — Also drop SCSS, Less, and similar markup.

## Decision

Option A. Exact names `HTML` and `CSS` only. Percents are among remaining languages. Repos that are only HTML/CSS stay in the repo list.

## Consequences

Frontend-heavy totals look like the non-markup stack. SCSS/Less still count. JSON `languages` matches the TUI bars.
