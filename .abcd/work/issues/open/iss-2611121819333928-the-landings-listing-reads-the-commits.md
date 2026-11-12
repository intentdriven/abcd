---
schema_version: 1
id: "iss-2611121819333928"
slug: "the-landings-listing-reads-the-commits"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "intent audit of itd-2610301850454860"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landings/landings.go"
remedy: "State the reading on the landings command page and in the brief chapter, and decide in a planned step whether the forge's commit-to-pull-request listing is read for such commits."
---

The landings listing reads the commits of a rebase-merged pull request as pushes: each is a first-parent commit whose message names no pull request, so it is listed as a push with no lane and no rule, unattended unless an approval of that commit is recorded, and the pull request that brought it is not named. Neither the command page nor the brief chapter says so. The repository allows rebase merges.
