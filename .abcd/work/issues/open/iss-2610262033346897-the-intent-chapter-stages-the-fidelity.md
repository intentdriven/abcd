---
schema_version: 1
id: "iss-2610262033346897"
slug: "the-intent-chapter-stages-the-fidelity"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 421 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:651"
remedy: "none (filed automatically)"
finding_ref: cfk-7a6ac4997ce3fef8
---

The intent chapter stages the fidelity audit's request under the local reviews directory, while universal patterns puts the same audit's output in a review sub-tier of the log tier.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:651` — "the emit path stages an ephemeral review request under `.abcd/.work.local/reviews/` (gitignored, report-only)"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:120` — "audit/review-< ts>/ (Role 1 itd-1 pass / /abcd:intent audit, itd-1)"

05-intent says the shipped audit keeps its verdict in the intent's Audit Notes, and that its emit stages an ephemeral review request under .abcd/.work.local/reviews/. 04-universal-patterns gives the single-document audit pass a per-run directory, audit/review-< ts>/, in the log tier. The two documents put the same audit's per-run artefact in different directories.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 421.
