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
