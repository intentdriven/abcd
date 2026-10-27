---
schema_version: 1
id: "iss-2610262032597891"
slug: "the-lab-discipline-says-a-lifeboat-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 345 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention.md:40"
remedy: "none (filed automatically)"
related_intents: [itd-2609251624540864]
finding_ref: cfk-eef5a47e33646a86
---

The lab discipline says a lifeboat's sections are intention, method, what worked, what is open, candidates and coverage; disembark takes a lifeboat's sections from the brief-section mapping

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention.md:40` — "The harvest is assembled in the lifeboat's section shape — intention, method, what worked, what is open, candidates, coverage"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:129` — "Each path derives from the brief-section-to-lifeboat-path mapping in `internal/core/lifeboat/mapping.go`"

The lab procedure assembles its harvest 'in the lifeboat's section shape' and lists six sections. The disembark chapter derives every lifeboat path from the brief-section-to-lifeboat-path mapping, giving brief sections, graveyard, rescue, retrospectives, ADRs, activity and coverage. Only coverage appears in both. 'Lifeboat' names one shape in one document and a different shape in the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 345.
