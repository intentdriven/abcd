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
resolution: "The configuration chapter and the comments on DefaultRefreshBackstop, InjectWithin and LoadBackstop now say the backstop clears the dedup and recalled ledgers, so any domain or memory note the prompt matches is sent again and nothing unmatched is sent. A test pins both the wording and the behaviour."
impact: internal
resolved_by:
  commit: "5a8d8dda2952a03b3f9ff35b6a43f13d352a8ee6"
---

`.abcd/development/brief/05-internals/03-configuration.md:150` says the `force_refresh_every_n` counter only re-injects always-relevant domains. At the backstop `Inject` clears the whole ledger and the recalled ledger, so every domain the prompt matches comes back, and `RuleSet.Match` (`internal/core/rules/rules.go:648`) injects nothing the prompt does not name or hit, so a domain that never recall-matches is not injected at all. The comments at `internal/core/rules/inject.go:56`, `internal/core/rules/inject.go:147` and `internal/core/rules/inject.go:455` carry the same wording.

## Grounds

- pursued: on a backstop prompt InjectWithin sends again only what Match returns for that prompt. This would be shown wrong if a backstop prompt sent a domain the prompt does not match; TestBackstopSendsOnlyWhatThePromptMatches checks for that.
