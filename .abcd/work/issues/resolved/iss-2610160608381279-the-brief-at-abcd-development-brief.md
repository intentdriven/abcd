---
schema_version: 1
id: "iss-2610160608381279"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Change 'Three audit-loop modes' to 'Two' at 05-intent.md:569, or name the third mode."
resolution: "05-intent.md now counts two audit-loop modes, matching the two it lists (record-only, loop-to-acceptance)"
impact: internal
---

The brief at .abcd/development/brief/04-surfaces/05-intent.md:569 says 'Three audit-loop modes, facilitator-elected per intent via the audit_mode frontmatter key', but lists only two beneath it: record-only and loop-to-acceptance.
