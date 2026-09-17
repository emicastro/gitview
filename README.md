# gitview

Terminal view of a GitHub user's public repositories: language bytes (summed from each repo's languages API, not `primaryLanguage`), stars, and a small TUI or JSON.

## Build

Needs Go 1.22 or newer (this module uses the version in `go.mod`).

```sh
go build -o gitview .
```

## Token

Live fetches require a GitHub token in the environment. Do not pass it as a flag, do not put it in the repo, and do not write it to `.env` in this tree (`.env` is gitignored).

```sh
export GITHUB_TOKEN
# then paste the value into your shell, or:
export GITHUB_TOKEN="$(cat /path/to/local/token-file)"
```

Public repos of any user work with a token that can read public data.

**Your own private repos** appear only when `<user>` is your GitHub login **and** the token can read private repos:

- Classic PAT: enable the `repo` scope (not public_repo only).
- Fine-grained PAT: resource owner = you, repository access = **All repositories** (or the private ones you care about).

`GET /users/{you}/repos` is public-only even with a powerful token; gitview uses `GET /user/repos` when the login matches. Organization repos are still excluded.

The token is never printed in logs, JSON, the cache file, or the TUI. If you ran gitview before private-repo support, pass `-fresh` (or wait an hour) so the cache is not reused.

`-h`, `-version`, and a cache hit less than one hour old do not need a token.

## Usage

```text
gitview [flags] <user>
```

Flags go before `<user>`.

| Flag | Default | Meaning |
|---|---|---|
| `-top N` | `8` | Languages to show before `Other` (`>= 1`) |
| `-json` | off | Print JSON to stdout; do not open the TUI |
| `-forks` | off | Include forked repos |
| `-fresh` | off | Ignore cache and refetch |
| `-all` | off | List every included repo (by last update). Default is 5. |
| `-version` | | Print `gitview 0.1.0` and exit 0 |

```sh
gitview -json golang
gitview -top 5 octocat
gitview -all emicastro
```

Default view is one compact panel: a header line, a full-width ribbon segmented
by language share, and a grid of `name | bar | percent` rows (HTML and CSS
omitted). It is two columns at 85 columns or wider, one column below that, and
percentages only under 60 columns. The panel never grows past 85 columns, so the
bars stay comparable between runs.

```text
╭─────────────────────────────────────────────────────────────────────────────╮
│  gitview  ·  @emicastro            87 repos · 0 stars · 9 langs · cached     │
│  ─────────────────────────────────────────────────────────────────────────  │
│  █████████████████████████████████████████████████████████████████████████   │
│                                                                             │
│  Rust       ███████▎░░░░░░░░░░░  38.0   Go         ███▋░░░░░░░░░░░░░░  18.8  │
│  JavaScript ██▍░░░░░░░░░░░░░░░░  12.4   Lua        ██░░░░░░░░░░░░░░░░  10.4  │
│  other      █▎░░░░░░░░░░░░░░░░░   6.4                                        │
╰─────────────────────────────────────────────────────────────────────────────╯
  tab expand   r refresh   q quit
```

Press `tab` for **Recently updated** — five repos by GitHub `updated_at` in a
second panel, or the full list with `-all`.

TUI keys: `tab` expand or collapse the repo list, `j`/`k` or arrows move the
selection in it, `r` refresh, `q` or Ctrl+C quit.

Colors are Catppuccin: Mocha on a dark terminal, Latte on a light one.

## Cache and limits

- Cache: `$XDG_CACHE_HOME/gitview/<user>.json` or `~/.cache/gitview/<user>.json`, TTL 1 hour.
- Only repos listed under that login (not organization repos).
- GitHub rate limits still apply with a token (unauthenticated mode is not supported).
- A missing user, bad token, or rate limit is an error (exit 1). Missing `<user>` or missing `GITHUB_TOKEN` on a live fetch is usage (exit 2).

## Tests

Offline; they do not call GitHub.

```sh
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
```
