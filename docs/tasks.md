# Tasks

Status: accepted
Date: 2026-09-17
Each task is one Implement session and ends in a runnable check.

## Group 1 — Spec

- [x] **1.1** Write `docs/adr/0001-github-token-required.md`, `0002-stdlib-http.md`, `0003-bubbletea-tui.md` (Status: accepted, Date: 2026-09-17).
      Check: `test -f docs/adr/0001-github-token-required.md && test -f docs/adr/0002-stdlib-http.md && test -f docs/adr/0003-bubbletea-tui.md`
- [x] **1.2** Write `docs/design.md` and `docs/tasks.md` from the accepted plan; patch `docs/requirements.md` Auth/MVP to required token (cache hit and `-h`/`-version` exempt). Add `.env` to `.gitignore`.
      Check: `rg -n 'optional token|Without a token' docs/requirements.md; test $? -ne 0` and `git check-ignore -q .env`

## Group 2 — CLI binary tests (Slice 0)

- [x] **2.1** `parseArgs` returns errors (no `os.Exit`). Add `-fresh`, `-version`. Table-driven tests: missing user, `-top 0`, unknown flag, happy path, flags after user ignored by `flag`.
      Check: `go test -race -count=1 -run 'TestParseArgs' .`
- [x] **2.2** `run(...) int` wires parse → stdout config-or-dispatch later. Tests: `-h` and `-version` exit 0 with no token; missing user exit 2; missing token on a fetch path exit 2 with exact `GITHUB_TOKEN required` and no secret in stdout/stderr (`t.Setenv` empty vs dummy).
      Check: `go test -race -count=1 -run 'TestRun' .`

## Group 3 — Domain (Slice 1 stats)

- [x] **3.1** `internal/stats`: aggregate bytes, `TopN`, repo sort, percent. Fixture tests including empty, all-in-Other, ties.
      Check: `go test -race -count=1 ./internal/stats`

## Group 4 — GitHub client (Slice 1 fetch, no TUI)

- [x] **4.1** `internal/github` against `httptest`: list+paginate, languages pool of 4, 404 / 401 / 403+reset, per-repo languages skip, `User-Agent: gitview/0.1`, `Authorization` header present and **not** logged. Token passed in, never stored on a shared struct that gets printed.
      Check: `go test -race -count=1 ./internal/github`

## Group 5 — Load + JSON/text (Slice 1)

- [x] **5.1** Orchestrate fetch→stats→`internal/render` JSON (schema in requirements) and text bars (~20 chars). `-json` writes JSON only to stdout. Wire `run` so `gitview -json <user>` works against a test server (inject base URL in tests).
      Check: `go test -race -count=1 ./internal/render ./...`

## Group 6 — Cache + rate-limit messages (Slice 2)

- [x] **6.1** Disk cache, TTL 1h, `-fresh`, temp-dir tests. Cache file contains no token field. Second `Load` with valid cache does not call the test server.
      Check: `go test -race -count=1 ./internal/cache ./...`

## Group 7 — TUI (Slices 3–4)

- [x] **7.1** Bubble Tea model: `loading | ready | failed`, spinner `fetching <user>…`, `q` / Ctrl+C restore terminal exit 0, `r` refetch as `-fresh`. `-json` does not start tea.
      Check: `go test -race -count=1 ./internal/ui`
- [x] **7.2** Repo list: truncated name, stars, language, `updated`, `j/k`/arrows, viewport; no extra fetch on select; layout does not panic at 80×24 (fixed-size `tea.WindowSizeMsg` test).
      Check: `go test -race -count=1 ./internal/ui`

## Group 8 — Wrap-up (Slice 5)

- [x] **8.1** README: build, `export GITHUB_TOKEN`, flags, limits, “token is never printed”. Confirm `.env` ignored. Full verify recipe.
      Check: `gofmt -l . | grep . && exit 1; go vet ./...; go test ./...; go test -race ./...`
