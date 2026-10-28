---
schema_version: 1
id: "iss-2610262041289117"
slug: "itd-147-is-sequenced-as-phase-8-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1002 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-147-the-brief-s-surface-chapters-are-a.md:99"
remedy: "none (filed automatically)"
related_intents: [itd-147]
finding_ref: cfk-a666b9be5e69c153
---

itd-147 is sequenced as Phase 8; the phase glossary entry says roadmap phases run 0–7

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-147-the-brief-s-surface-chapters-are-a.md:99` — "Sequenced as Phase 8"
- End B: `.abcd/development/brief/glossary/core/phase.md:43` — "Roadmap phases run 0–7 and the brief's"

itd-147 says it was sequenced as Phase 8 (the brief is the shipped state). The phase glossary entry states that roadmap phases run 0–7, against the brief's plumbing-phases 0–6. A Phase 8 is outside the numbering the glossary gives, so one of the two is wrong about the phase set. Phases are retired, so the cost is low, but the entry exists to keep those history documents readable.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1002.
