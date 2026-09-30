---
schema_version: 1
id: "iss-380"
slug: "homebrew-tap-distribution"
severity: "minor"
category: "future-work-seed"
source: "review-followup"
found_during: "update-mechanism SOTA"
deferred_after: "v0.11.1"
deferral_reason: "decide later (product thinker, 2026-09-29)"
---

Homebrew tap for abcd (a personal tap, brew install REPPL/tap/abcd). Trigger conditions are now MET: the repo is public and abcd update exists (its install-channel refusal already prints brew upgrade abcd on a Cellar-resolved path). Recorded and parked in DECISIONS.md 2026-08-20; this ledger entry makes it queryable rather than prose. abcd's release pipeline is bespoke (semantic-gate attestations, no goreleaser), so the tap push is a hand-rolled release-workflow step, not goreleaser's homebrew_casks pipe. Ready for triage rather than deferral.