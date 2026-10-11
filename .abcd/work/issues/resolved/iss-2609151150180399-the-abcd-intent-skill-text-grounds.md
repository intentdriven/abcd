---
schema_version: 1
id: "iss-2609151150180399"
slug: "the-abcd-intent-skill-text-grounds"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "peer session report 2026-09-15 (a teaching-repo session planning five intents)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/intent.md"
resolution: "commands/intent.md says grounds.redacted is omitted when zero, matching the omitempty convention every write verb's redacted count follows"
impact: fix
resolved_by:
  commit: "07af2eb092552739a5360c39e5659457e7b3c734"
---

The /abcd:intent skill text (Grounds section) shows the ready --grounds --json envelope carrying "redacted": 0 and asks the host to report grounds.redacted whenever it is non-zero, but GroundsResult.Redacted (internal/core/intent/grounds.go) is tagged omitempty, so the key is absent whenever the count is zero. A host following the page finds no key at all on every ordinary write (five of five in the reporting session, v0.8.0 plugin binary) and cannot tell an omitted zero from a missing field. Either the skill text says the key is omitted when zero, or the envelope carries it always; the envelope's own convention for counts elsewhere (json_empty_collections_test) should decide which.

## Grounds

- pursued: a host reading the page expects no key on an unredacted write; an envelope that emits redacted: 0 would show it wrong
