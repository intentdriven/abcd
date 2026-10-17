---
schema_version: 1
id: "iss-2610151422055601"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/06-capture.md:394 to match the shipped surface: internal/core/issueschema/issueschema.go:93 adds \"finding_ref\" to the Known allow-list (the key the consistency pass stamps on a filed finding so it is not filed twice); the brief's schema block omits it, and no brief chapter names finding_ref at all."
---

The brief at .abcd/development/brief/04-surfaces/06-capture.md:394 says: §3 'Ledger structure' presents the issue frontmatter 'per the issue-ledger schema in internal/core/issueschema' as a field-by-field list of what a record may carry (schema_version through resolved_by). The shipped surface differs: internal/core/issueschema/issueschema.go:93 adds "finding_ref" to the Known allow-list (the key the consistency pass stamps on a filed finding so it is not filed twice); the brief's schema block omits it, and no brief chapter names finding_ref at all. (v0.14.2 release crosscheck, undocumented-surface, raised by opus in a05.)
