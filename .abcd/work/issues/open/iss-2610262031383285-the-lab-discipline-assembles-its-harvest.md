---
schema_version: 1
id: "iss-2610262031383285"
slug: "the-lab-discipline-assembles-its-harvest"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 156 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention.md:40"
remedy: "none (filed automatically)"
related_intents: [itd-2609251624540864]
finding_ref: cfk-f4ec523a961973f3
---

The lab discipline assembles its harvest in the lifeboat's section shape but names sections the canonical skeleton does not have

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention.md:40` — "The harvest is assembled in the lifeboat's section shape — intention, method, what worked, what is open, candidates, coverage"
- End B: `.abcd/development/brief/00-meta.md:44` — "There is one canonical skeleton, used three ways."

The lab rule says the harvest is assembled in the lifeboat's section shape: intention, method, what worked, what is open, candidates, coverage. The meta page says there is one canonical skeleton used three ways, and its mapping table has no intention, method or candidates section. Two different shapes go by one name, which a lifeboat packed from a lab session would have to reconcile.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 156.
