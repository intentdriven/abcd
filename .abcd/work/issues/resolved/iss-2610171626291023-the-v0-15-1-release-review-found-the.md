---
schema_version: 1
id: "iss-2610171626291023"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/08-abcd.md:232-259 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
resolution: "the peers reservations passage now names the release-tag fetch and its refs, the release-by-tag state, the line per release being cut and release_holds"
impact: internal
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/08-abcd.md:232-259 out of step with the shipped code. It says: abcd peers' one network act is a fetch from origin of every abcd-reserved/ branch (pruned) and of the default branch, which updates the remote-tracking reservation branches and stamps when it ran; the listing is reservations, per-person li… In the code: abcd peers --help (and commands/peers.md:15-19) says the fetch also reads origin's release tags v* and writes them under refs/abcd/release-tags/, and the listing adds a line for each release being cut (a live rel-vX.Y.Z reservation). The fetch and refs are documented in 37-reserve.md:315, but this chapter's enumeratio…
