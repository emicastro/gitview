# gitview

Go CLI + TUI that shows a GitHub user's repositories in the terminal (private included when you view your own login with a token that can read them).

Invocation:

```text
gitview [flags] <user>
```

Example:

```text
gitview -top 8 golang
gitview -json -forks octocat
```

Flags go **before** the user. Go's `flag` package stops parsing options at the first argument that does not start with `-`.

---

## Goal

Given a GitHub username, aggregate code bytes by language across their repositories and display them in a TUI (or JSON).

Language data is **not** the repo's `primaryLanguage`. Bytes returned by each included repository's languages endpoint are summed.

---

## MVP scope

In:

- Header: user, number of repos counted, total stars of those repos.
- Languages: top N by bytes + `Other` bucket. Bars + percentage.
- Repo list: name, stars, primary language, update date.
- Flags, on-disk cache, required `GITHUB_TOKEN` for live fetch, clear errors.
- `--json` mode for scripts (no TUI).

Out:

- Unauthenticated GitHub REST (no token).
- Contributions calendar.
- PRs, issues, orgs, followers.
- ASCII pie chart.
- Search, mouse, themes, self-update.
- Including organization repos of the user (only repos listed under that login).

---

## CLI

| Flag | Default | Meaning |
|---|---|---|
| `-top N` | `8` | How many languages to show before `Other`. Must be `>= 1`. |
| `-json` | `false` | Print the result to stdout as JSON. Do not open the TUI. |
| `-forks` | `false` | Include forks. Forks are excluded by default. |
| `-fresh` | `false` | Ignore cache and refetch. |
| `-version` | — | Print `gitview 0.1.0` and exit 0. |

Positional:

- `<user>` required. If missing: usage error, exit 2.

Auth (see `docs/adr/0001-github-token-required.md`):

- Live GitHub calls require `GITHUB_TOKEN` in the environment. No token flag, no token file.
- Missing token on a live fetch: `GITHUB_TOKEN required`, exit 2.
- `-h`, `-version`, and a valid cache hit do not need a token.
- The token is never printed in logs, JSON, cache, or the TUI.

---

## Data

Source: GitHub REST API.

1. List the user's repos, paginated (`per_page=100`).
2. For each included repo, `GET /repos/{owner}/{repo}/languages`.
3. Sum bytes by language name.

Inclusion rules:

- Only repos of the requested user.
- If the token's `/user` login matches `<user>`, include private owned repos (`GET /user/repos`). Otherwise public only (`GET /users/{user}/repos`). See `docs/adr/0004-authenticated-private-repos.md`.
- Without `-forks`, drop `fork == true`.
- Drop repos with no languages map (empty or a one-off error).
- A failure on a repo's `/languages` does not abort the rest: skip it and warn on stderr.

Aggregation:

- `map[string]int64` bytes per language.
- `TopN(n)`: N largest + `Other` with the remainder.
- Percentage = `bytes_lang / bytes_total * 100`.
- Repos sorted by stars desc, tie-break by name asc.
- Total stars = sum of `stargazers_count` of included repos.

HTTP:

- Timeout 10s.
- User-Agent: `gitview/0.1`.
- Nonexistent user (404): `user not found: <user>`, exit 1.
- Rate limit (403): message with reset if the header is present, exit 1.
- Pool of at most 4 parallel requests to `/languages`.

---

## Cache

- Path: `$XDG_CACHE_HOME/gitview/<user>.json`, or `~/.cache/gitview/<user>.json` if XDG is unset.
- TTL: 1 hour.
- Store `fetched_at` alongside the stats.
- `-fresh` ignores cache.
- Cache hit: no network.

---

## Output

### Text / TUI

`-json` off:

- While loading: spinner `fetching <user>…`.
- List view:
  - Line 1: `@user  repos=N  stars=S  cached|live`
  - Languages block: fixed-width bar + percentage.
  - Repos block: truncated name, stars, primary language, `updated` (`YYYY-MM-DD`).
  - Footer: `q quit  r refresh`.
- `j/k` or arrows: move selection in the list. No detail panel.
- `r`: refetch with `-fresh` semantics.
- `q` / Ctrl+C: restore the terminal and exit 0.
- Fetch error: shown in the TUI; the spinner must not hang.
- Resize must not crash. Must be usable at 80×24.

### JSON

`-json` on: stdout only, no TUI. Stable shape:

```json
{
  "user": "octocat",
  "fetched_at": "2026-09-17T15:00:00Z",
  "cached": false,
  "repos_count": 12,
  "stars": 340,
  "languages": [
    {"name": "Go", "bytes": 12000, "percent": 61.2},
    {"name": "Other", "bytes": 1000, "percent": 5.1}
  ],
  "repos": [
    {
      "name": "hello",
      "stars": 10,
      "language": "Go",
      "updated_at": "2026-01-02",
      "fork": false
    }
  ]
}
```

---

## Exit codes

| Code | When |
|---|---|
| 0 | OK, or `-h` / `-version` |
| 1 | Runtime error (network, 404, rate limit, unrecoverable corrupt cache) |
| 2 | Invalid usage (no user, `-top` < 1, unknown flag, missing `GITHUB_TOKEN` on live fetch) |

`-h` is handled by `flag` and exits 0.

---

## Packages

```text
gitview/
  main.go                 # flags + dispatch json vs tui
  internal/github/        # HTTP, pagination, API errors
  internal/stats/         # aggregate, TopN, sort
  internal/cache/         # disk
  internal/render/        # text + json
  internal/ui/            # TUI
```

`internal/` is not importable from outside the module. The domain (`stats`) does not import TUI or HTTP.

---

## Delivery slices

Each slice is demoable on its own. Domain tests with no network.

### Slice 0 — CLI

Binary that parses args and prints config. No network.

- `user` required.
- Flags `-top`, `-json`, `-forks`.
- Exit 2 if user is missing or `-top < 1`.
- Table-driven tests of `parseArgs`.

### Slice 1 — Fetch + stdout

No TUI. Text or JSON with real numbers.

- List repos + languages.
- Aggregate + TopN.
- Text renderer (bar ~20 chars) and JSON.
- Tests of `Aggregate` / `TopN` with fixtures.

### Slice 2 — Cache and rate limit

- Cache TTL 1h + `-fresh`.
- Pool of 4.
- 403 / 404 messages.
- Skip a repo if `/languages` fails.
- Cache tests with a temp dir.

### Slice 3 — Minimal TUI

- Bubble Tea + Lip Gloss.
- States `loading | ready | failed`.
- Same `Load()` as JSON mode.
- `-json` does not open the TUI.
- Keys `q` and `r`.

### Slice 4 — Usable list

- Sorted table, name truncated to width.
- Scroll / viewport.
- Selection `j/k`. No extra fetch on select.

### Slice 5 — Wrap-up

- `-version`.
- README: build, token, flags, limits.
- `go test ./...` green with no network.
- Manual check: large user, 404, own user.

---

## Done when

- `go test ./...` passes offline.
- `gitview -json <user>` prints the schema above.
- `gitview <user>` opens a TUI usable at 80×24.
- A second run of the same user within <1h does not hit the network.
- A missing or invalid token is not leaked to output.
- No panic on a nonexistent user, downed network, or bad token.
