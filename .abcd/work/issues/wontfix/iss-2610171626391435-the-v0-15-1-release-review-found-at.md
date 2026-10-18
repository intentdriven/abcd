---
schema_version: 1
id: "iss-2610171626391435"
slug: "the-v0-15-1-release-review-found-at"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Rewrite the passage at CHANGELOG.md:54-56,61 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
wontfix_reason: "The finding is true, but a released changelog section is not rewritten: CHANGELOG.md has one writer, the ship path, which only inserts a new section under an empty Unreleased (principles/one-writer-per-file.md), and the v0.15.1 GitHub release body does not come from the file, so a hand edit would move only main's copy of a section whose tag and archive cannot change. The cause, the composer wording a fix to a surface new since the anchor as a fix, is iss-2610172148157726, so later cuts do not repeat it."
---

The v0.15.1 release review found at CHANGELOG.md:54-56,61: Four Fixed lines are written as fixes to behaviour no earlier release had. 'A status fetch reads an app's push ... no longer asked about again' (iss-2610161928560772) concerns `abcd status`, which did not exist at v0.15.0. The three cloud lines ('no longer reported as started', 'no longer reports a refused commit', and the author identity line) concern scripts/cloud, which `git ls-tree v0.15.0 -- scripts/cloud` shows did not exist then either. A reader infers the features shipped earlier. The 'no longer' wording also contradicts the section's own preamb…

## Grounds

- declined: The finding is true, but a released changelog section is not rewritten: CHANGELOG.md has one writer, the ship path, which only inserts a new section under an empty Unreleased (principles/one-writer-per-file.md), and the v0.15.1 GitHub release body does not come from the file, so a hand edit would move only main's copy of a section whose tag and archive cannot change. The cause, the composer wording a fix to a surface new since the anchor as a fix, is iss-2610172148157726, so later cuts do not repeat it.
