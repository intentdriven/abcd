---
schema_version: 1
id: "iss-2610262039022652"
slug: "itd-107-keeps-arming-auto-merge-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 835 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-107-autonomous-routines-assemble-from-one.md:143"
remedy: "none (filed automatically)"
related_intents: [itd-107]
finding_ref: cfk-c82af81c37d066b3
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037186224. Same bullet of itd-107 and the same fact, cited from the land chapter instead of the build chapter."
duplicates: [iss-2610262037186224]
---

itd-107 keeps arming auto-merge and polling merge state out of the binary; 38-land's binary verb arms and confirms merges itself

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-107-autonomous-routines-assemble-from-one.md:143` — "Arming auto-merge, polling merge state, updating a stale branch, and re-reading a created artefact are host-executed steps"
- End B: `.abcd/development/brief/04-surfaces/38-land.md:120` — "Whether the verb arms a merge is decided in one function, `mergeGate`"

itd-107 says arming auto-merge, polling merge state and updating a stale branch are host-executed steps in the rendered prompt, never in the binary, so the binary takes on no platform CLI dependency. 38-land's shipped verb decides and performs the merge arming itself in mergeGate, through the forge client. The binary already makes the platform calls itd-107 keeps out of it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 835.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037186224. Same bullet of itd-107 and the same fact, cited from the land chapter instead of the build chapter.
