---
schema_version: 1
id: "iss-2609252120215011"
slug: "the-guard-brief-scopes-a-bare-variable-to-a-payload-alone"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
---

Brief chapter 17-guard.md scopes a bare variable to inside a payload in what an allow still does not see, though a variable in the top-level command position or where a flag would be is not read either, so its claim that the obvious evasions are not evasions overclaims; and the cost guard comment states bounded shapes measure one to five units of work per byte, below what bounded shapes measure (review4-guard findings 6 and 7).
