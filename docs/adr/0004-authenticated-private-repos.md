# 0004. Private repos of the authenticated user

Status: accepted
Date: 2026-09-17

## Context

`GET /users/{user}/repos` returns **public** repos only, even with a PAT. A token with access to private repos has no effect on that URL. gitview was showing only public activity for the token owner. `GET /user/repos` is the endpoint that includes private repos the token can see. Other people's private repos stay invisible.

## Options

- **Option A** — Keep `/users/{user}/repos` only (public activity, original requirements).
- **Option B** — `GET /user`; if `login` matches the requested user (case-insensitive), list `/user/repos?affiliation=owner`; otherwise keep `/users/{user}/repos`.
- **Option C** — Always `/user/repos` (cannot view another login).

## Decision

Option B. Viewing your own login with a token that can read private repos includes those repos. Viewing anyone else stays public-only. Organization repos stay out (requirements non-goal).

## Decision mechanics

Classic PAT needs the `repo` scope for private repos. Fine-grained PAT needs repository access that includes those private repos (e.g. “All repositories”). JSON schema is unchanged. A stale cache from before this change is public-only until TTL expires or `-fresh`.

## Consequences

One extra `GET /user` per live fetch. Tests must stub `/user`. README documents the scopes. Other users' private repos cannot appear.
