---
schema_version: 1
id: "iss-2610262031039632"
slug: "the-meta-page-says-the-brief-lifeboat"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 80 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/00-meta.md:44"
remedy: "none (filed automatically)"
finding_ref: cfk-86e3bd635db4907a
---

The meta page says the brief/lifeboat skeleton is the one canonical shape, but the lab chapter calls an intention/method/candidates layout 'the lifeboat's section shape'

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/00-meta.md:44` — "There is one canonical skeleton, used three ways."
- End B: `.abcd/development/brief/04-surfaces/31-lab.md:99` — "assembles the lab's harvest in the lifeboat's section shape — intention, method, what worked, what is open, candidates, coverage"

The meta page says one canonical skeleton is used three ways (ahoy, disembark, embark), and its mapping table is the contract for the lifeboat's sections. The lab chapter says the harvest is assembled in the lifeboat's section shape, then lists intention, method, what worked, what is open, candidates and coverage. Intention, method and candidates are not sections of that skeleton. Either the harvest is not in the lifeboat's shape, or the skeleton is not the one canonical shape.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 80.
