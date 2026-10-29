---
schema_version: 1
id: "iss-2610262039290078"
slug: "the-land-page-calls-am1-s-arming-human"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 863 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610150602547271-a-repository-declares-whether-its.md:163"
remedy: "none (filed automatically)"
related_intents: [itd-2610150602547271]
finding_ref: cfk-59ae53671feeb3d7
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. internal/core/landing/land.go mergeGate and commands/land.md say every repository runs in human mode and AM1 is the behaviour until the toggle ships; the intent says it keeps AM1 only as the stand-in until then. The two describe different times, and the page is rewritten by the intent's own spec."
---

The land page calls AM1's arming 'human mode', but the facilitator-mode intent rules that `human` arms nothing, even where the forge would hold the merge for approval

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610150602547271-a-repository-declares-whether-its.md:163` — "`human` refuses arming outright, also where the forge would hold"
- End B: `.abcd/development/brief/04-surfaces/38-land.md:124` — "and in human mode ruling AM1"

The land page says every repository runs in human mode until the mode toggle ships, and that in human mode AM1 arms auto-merge where a queue and a required approval exist. itd-2610150602547271 defines `human` as arming nothing, even where the forge would hold the merge for an approval, and keeps AM1 only as the gate's stand-in until the intent ships. 'Human mode' means arming under AM1 in one document and refusing to arm in the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 863.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. internal/core/landing/land.go mergeGate and commands/land.md say every repository runs in human mode and AM1 is the behaviour until the toggle ships; the intent says it keeps AM1 only as the stand-in until then. The two describe different times, and the page is rewritten by the intent's own spec.
