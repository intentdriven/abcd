---
schema_version: 1
id: "iss-2610262035195318"
slug: "the-docs-chapter-has-the-spec-close"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 563 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/10-docs.md:92"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does an unattended landing's spec close owe the doc-fidelity gate (a docs-review stage in the lane before the close), or is the gate deliberately left to the release cut for loop landings? Answer 1: leave it to the cut and say so in 34-build.md and 10-docs.md. Answer 2: add a docs-review stage to the lane. Recommend answer 1: the receipt is keyed on HEAD, and a squash or rebase merge changes HEAD after the lane's review, so a per-lane receipt would not match what lands."
finding_ref: cfk-8eaff51bf8bff544
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

The docs chapter has the spec close refuse without a saved docs review for HEAD, but the build chapter's unattended landing runs that close with no stage that records one

A sequencing impossibility (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/10-docs.md:92` — "The spec close runs it over the intents the close would ship, before anything moves"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:743` — "In the lane's worktree it runs the spec's close (which ships the intent"

10-docs says the doc-fidelity gate runs at the spec close over the intents it would ship, and a missing receipt for HEAD refuses with 'run the docs review first'. 34-build's landing runs the spec close in the lane's worktree to ship the intent, but its lane stages (worktree, brief, implementer, validators: ruthless, security, intent-auditor; landing) never run a docs review or record a docs-fidelity receipt for the lane's head. In a repository where the gate is armed, every spec-closing landing would be refused, so the loop as written cannot deliver an intent unattended.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 563.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
