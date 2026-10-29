---
schema_version: 1
id: "iss-2610262033418681"
slug: "the-intent-chapter-says-itd-to-spc-link"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 430 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:528"
remedy: "Reword 05-intent.md line 528: itd-to-spc link drift blocks (spec_lifecycle at blocker); only the warn-armed rules (stale_edge, edge_cycle, the reciprocal-link rules) warn. Also correct line 526, which says the issue back-link is not checked by the record lint although related_issues_backlink is armed at warn (line 511)."
finding_ref: cfk-764359008b818b00
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

The intent chapter says itd-to-spc link drift warns and never blocks; the verification matrix says the same disagreement is a blocker that refuses to lint.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:528` — "Drift triggers a warning, not a block (since spec-store state may legitimately lag intent state during work in progress)."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:60` — "when an intent's spec reference and the spec's intent reference disagree: the pair refuses to lint until they agree"

05-intent states that link drift 'triggers a warning, not a block' because spec-store state may lag intent state during work in progress. The matrix's Bidirectional link drift row says that when an intent's spec reference and the spec's intent reference disagree 'the pair refuses to lint until they agree', calls it a blocker, and places it in the pre-push gate and CI. One rule cannot both warn and refuse on the same disagreement, so a contributor cannot tell from the record whether a mismatched link stops a push.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 430.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
