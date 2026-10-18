---
schema_version: 1
id: "iss-2610171626364475"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/39-status.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/39-status.md to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
resolution: "The page now states what the shipped code does, re-verified against the code and the verb's --help at v0.15.1 before the edit."
impact: internal
resolved_by:
  commit: "a394eb10a"
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/39-status.md 'What it shows' out of step with the shipped code. It says: lists announcements, doubtful branches, then the reservations block and the grants line In the code: abcd status --help also reports a live drain lane's stage and each holder's hand-backs from the last day; JSON also carries unread_names, branches_not_read, branches_left_to_read, announcements_not_read, not_dropped, local_tier_missing, read_error that the doc's key list omits
