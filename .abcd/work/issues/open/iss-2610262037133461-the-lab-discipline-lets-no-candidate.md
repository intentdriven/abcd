---
schema_version: 1
id: "iss-2610262037133461"
slug: "the-lab-discipline-lets-no-candidate"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 707 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/31-lab.md:103"
remedy: "none (filed automatically)"
related_intents: [itd-2609251624540864]
finding_ref: cfk-1ef617386e86c7e1
---

The lab discipline lets no candidate into a harvest before a refutation pass; the lab verb's harvest lists candidates nobody tried to refute and only flags them.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/31-lab.md:103` — "and flagged where no refutation was attempted"
- End B: `.abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention.md:119` — "no candidate enters the harvest before a refutation pass"

itd-2609251624540864 makes a refutation pass a precondition for a candidate entering the harvest. 31-lab says its procedure is that discipline, yet its harvest lists every product finding as a capture candidate and only flags those where no refutation was attempted. So the harvest the verb builds can hold exactly the candidates the procedure forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 707.
