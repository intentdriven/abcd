---
schema_version: 1
id: "iss-2610262045503347"
slug: "release-lands-a-held-lane-on-implement"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1301 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609201925079472-an-autonomous-implementation-run-paces.md:58"
remedy: "none (filed automatically)"
related_intents: [itd-2609201925079472, itd-2609091034175565]
finding_ref: cfk-a238acc654d03974
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. commands/build.md, implement.md and reserve.md each document their own use."
---

--release lands a held lane on implement step but gives back a reservation on build

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-2609201925079472-an-autonomous-implementation-run-paces.md:58` — "`implement step --release <lane>` or `--discard <lane>`"
- End B: `.abcd/development/intents/shipped/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md:90` — "`abcd build <itd-N> --release` gives it back"

itd-2609201925079472 uses implement step --release < lane> to let a held sibling lane land after a hand-back. itd-2609091034175565 uses abcd build < itd-N> --release to give the intent's reservation back. build hands over to implement, so the same flag word names two different acts in one loop, beside release as a version cut.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1301.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. commands/build.md, implement.md and reserve.md each document their own use.
