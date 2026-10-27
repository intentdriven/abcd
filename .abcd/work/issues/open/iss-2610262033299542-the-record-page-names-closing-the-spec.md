---
schema_version: 1
id: "iss-2610262033299542"
slug: "the-record-page-names-closing-the-spec"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 412 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md:101"
remedy: "none (filed automatically)"
finding_ref: cfk-2b21202c8bb04a8d
---

The record page names closing the spec as a READY planned intent's next move; the build chapter makes READY the state from which an intent is built.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/08-abcd.md:101` — "For a ready planned intent, and for its open spec, the move is closing the spec"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:48` — "the implement-readiness gate the intent verb reports: planned, criteria written, the spec linked both ways and written past its stub"

08-abcd says that, for a ready planned intent and for its open spec, the next move `abcd <record-id>` reports is closing the spec, which ships the intent. 34-build defines READY through the implement-readiness gate (planned, criteria written, spec linked both ways and written past its stub) as the condition for starting a build. The close happens only at the landing of the lane that delivers the last step. Following the record page for a READY intent would close its spec and ship it before any of it is built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 412.
