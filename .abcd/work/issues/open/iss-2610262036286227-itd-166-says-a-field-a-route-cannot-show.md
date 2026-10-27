---
schema_version: 1
id: "iss-2610262036286227"
slug: "itd-166-says-a-field-a-route-cannot-show"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 650 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-166-each-run-should-record-the-model.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-166]
finding_ref: cfk-ffcbd9bde1eac6c8
---

itd-166 says a field a route cannot show is recorded as unpinned or self-reported, never left out; the reading chapter's route receipt leaves model_reported empty when the payload names no model

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-166-each-run-should-record-the-model.md:18` — "Whatever the route cannot show is written down as unpinned or self-reported, never left out and never guessed."
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:355` — "`model_reported`, the last read from the payload's own `model` field (a reading's `instrument.model`) and empty when the payload names none"

itd-166 extends the override's route receipt so that every model call records the served side, with a missing field written down under a stated reason and never omitted. 23-reading describes the ingest's route receipt as carrying model_reported read from the payload and empty when the payload names none, with no reason recorded. An empty field is what itd-166 says the receipt never holds.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 650.
