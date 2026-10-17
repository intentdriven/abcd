---
schema_version: 1
id: "iss-2610171626380056"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Rewrite the passage at .abcd/development/brief/06-delivery/02-verification-matrix.md to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
---

The v0.15.1 release review found the brief at .abcd/development/brief/06-delivery/02-verification-matrix.md row 'Capture resolve' out of step with the shipped code. It says: 'the optional intent, spec, commit and release references write the structured provenance' In the code: abcd capture resolve --help has --intent, --spec, --commit and --shipped-in (a migration-only release stamp); there is no release-reference flag by that role, and --production-mode is undocumented in the row
