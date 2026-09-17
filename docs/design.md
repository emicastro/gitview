# Design

Status: accepted
Date: 2026-09-17
Source of truth for requirements: `docs/requirements.md`.

Cites `docs/adr/0001-github-token-required.md`, `docs/adr/0002-stdlib-http.md`, `docs/adr/0003-bubbletea-tui.md`, `docs/adr/0004-authenticated-private-repos.md`.

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
  internal/ui/            # Bubble Tea  (ADR 0003)
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
6. `stats`: sum bytes by language; `TopN(n)` + `Other`; repos by stars desc, name asc; total stars = sum of included `stargazers_count`.
7. Write cache: same snapshot as JSON stdout. **No token field.**

Errors: 404 → `user not found: <user>`, exit 1. 401 → `github authentication failed`, exit 1. 403 → rate-limit message including reset if `X-RateLimit-Reset` is present, exit 1. Corrupt cache that cannot be ignored → exit 1. TUI shows fetch errors in-view; spinner must not hang.

Tests inject the API base URL (production default `https://api.github.com`). No test dials GitHub. Fake tokens via `t.Setenv` only.

## Render

JSON: exact schema in requirements, stdout only, no TUI (ADR 0003).

Text bars (non-TUI path used by tests and as a building block): ~20 character bar + percent.

TUI (ADR 0003): states `loading | ready | failed`; spinner `fetching <user>…`; header `@user  repos=N  stars=S  cached|live`; language bars; repo list (truncated name, stars, primary language, `updated` YYYY-MM-DD); footer `q quit  r refresh`; `j/k` or arrows move selection; `r` reloads with `-fresh` semantics; `q`/Ctrl+C restore terminal, exit 0. Usable at 80×24; resize must not panic.

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
