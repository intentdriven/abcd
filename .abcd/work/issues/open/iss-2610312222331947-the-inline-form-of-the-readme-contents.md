---
schema_version: 1
id: "iss-2610312222331947"
slug: "the-inline-form-of-the-readme-contents"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "ruthless review of run-2610312001568919 lane-1 (iss-2610311544004033)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_derive.go"
remedy: "Render a heading's emphasis in the inline contents as the page would (em and strong tags, escaped text), or strip the markers, and test a heading carrying each kind of emphasis."
---

The inline form of the README contents part shows Markdown emphasis as typed. A level-two heading such as 'Why *abcd* exists' becomes a link inside an HTML block, where nothing is rendered, so the reader sees the asterisks; the block form of the same heading shows the word in italics. The plain-text step decodes character references and drops backslash escapes but keeps emphasis markers.
