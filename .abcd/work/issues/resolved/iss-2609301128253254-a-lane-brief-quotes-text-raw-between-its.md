---
schema_version: 1
id: "iss-2609301128253254"
slug: "a-lane-brief-quotes-text-raw-between-its"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/issuebrief.go"
remedy: "Quote every fenced text in both briefs through one helper that writes an HTML comment opener or closer with its second hyphen as the entity &#45;, and say so once in the brief; test a remedy carrying an end-remedy marker stays inside its fence."
resolution: "Every fenced quote in both lane briefs goes through one helper that escapes comment markers, so quoted text cannot close its fence."
impact: fix
resolved_by:
  commit: "b6b7670f02c68750f6f1aef40c01ce72a63e039b"
---

A lane brief quotes text raw between its begin/end comment fences: internal/core/implement/loop/issuebrief.go writes the remedy and the issue record, and brief.go the intent, the spec and the conventions, unescaped, so quoted text carrying an end marker closes its fence early and what follows reads as the brief's own words (review-drainLoop finding 2).

## Grounds

- pursued: a remedy carrying an end-remedy marker stays inside its fence and each brief carries only its own six markers; a marker surviving in any quote would show it wrong (TestQuotedTextCannotCloseItsFence, TestTheIntentBriefQuotesThroughTheSameFence, TestFenceQuoteWritesNoMarker).
