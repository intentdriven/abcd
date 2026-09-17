---
schema_version: 1
id: "iss-2609100507432904"
slug: "the-installed-binary-lacks-verbs-the-repo-s-records-name"
severity: "major"
category: "ux"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (version, ahoy session-start hook, decide)"
resolution: "Fixed in v0.8.0: abcd decide exists and mints a decision record, allocating its id through the same collision-proof seam every other record family uses. The instance this record names is closed, verified against the current binary. The general condition it sits inside, that an agent in a managed repository runs the published release and so meets problems already solved upstream, is a separate finding and is recorded separately."
impact: fix
---

The installed binary can lack verbs the repository's own records tell an author to run, and nothing says so until each verb fails on first use.

Observed across a full day of autonomous work in a managed repository. `abcd` on PATH was a release in which `abcd decide` does not exist: it exits 2 with "this binary predates the decide command". The ADRs README in that repository tells authors to run exactly that verb. Three ADRs were therefore hand-minted, each by copying a neighbouring file's shape and guessing at the frontmatter, by workers who had no way to know whether the shape they copied was current. The ledger half of the tool was fine throughout — `capture resolve`, `intent ready` and `spec close` all worked — so the failure was not a broken install, it was a version skew nobody could see.

The skew is discoverable in principle: the repository's records name the verbs they expect, and the binary knows which verbs it has. Nothing compares them. Each verb simply fails when a worker reaches it, which in an autonomous run means the worker improvises rather than stops, and the improvisation lands in the durable record.

Wanted, cheapest first: have `abcd version --check` (or the session-start hook) report when the repo's records or docs reference a verb the installed binary lacks, rather than leaving each verb to fail on first use. And give a missing verb a useful refusal: `decide` exiting 2 could still print the expected filename and frontmatter for a hand-minted ADR, so an author who has no choice but to hand-mint one produces the right shape instead of a copied guess.

## Grounds

- pursued: we expect a released decide verb to close this because the records that told authors to run it can now be followed literally; it is shown wrong if a managed repo on the release still cannot mint an ADR
