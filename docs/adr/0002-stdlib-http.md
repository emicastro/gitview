# 0002. Stdlib HTTP client

Status: accepted
Date: 2026-09-17

## Context

gitview needs two GitHub REST resources: paginated repo list and per-repo languages. Tests must run offline. A generated or third-party GitHub SDK would pull a large API surface for those two calls.

## Options

- **Option A** — `net/http` + `encoding/json` against `https://api.github.com` (injectable base URL in tests).
- **Option B** — `google/go-github` (or similar) for pagination and typed errors.

## Decision

Option A. The endpoints and error mapping are small enough to own; `httptest` is the contract for pagination, 401/403/404, and the `Authorization` header.

## Consequences

We implement Link-header pagination and GitHub error bodies ourselves. We do not take on SDK version drift. Tests never dial the real API.
