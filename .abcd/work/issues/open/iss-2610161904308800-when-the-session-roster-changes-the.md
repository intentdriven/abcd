---
schema_version: 1
id: "iss-2610161904308800"
slug: "when-the-session-roster-changes-the"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "design fix round"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/session/web/session.js"
remedy: "Update the invite card's parts in place instead of rebuilding them, or restore focus to the same control after the rebuild; test with a roster change while a card button has focus."
---

When the session roster changes, the invite card's contents are rebuilt, so a focused button inside the open invite card loses keyboard focus.
