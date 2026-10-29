---
schema_version: 1
id: "iss-2610262039314007"
slug: "itd-50-calls-the-implementation-fix-loop"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 866 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-50]
finding_ref: cfk-fcfc1fc436801f22
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037443119. Same End A (itd-50 line 47, SHIP/NEEDS_WORK); the agent catalog says the implement loop's reviewer states SHIP or FIX FIRST."
duplicates: [iss-2610262037443119]
---

itd-50 calls the implementation fix loop a SHIP/NEEDS_WORK loop; the agent catalog says the loop's reviewer states SHIP or FIX FIRST, and that NEEDS_WORK belongs to the lifeboat review

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:47` — "Mirrors the implementation-grain `SHIP`/`NEEDS_WORK` fix-loop, lifted to the intent grain."
- End B: `.abcd/development/brief/05-internals/01-agents.md:233` — "`ruthless-reviewer` states `SHIP` or `FIX FIRST` under its `### Verdict` heading"

itd-50 says its loop mirrors the implementation-grain `SHIP`/`NEEDS_WORK` fix loop. The agent catalog's verdict protocol says the implement loop's ruthless reviewer states `SHIP` or `FIX FIRST`, and keeps `{SHIP, NEEDS_WORK, MAJOR_RETHINK}` as the lifeboat review's enum, which is never mixed with the other families. The intent therefore uses one verdict family's words for a different family's loop.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 866.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037443119. Same End A (itd-50 line 47, SHIP/NEEDS_WORK); the agent catalog says the implement loop's reviewer states SHIP or FIX FIRST.
