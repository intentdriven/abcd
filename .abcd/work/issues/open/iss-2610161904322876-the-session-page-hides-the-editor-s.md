---
schema_version: 1
id: "iss-2610161904322876"
slug: "the-session-page-hides-the-editor-s"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "design fix round, the Ctrl+{ switch"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/session/web/session.css"
remedy: "Ask the product thinker whether folding is wanted; if so show the fold gutter (fold arrows only, no line numbers), otherwise remove the fold keymap so the docs name no folding at all."
---

The session page hides the editor's gutter, so on Windows and Linux, where Ctrl+Shift+[ and ] now switch the page's design, there is no way left to fold a section of the brief.
