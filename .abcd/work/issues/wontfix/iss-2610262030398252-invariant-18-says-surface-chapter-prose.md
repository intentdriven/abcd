---
schema_version: 1
id: "iss-2610262030398252"
slug: "invariant-18-says-surface-chapter-prose"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 28 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:173"
remedy: "none (filed automatically)"
finding_ref: cfk-704b6b5a7c24d0b7
refines: [iss-2609091956001547]
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The check's own definition of a shape claim is the dashed spelling, which the launch prose avoids, and the sentence exists to refuse a sub-verb reading. A one-line clarification of the invariant is optional, not owed."
---

Invariant 18 says surface-chapter prose states no flag the tree registers; the launch chapter's prose names its `dry-run` flag outright

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:173` — "The prose above the appendix states no flag the tree registers and no sub-verb written as an invocation"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:257` — "**The preview is spelled `dry-run`, and it is a flag, not a sub-verb.**"

Invariant 18 says the prose above a surface chapter's generated appendix states no flag the command tree registers. The launch chapter's prose names the preview's flag directly as `dry-run` and goes on to describe it. The appendix registers that flag, so the chapter's prose states a shape claim the invariant reserves for the appendix.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 28.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The check's own definition of a shape claim is the dashed spelling, which the launch prose avoids, and the sentence exists to refuse a sub-verb reading. A one-line clarification of the invariant is optional, not owed.
