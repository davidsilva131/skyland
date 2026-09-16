# Skyland

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`gh` CLI). See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical roles mapped to defaults (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout — `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.

### Specs

Implementation specs live in `docs/specs/`. Read the matching spec before touching an area it covers; it is the source of truth over chat history. `docs/specs/auth-player.md` governs player auth: `internal/auth`, `internal/ratelimit`, the auth routes, the `players`/`sessions` tables, and `apps/web` session/login code.

### Language

English only for everything an agent produces: docs, code comments, commit messages, ADRs, issue and PR text. Never Spanish, and never mixed Spanish/English inside the same document — even when the user writes in Spanish. Existing Spanish docs get converted when next touched.