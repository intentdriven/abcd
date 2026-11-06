---
schema_version: 1
id: "iss-2611060809419595"
slug: "abcd-development-brief-05-internals"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Say the backstop clears the dedup ledgers so any domain or note the prompt matches is sent again, and correct the three comments."
---

`.abcd/development/brief/05-internals/03-configuration.md:150` says the `force_refresh_every_n` counter only re-injects always-relevant domains. At the backstop `Inject` clears the whole ledger and the recalled ledger, so every domain the prompt matches comes back, and `RuleSet.Match` (`internal/core/rules/rules.go:648`) injects nothing the prompt does not name or hit, so a domain that never recall-matches is not injected at all. The comments at `internal/core/rules/inject.go:56`, `internal/core/rules/inject.go:147` and `internal/core/rules/inject.go:455` carry the same wording.
