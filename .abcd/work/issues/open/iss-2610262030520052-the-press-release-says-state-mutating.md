---
schema_version: 1
id: "iss-2610262030520052"
slug: "the-press-release-says-state-mutating"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 56 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:5"
remedy: "none (filed automatically)"
finding_ref: cfk-edb0bdb77e1a80ce
---

The press release says state-mutating actions require an explicit sub-verb; the capture fast path writes a record from a bare quoted argument

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:5` — "state-mutating actions require an explicit sub-verb (the universal abcd convention)"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:61` — "is the fast path: it appends a structured entry with an auto-assigned `iss-N` and writes it to `open/`"

01-press-release says state-mutating actions require an explicit sub-verb as the universal convention, while 06-capture's fast path /abcd:capture "< text>" takes no sub-verb and appends a new record to open/. The quoted create is a state-mutating form with no sub-verb.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 56.
