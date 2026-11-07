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
---

`11-history.md:186` says every history verb reaches the store through one resolve seam, which creates the store chain and moves a legacy corpus. `abcd history staged --all-repos` (`internal/surface/cli/history.go:215`) and `abcd history discard --pending` (`internal/surface/cli/history.go:526`) skip it: on fresh homes each left the home directory empty, while `abcd history list` created the chain.
