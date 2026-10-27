---
schema_version: 1
id: "iss-2610262041209286"
slug: "the-cold-reading-draft-only-ever-reads-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 993 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/reading-position.md:30"
remedy: "none (filed automatically)"
related_intents: [itd-86]
finding_ref: cfk-62a6b515c983c966
---

The cold-reading draft only ever reads a design document, never code; the glossary's detection position reads the shipped tree against the claim record.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/reading-position.md:30` — "Where is the shipped tree in tension with the claim record?"
- End B: `.abcd/development/intents/drafts/itd-86-cold-reading-surface.md:41` — "The target is a design document."

reading-position.md defines four positions for a cold reading, one being detection: where the shipped tree is in tension with the claim record. itd-86 says a cold reading's target is a design document, keeps code out of scope, and takes exactly one document. The two disagree on what a cold reading may read.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 993.
