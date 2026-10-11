---
schema_version: 1
id: "iss-2608261132593030"
slug: "private-banlist-write-gate-fails-open"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-8"
found_at: "internal/core/banlist/private.go:610"
resolution: "requireIgnoredStore refuses in a repo-shaped tree git cannot answer for and ListPrivate carries a distinct ignore_unanswerable state, matching ahoy storePathIsSafe"
impact: fix
resolved_by:
  commit: "1dd23e5b"
---

requireIgnoredStore and the ListPrivate warning skip whenever InRepo is false, so with git unanswerable in a repo-shaped tree banlist add --private writes the secret-pattern store unignored with exit 0