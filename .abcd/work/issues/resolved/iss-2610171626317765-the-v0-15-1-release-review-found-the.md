---
schema_version: 1
id: "iss-2610171626317765"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/06-capture.md:396-432 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
refines: [iss-2610151422055601]
resolution: "The page now states what the shipped code does, re-verified against the code and the verb's --help at v0.15.1 before the edit."
impact: internal
resolved_by:
  commit: "a394eb10a"
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/06-capture.md:396-432 out of step with the shipped code. It says: The frontmatter block 'per the issue-ledger schema in internal/core/issueschema' lists every field a record may carry; it has no finding_ref In the code: internal/core/issueschema/issueschema.go Known allow-list carries finding_ref (the consistency finding's cfk- key, written by the pass's filer); no brief chapter mentions it, only .abcd/work/issues/README.md:94
