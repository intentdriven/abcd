---
schema_version: 1
id: "iss-2611252133341605"
slug: "status-json-prints-a-zero-lapsed-at"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "planning of 25 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/release.go"
remedy: "Use omitzero, or a pointer as the sibling record in claim does, and test that an ended release with no lapse prints no lapsed_at."
---

abcd status --json prints a lapsed_at of 0001-01-01T00:00:00Z for a release withdrawal that has not lapsed: the field is a time with omitempty, which never drops a zero time. A reader of the JSON takes it for a lapse at that instant. Run on 2026-11-25.
