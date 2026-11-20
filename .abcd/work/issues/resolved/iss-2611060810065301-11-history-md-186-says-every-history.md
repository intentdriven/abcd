---
schema_version: 1
id: "iss-2611060810065301"
slug: "11-history-md-186-says-every-history"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "Name the two forms that skip the seam and say they create nothing."
resolution: "The history chapter now limits the resolve-seam claim to verbs that name a repository. It names the two forms that skip the seam, the all-repositories form of the staged listing and the discard of a pending transcript, and says they create nothing and move nothing."
impact: internal
resolved_by:
  commit: "1a178b0efe768c837bb3494902277156aa5747d4"
---

`11-history.md:186` says every history verb reaches the store through one resolve seam, which creates the store chain and moves a legacy corpus. `abcd history staged --all-repos` (`internal/surface/cli/history.go:215`) and `abcd history discard --pending` (`internal/surface/cli/history.go:526`) skip it: on fresh homes each left the home directory empty, while `abcd history list` created the chain.

## Grounds

- pursued: on a fresh home, staged --all-repos and discard --pending leave the home empty while history list creates the chain, and the chapter says so. TestHistoryFormsThatSkipTheSeamCreateNothing would show it wrong by finding either form writing to the home, or the chapter missing the sentence.
