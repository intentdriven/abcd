---
schema_version: 1
id: "iss-2610150827113191"
slug: "the-build-loop-s-landmerged-proves-a"
severity: "minor"
category: "process"
source: "impl-review"
found_during: "abcd land review round 3, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Have the build loop's landMerged ask landing.MergedAt (same-repository, same head, default base) when the ancestry check fails, and report a squash- or rebase-merged lane landed and clean it up, as abcd land does, instead of refusing and leaving the cleanup to a person"
---

The build loop's landMerged proves a lane landed only by ancestry; after a squash or rebase merge it refuses and hands the worktree and branch cleanup to a person, though abcd land now proves the same case through the forge (landing.MergedAt: a merged, same-repository pull request at the gated head into the default branch). The two landings disagree about one fact, and the loop's refusal costs a person a manual cleanup on every squash-merged lane. Found by the round 3 reviewers of the abcd land change, 2026-10-15.
