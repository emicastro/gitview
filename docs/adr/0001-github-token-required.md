# 0001. GitHub token required

Status: accepted
Date: 2026-09-17

## Context

A full run lists every repo under the login, then calls `/languages` once per included repo. Unauthenticated GitHub REST allows 60 requests per hour, which a typical user exhausts immediately. The token is a secret: a CLI flag appears in `ps`, a config file is easy to commit, and interpolating the value into logs or errors leaks it.

## Options

- **Option A** — Optional `GITHUB_TOKEN`; fall back to unauthenticated REST (original requirements).
- **Option B** — Required `GITHUB_TOKEN` environment variable for live fetches; no flag; no token file. `-h` / `-version` and a valid cache hit are exempt.
- **Option C** — `--token` flag, with env as a fallback.

## Decision

Option B. Unauthenticated mode cannot deliver the product. A flag is visible to other processes; a file is a commit risk. The process reads `os.Getenv("GITHUB_TOKEN")` (tests: `t.Setenv`) and sends it only as `Authorization: Bearer <token>`.

## Consequences

Live fetch without the env var is a usage error (exit 2, message `GITHUB_TOKEN required`). 401 is a runtime error (exit 1, `github authentication failed`) with no secret in the string. Cache must not store the token. README documents `export GITHUB_TOKEN=…` only. `.env` is gitignored and never written by this repo with a real value. Unauthenticated REST is a non-goal.
