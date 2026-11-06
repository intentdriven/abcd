---
schema_version: 1
id: "iss-2611060809360063"
slug: "abcd-development-brief-06-delivery"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Limit the row's fallback to an unarmed backend and write paths, and state that a release refuses when an armed backend is absent."
---

`.abcd/development/brief/06-delivery/02-verification-matrix.md:29` says a missing opt-in scanner backend falls back to the native scan, never a hard failure on absence. A repository that armed `gitleaks` in `.abcd/config/gitleaks.json` and lacks the binary fails closed: `internal/adapter/gitleaks/gitleaks.go:19-25` returns `ErrConfiguredNotFound`, carried as a coverage gap on which a release refuses (`internal/core/launch/dryrun.go:321`), and only a write path proceeds on the native scanner and names the gap. `.abcd/development/brief/04-surfaces/01-ahoy.md:980` already calls `gitleaks` required where the repository armed it.
