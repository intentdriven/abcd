---
schema_version: 1
id: "iss-2610262037509765"
slug: "the-spec-steps-intent-starts-each-step-s"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 753 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md:417"
remedy: "none (filed automatically)"
related_intents: [itd-2609212103565953]
finding_ref: cfk-7166b4d8b7d69cb3
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The two agree for a spec with no needs line; the later shipped pacing ruling names the parallel exception, and 34-build states both."
---

The spec-steps intent starts each step's lane only after the previous step merged; the build chapter opens a lane once the steps its needs line names have landed, so lanes can run side by side.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/34-build.md:417` — "A lane opens for a spec step once every step it needs has landed (its `- needs:` line, or by default every step before it, ruling DR6b)"
- End B: `.abcd/development/intents/shipped/itd-2609212103565953-a-spec-lists-its-steps-and-the-build.md:40` — "runs one lane per step in order, each landing as its own pull request; the next step's lane starts after the previous has merged"

itd-2609212103565953 says the loop runs one lane per step in order and the next step's lane starts after the previous has merged. 34-build.md opens a lane once every step it needs has landed, read from the step's - needs: line, with every earlier step only as the default, and runs lanes in parallel up to the ceiling. The two agree for a spec that declares no needs. They disagree for one that does: there the brief opens a later step's lane before the previous one merges.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 753.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The two agree for a spec with no needs line; the later shipped pacing ruling names the parallel exception, and 34-build states both.
