---
schema_version: 1
id: "iss-2610262032262544"
slug: "embark-s-lessons-view-reads-only"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 268 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:57"
remedy: "none (filed automatically)"
finding_ref: cfk-f88fcde084537f2f
---

Embark's lessons view reads only retrospective lessons; disembark's graveyard also writes lessons into the same lifeboat

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:57` — "answer what the previous voyage learned (itd-24): every lesson the lifeboat's release retrospectives carry"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:191` — "returns the lessons payload the graveyard sub-verb validates, each lesson citing the finding ids it rests on"

Embark says its lessons sub-verb answers what the previous voyage learned by reading every lesson the lifeboat's release retrospectives carry. Disembark's graveyard sub-verb writes lessons of its own into the lifeboat, each citing the finding ids it rests on. One word names two different lesson records in one pack, and the view that claims to say what the voyage learned reads only one of them.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 268.
