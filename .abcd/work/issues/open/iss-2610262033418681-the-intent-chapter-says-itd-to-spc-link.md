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
remedy: "none (filed automatically)"
finding_ref: cfk-764359008b818b00
---

The intent chapter says itd-to-spc link drift warns and never blocks; the verification matrix says the same disagreement is a blocker that refuses to lint.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:528` — "Drift triggers a warning, not a block (since spec-store state may legitimately lag intent state during work in progress)."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:60` — "when an intent's spec reference and the spec's intent reference disagree: the pair refuses to lint until they agree"

05-intent states that link drift 'triggers a warning, not a block' because spec-store state may lag intent state during work in progress. The matrix's Bidirectional link drift row says that when an intent's spec reference and the spec's intent reference disagree 'the pair refuses to lint until they agree', calls it a blocker, and places it in the pre-push gate and CI. One rule cannot both warn and refuse on the same disagreement, so a contributor cannot tell from the record whether a mismatched link stops a push.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 430.
