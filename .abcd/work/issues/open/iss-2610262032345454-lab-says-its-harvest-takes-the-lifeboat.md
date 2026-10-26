---
schema_version: 1
id: "iss-2610262032345454"
slug: "lab-says-its-harvest-takes-the-lifeboat"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 287 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/31-lab.md:99"
remedy: "none (filed automatically)"
finding_ref: cfk-97916ed151c6ff3f
---

Lab says its harvest takes the lifeboat's section shape, but disembark gives the lifeboat a different set of sections

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/31-lab.md:99` — "assembles the lab's harvest in the lifeboat's section shape — intention, method, what worked, what is open, candidates, coverage"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:129` — "Each path derives from the brief-section-to-lifeboat-path mapping"

31-lab says the harvest is assembled in the lifeboat's section shape: intention, method, what worked, what is open, candidates and coverage. 02-disembark derives every lifeboat path from the brief-section-to-lifeboat-path mapping, which yields brief sections, a graveyard, a rescue spine, retrospectives, ADRs and an issue snapshot alongside coverage. Of lab's six sections only coverage is a lifeboat part, so a builder who follows lab's pointer to the lifeboat finds a different shape.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 287.
