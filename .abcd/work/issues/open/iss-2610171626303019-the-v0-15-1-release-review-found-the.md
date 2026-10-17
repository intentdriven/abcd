---
schema_version: 1
id: "iss-2610171626303019"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/08-abcd.md:523 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/08-abcd.md:523 out of step with the shipped code. It says: In the itd-20 design-target note: 'There is no alias routing either: `status` is refused and `help` prints the verb list'; line 590 adds 'Nothing routes them today'. In the code: `status` is a shipped top-level verb (bin/abcd-darwin-arm64 status --help: 'List everyone's announcements, reservations and releases being cut'), and the chapter itself says so at 'abcd status is its own verb' (39-status.md); it is not refused.
