---
schema_version: 1
id: "iss-2609240227236354"
slug: "itd-6-reads-ready-but-four-prerequisites-are-unbuilt"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A, planning briefs"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/open/spc-2609211950427074-rp-mcp-only-integration.md"
---

itd-6 passes its readiness gate (intent ready itd-6: READY, all seven checks ok) but cannot be built from today's tree: its spec spc-2609211950427074 stands on four things that do not exist. (1) The build loop's validator stage the adapter implements, itd-2609201916151817 (planned, spec spc-2609202134338445 open; there is no build verb). (2) The layered oracle.review resolver the model-tier and pacing intents share, itd-2609170822093401 and itd-2609201925079472 (both planned, specs spc-2609180535002478 and spc-2609202134341288 open). (3) The command-line runner the adapters chapter entry sits beside, itd-2609201916056194 (planned, spec spc-2609221533057881 open). (4) An MCP client, which the spec's Approach names as a new dependency subject to the new-dependency sign-off; go.mod has none. itd-6 declares builds_on [itd-2] only, so the gate has no edge to refuse on and READY reads as buildable to a run that picks it.
