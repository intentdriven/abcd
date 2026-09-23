---
schema_version: 1
id: "iss-380"
slug: "homebrew-tap-distribution"
severity: "minor"
category: "future-work-seed"
source: "review-followup"
found_during: "update-mechanism SOTA"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: Homebrew tap parked in DECISIONS 2026-08-20; its trigger is now met (repo public, abcd update exists)). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Homebrew tap for abcd (a personal tap, brew install REPPL/tap/abcd). Trigger conditions are now MET: the repo is public and abcd update exists (its install-channel refusal already prints brew upgrade abcd on a Cellar-resolved path). Recorded and parked in DECISIONS.md 2026-08-20; this ledger entry makes it queryable rather than prose. abcd's release pipeline is bespoke (semantic-gate attestations, no goreleaser), so the tap push is a hand-rolled release-workflow step, not goreleaser's homebrew_casks pipe. Ready for triage rather than deferral.