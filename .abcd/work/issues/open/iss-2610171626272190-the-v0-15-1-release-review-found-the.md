---
schema_version: 1
id: "iss-2610171626272190"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Rewrite the passage at .abcd/development/brief/02-constraints/04-naming.md:49-51 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
---

The v0.15.1 release review found the brief at .abcd/development/brief/02-constraints/04-naming.md:49-51 out of step with the shipped code. It says: The namespace root refuses any positional that is not a record id, so `status` is not a registered command. In the code: `abcd status` is a registered top-level verb (root --help, Records group: 'List everyone's announcements, reservations and releases being cut'), with commands/status.md and .abcd/development/brief/04-surfaces/39-status.md.
