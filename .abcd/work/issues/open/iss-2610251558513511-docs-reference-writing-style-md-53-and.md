---
schema_version: 1
id: "iss-2610251558513511"
slug: "docs-reference-writing-style-md-53-and"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D3"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/reference/writing-style.md"
remedy: "Decide where a user-facing term of art is defined: give docs/ a glossary for abcd's own terms (rows in the crosswalk or a page of its own) and keep the rule, or reword both rows to link whichever page defines the term; prove it on the first page that follows the rule."
---

docs/reference/writing-style.md:53 and :55 tell a page to link a repo-specific term of art to its terminology.md entry, but terminology.md crosswalks established agentic-AI vocabulary and holds no entry for most of abcd's own terms (intent, spec, receipt, lifeboat); the one canonical glossary is .abcd/development/brief/glossary/ (its README:5-7), which is not user-facing. No docs page links terminology.md at a term's first use, so the rule cannot be followed as written.
