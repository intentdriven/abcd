---
schema_version: 1
id: "iss-2610262045343669"
slug: "itd-4-s-issue-schema-carries-related"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1283 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-4-issue-capture.md:42"
remedy: "none (filed automatically)"
related_intents: [itd-4, itd-43]
finding_ref: cfk-5a8451e74cf00cbf
---

itd-4's issue schema carries related_epics; itd-43 says the validator uses related_specs exclusively

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-4-issue-capture.md:42` — "`related_epics` (list of `spc-N`)"
- End B: `.abcd/development/intents/shipped/itd-43-epic-to-spec-terminology.md:42` — "the native validator already uses `related_specs` exclusively"

itd-4 defines the issue frontmatter with related_epics as the list of spc-N. itd-43 states the native validator already uses related_specs exclusively and that epic is a forbidden synonym, so the two shipped records name the same issue field two ways.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1283.
