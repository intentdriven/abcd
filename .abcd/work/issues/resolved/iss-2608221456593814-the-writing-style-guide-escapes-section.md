---
schema_version: 1
id: "iss-2608221456593814"
slug: "the-writing-style-guide-escapes-section"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: "docs/reference/writing-style.md"
resolution: "Escapes section scoped to the banned-token families; names/* row added to the surface table."
impact: fix
resolved_by:
  commit: "be8b924"
---

The writing-style guide Escapes section says any machine-enforced rule accepts a docs-lint: allow line, but only the banned-token families honour it; links_resolve, stray_root_docs and citation_* ignore it, so a contributor adds a no-op escape and still fails the gate. The Structure table also omits the shipped names/* blocker family.