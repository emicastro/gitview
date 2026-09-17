# Design

Status: accepted
Date: 2026-09-17
Source of truth for requirements: `docs/requirements.md`.

Cites `docs/adr/0001-github-token-required.md`, `docs/adr/0002-stdlib-http.md`, `docs/adr/0003-bubbletea-tui.md`, `docs/adr/0004-authenticated-private-repos.md`, `docs/adr/0005-recent-repos-default.md`, `docs/adr/0006-exclude-html-css.md`, `docs/adr/0007-compact-strip-tui.md`.

## Process

`main` installs `signal.NotifyContext` for SIGINT/SIGTERM and exits with `run(...)`.

```text
run(ctx, args, getenv, stdout, stderr) int
```

`parseArgs` uses a private `flag.FlagSet` (`ContinueOnError`, usage to `stderr`) and **returns** errors. It does not call `os.Exit`. `-h` is `flag.ErrHelp` → exit 0. `-version` prints `gitview 0.1.0` on stdout and exits 0 without a user or a token.

Stdout is the program result (JSON or TUI). Stderr is diagnostics (warnings, usage, errors).

## Packages

```text
gitview/
  main.go                 # flags, env, exit codes, json vs TUI
  internal/github/        # REST, pagination, API errors  (ADR 0002)
  internal/stats/         # aggregate, TopN, sort
  internal/cache/         # disk snapshot, TTL 1h
  internal/render/        # JSON + text bars
  internal/ui/            # Bubble Tea  (ADR 0003, ADR 0007)
      ui.go               # tea.Model, Init, Update, scrolling
      view.go             # View, layout selection by width and height
      bars.go             # bar(), ribbon() -- geometry, no styling decisions
      theme.go            # palette, language accents, styles (AGENTS.md)
      keys.go             # key map + help line
```

`internal/stats` imports neither HTTP nor TUI.

## Auth

See ADR 0001. Live GitHub calls require `getenv("GITHUB_TOKEN")` non-empty. Missing: exit 2, stderr `GITHUB_TOKEN required`. The value is passed into the client as a function argument / request header only. It is not a field on any type that is JSON-encoded, logged, or printed.

## Load

Shared by `-json` and the TUI:

1. Unless `-fresh`, try cache for `<user>` (`$XDG_CACHE_HOME/gitview/<user>.json` or `~/.cache/gitview/<user>.json`). Username is a single path segment; `/` and `..` are rejected. Hit if file exists, parses, and `fetched_at` is within 1 hour → `cached=true`, no network, no token.
2. Else require token (ADR 0001).
3. Client (ADR 0002, ADR 0004): `GET /user`. If that login matches `<user>`, `GET /user/repos?per_page=100&affiliation=owner`; else `GET /users/{user}/repos?per_page=100&type=owner`. Paginate `Link`. Timeout 10s. `User-Agent: gitview/0.1`. `Authorization: Bearer <token>`.
4. Drop `fork==true` unless `-forks`.
5. At most 4 concurrent `GET /repos/{owner}/{repo}/languages`. One-off failure: skip repo, warn on stderr, continue.
6. `stats`: sum bytes; drop `HTML`/`CSS` (ADR 0006); `TopN(n)` + `Other`; repos by `updated_at` desc, name asc; total stars = sum of included `stargazers_count`. Snapshot.Repos is the **full** sorted list (ADR 0005).
7. Write cache: full snapshot (all repos). **No token field.** JSON/TUI display 5 repos unless `-all`.

Errors: 404 → `user not found: <user>`, exit 1. 401 → `github authentication failed`, exit 1. 403 → rate-limit message including reset if `X-RateLimit-Reset` is present, exit 1. Corrupt cache that cannot be ignored → exit 1. TUI shows fetch errors in-view; spinner must not hang.

Tests inject the API base URL (production default `https://api.github.com`). No test dials GitHub. Fake tokens via `t.Setenv` only.

## Render

JSON: exact schema in requirements, stdout only, no TUI (ADR 0003).

Text bars (non-TUI path used by tests and as a building block): ~20 character bar + percent.

TUI (ADR 0003, 0005, 0007): states `loading | ready | failed`; spinner `fetching <user>…`.

Compact view (default) is one bordered panel, width `min(terminal, 85)`, left-aligned, 2 cells of horizontal padding: header row (`gitview · @user` left, `N repos · S stars · N langs · cached|live` right), rule, full-width ribbon segmented by share, blank row, mini-bar grid (`name 11 | track 20 | gutter 2 | pct 6`, columns separated by 3), then the skipped-repos note if any. Help line renders below the panel.

Expanded view (`tab`) appends a second panel: heading `Recently updated`, columns NAME / STARS / LANG / UPDATED, `> ` cursor, em dash for zero stars, 5 repos or all if `-all`. `j/k` and arrows move selection there and are inert while collapsed; no fetch on select.

Width breakpoints, recomputed on every `tea.WindowSizeMsg`: `>= 85` two-column grid; `60..84` single column with the track shrunk to fit, floor 10; `< 60` ribbon plus a name-and-percentage list, no bars; `< 8` rows ribbon and help line only. `r` reloads with `-fresh` semantics; `q`/Ctrl+C restore terminal, exit 0. Usable at 80×24; resize must not panic.

All colors are Catppuccin `lipgloss.AdaptiveColor` (Mocha dark, Latte light) and live in `theme.go`; `bars.go` takes the two styles it paints with and makes no styling decisions of its own. Rows that carry ANSI are measured with `lipgloss.Width`, never `len`.

## Traceability

| Requirements “done when” | Mechanism |
|---|---|
| `go test ./...` offline | httptest + fixtures; no live API |
| `gitview -json <user>` schema | `internal/render` JSON + cache snapshot shape |
| TUI usable 80×24 | `internal/ui` + `WindowSizeMsg` test |
| Second run <1h no network | `internal/cache` TTL |
| Token not leaked | ADR 0001; no token in cache/JSON/logs |
| No panic on 404 / down / bad token | mapped errors; TUI `failed` |

## Non-goals

Unauthenticated REST (ADR 0001). Contributions calendar, PRs/issues/orgs/followers, pie chart, search/mouse/themes/self-update, org repos.
