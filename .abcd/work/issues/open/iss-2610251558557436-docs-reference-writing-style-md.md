---
schema_version: 1
id: "iss-2610251558557436"
slug: "docs-reference-writing-style-md"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D3"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/reference/writing-style.md"
remedy: "Reword (a) to the practice (a rule is armed at blocker only on a corpus-clean tree, entering as a warn where the corpus cannot be cleaned in the arming change), (b) to say the bundled domain carries the short form for repositories without this page, and add persona_registry to (c); run abcd lint docs."
---

docs/reference/writing-style.md describes its own standing in three ways the repository does not bear out: (a) lines 21-23 say a new rule enters as a warn and is promoted once the corpus is clean, but punctuation/em-dash-in-list-item (fb5ac85ea) and roles/retired-role-word (b4df4f25d) were armed at blocker in the change that cleaned the corpus, and only link_anchors entered warn-first (internal/core/lint/linkanchors.go:16-21); (b) lines 3-5 say other surfaces point at the guide rather than restate it, but the bundled DOCUMENTATION rules domain restates four of its rules (internal/core/rules/defaults/rules.json); (c) lines 87-90 list the rules without a line escape and omit persona_registry, which reads none (internal/core/lint/persona.go:65-85).
