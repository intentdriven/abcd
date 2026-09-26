---
schema_version: 1
id: "iss-2609261036366114"
slug: "scribe-ingest-json-renders-refusals-reason-refusals-subject"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/scribe.go"
resolution: "scribe ingest's JSON render carries refusals and fidelity flags through termsafe.EncodeHiddenRunes, the canonical JSON-boundary encoder, so hidden and terminal-control runes arrive percent-encoded"
impact: internal
resolved_by:
  commit: "6aae3990"
---

scribe ingest --json renders refusals[].reason, refusals[].subject and the fidelity flags, which are scribe free text, raw: the text render neutralises them through termsafe and the JSON render leaves bidi and C1 controls in place (the iss-359 class)

## Grounds

- pursued: a refusal or flag carrying a bidi, C1 or zero-width rune renders in JSON with that rune percent-encoded; the raw rune in ingest --json output would show it wrong
