---
schema_version: 1
id: "iss-2609240227236354"
slug: "itd-6-reads-ready-but-four-prerequisites"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A, planning briefs"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/open/spc-2609211950427074-rp-mcp-only-integration.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (run A 2026-09-29, lane drainDrift3; rulings-owed BF): itd-6's builds_on and Implementation status now name the four missing prerequisites, but abcd intent ready reads no builds_on edge, so it still says READY. Should the readiness gate gain a check that refuses an intent whose builds_on names an intent not yet shipped (a ninth check, changing every planned intent's verdict that builds on planned work), or is the recorded prerequisite list enough and the gate stays as it is?"
remedy: "Waits on ruling BF: if a ninth check: abcd intent ready refuses an intent whose builds_on names an intent not in shipped/, naming the edge, proven by a ready_test.go fixture with a planned prerequisite; if not: resolve on itd-6's recorded builds_on and Implementation status. Prerequisite 4 (the MCP client) is settled by ruling H1: the official modelcontextprotocol go-sdk v1.8.0, signed off."
---

itd-6 passes its readiness gate (intent ready itd-6: READY, all seven checks ok) but cannot be built from today's tree: its spec spc-2609211950427074 stands on four things that do not exist. (1) The build loop's validator stage the adapter implements, itd-2609201916151817 (planned, spec spc-2609202134338445 open; there is no build verb). (2) The layered oracle.review resolver the model-tier and pacing intents share, itd-2609170822093401 and itd-2609201925079472 (both planned, specs spc-2609180535002478 and spc-2609202134341288 open). (3) The command-line runner the adapters chapter entry sits beside, itd-2609201916056194 (planned, spec spc-2609221533057881 open). (4) An MCP client, which the spec's Approach names as a new dependency subject to the new-dependency sign-off; go.mod has none. itd-6 declares builds_on [itd-2] only, so the gate has no edge to refuse on and READY reads as buildable to a run that picks it.

## Remedy grounds (2026-09-29)

Confirmed at this base: itd-6 declares all four prerequisites in builds_on, and no reader in internal/core/intent's readiness gate reads it. Rejected: stamping itd-6 back to drafts, which would discard a completed planning interview.
