# 0005. Default five recently updated repos

Status: accepted
Date: 2026-09-17

## Context

The TUI listed every included repo, stars-desc. That is a long, low-signal table. The useful default is language bars plus a handful of recently touched repos. Scripts using `-json` should match what the TUI shows. `-all` must not refetch if the cache already has the full set.

## Options

- **Option A** — Default: 5 repos by `updated_at` desc. `-all`: full list, same order. Cache stores the full included set. JSON `repos` follows the same slice; `repos_count` and `stars` stay totals.
- **Option B** — Default still all repos; only the TUI hides extras (JSON stays a full dump).
- **Option C** — No repo list unless `-all`.

## Decision

Option A. Display is a view over the cached snapshot. Heading is **Recently updated** (GitHub `updated_at`, not the contribution calendar).

## Consequences

`Build` sorts the full list by updated desc. Render and TUI take 5 unless `-all`. Old caches with stars-desc order are wrong until TTL or `-fresh`.
