---
schema_version: 1
id: "iss-2610262041175899"
slug: "the-foundations-auditor-draft-says-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 990 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md:99"
remedy: "none (filed automatically)"
related_intents: [itd-55]
finding_ref: cfk-993a1aa12f978831
---

The foundations-auditor draft says abcd already stress-tests acceptance criteria with the grill; the build sequence says there is no grill sub-verb.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/01-build-sequence.md:99` — "There is no `grill` sub-verb"
- End B: `.abcd/development/intents/drafts/itd-55-first-principles-foundations-auditor.md:17` — "abcd already stress-tests acceptance criteria (the grill)"

itd-55 positions itself against an existing grill that stress-tests acceptance criteria. 01-build-sequence.md says there is no `grill` sub-verb: what itd-27 asked for became the planning interview under itd-94. The surface the draft defines itself against does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 990.
