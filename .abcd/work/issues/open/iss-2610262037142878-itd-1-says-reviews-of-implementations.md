---
schema_version: 1
id: "iss-2610262037142878"
slug: "itd-1-says-reviews-of-implementations"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 709 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:47"
remedy: "In itd-1 line 47 say the SHIP, NEEDS_WORK, MAJOR_RETHINK family assesses a packed lifeboat only (01-agents.md verdict-tag protocol) and that the implement loop's reviewers state SHIP or FIX FIRST, APPROVE, BLOCK or NEEDS-INPUT; drop the oracle reviews of plans, implementations and completions and the native receipt schema validator."
related_intents: [itd-1]
finding_ref: cfk-6c071faeba99b9d5
---

itd-1 says reviews of implementations return SHIP / NEEDS_WORK / MAJOR_RETHINK; the build's implementation reviewers return SHIP or FIX FIRST.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:47` — "(`SHIP` / `NEEDS_WORK` / `MAJOR_RETHINK`) which score *changes/runs*"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:696` — "stating one verdict of its role (`SHIP` or `FIX FIRST`"

itd-1 fixes one set of review verdicts, kept separate from criterion verdicts, for reviews of plans, implementations and completions: SHIP / NEEDS_WORK / MAJOR_RETHINK. In 34-build the validators review a lane's implementation, and the loop reads one verdict per role: SHIP or FIX FIRST. The two documents give different words for the same set of verdicts, so a reader built from itd-1 would reject what the build's reviewers return.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 709.
