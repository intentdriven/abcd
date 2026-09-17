---
schema_version: 1
id: "iss-2609151150180399"
slug: "the-abcd-intent-skill-text-grounds-section-shows-the-ready-g"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "peer session report 2026-09-15 (a teaching-repo session planning five intents)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/intent.md"
---

The /abcd:intent skill text (Grounds section) shows the ready --grounds --json envelope carrying "redacted": 0 and asks the host to report grounds.redacted whenever it is non-zero, but GroundsResult.Redacted (internal/core/intent/grounds.go) is tagged omitempty, so the key is absent whenever the count is zero. A host following the page finds no key at all on every ordinary write (five of five in the reporting session, v0.8.0 plugin binary) and cannot tell an omitted zero from a missing field. Either the skill text says the key is omitted when zero, or the envelope carries it always; the envelope's own convention for counts elsewhere (json_empty_collections_test) should decide which.
