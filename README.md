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

A classic PAT or a fine-grained token that can read public repositories is enough. The token is never printed in logs, JSON, the cache file, or the TUI.

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
| `-version` | | Print `gitview 0.1.0` and exit 0 |

```sh
gitview -json golang
gitview -top 5 octocat
```

TUI keys: `j`/`k` or arrows move, `r` refresh, `q` or Ctrl+C quit.

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
