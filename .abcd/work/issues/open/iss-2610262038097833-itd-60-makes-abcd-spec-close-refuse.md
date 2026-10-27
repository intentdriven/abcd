---
schema_version: 1
id: "iss-2610262038097833"
slug: "itd-60-makes-abcd-spec-close-refuse"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 775 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:124"
remedy: "none (filed automatically)"
related_intents: [itd-60]
finding_ref: cfk-ed05aadd9c1412da
---

itd-60 makes `abcd spec close` refuse while the brief lags, failing closed without a reviewer; 34-build's closing lane runs the spec's close with no doc-fidelity review among its stages

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:124` — "`abcd spec close`, the verb that moves a planned intent to `shipped/`, refuses per intent while the brief lags the surface that intent delivered"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:743` — "In the lane's worktree it runs the spec's close (which ships the intent and parks the fidelity receipt)"

itd-60 makes the close that moves a planned intent to `shipped/` refuse per intent while the brief lags the delivered surface, its semantic layer host-delegated and failing closed when no reviewer runs. 34-build's closing lane runs the spec's close in its worktree to ship the intent, and its validate stage hands the lane only to the ruthless reviewer, the security reviewer and the intent-auditor, so nothing the lane runs produces the doc-fidelity verdict that close requires. As written, the closing landing depends on a close itd-60's gate refuses.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 775.
