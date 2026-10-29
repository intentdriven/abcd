---
schema_version: 1
id: "iss-2610262030536960"
slug: "the-truth-rule-admits-only-shipped-or"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 58 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/00-meta.md:9"
remedy: "Rewrite the 'The audit loop' subsection of 05-intent.md (per-intent audit_mode and audit_budget, loop-to-acceptance) to the itd-50 re-scope shape (the loop as the last stage of abcd build), or drop it until that spec lands; do it together with iss-2610160855131633, which already carries this remedy and also names the configuration keys in 05-internals/03-configuration.md."
finding_ref: cfk-a9ee3897f9bd5865
---

The truth rule admits only shipped or decided staged machinery; the intent chapter keeps a staged audit-loop design it says was re-scoped away

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/00-meta.md:9` — "but never an undecided option: the brief records no proposals, no alternatives still under consideration, no aspirational maybes"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:578` — "This per-intent shape predates itd-50's 2026-09-21 re-scope, which makes the loop the last stage of `abcd build` rather than a per-intent mode"

00-meta says the brief records no proposals or undecided options, only shipped machinery or a staged target a future change builds toward, while 05-intent sets out the per-intent audit_mode loop as staged and says that shape predates itd-50's re-scope, which moved the loop into abcd build, with the rewrite owed. The passage describes a design the record no longer builds toward.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 58.
