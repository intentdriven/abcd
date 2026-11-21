---
schema_version: 1
id: "iss-2611060809301279"
slug: "abcd-development-brief-05-internals"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Say ten blocks, add `status` to the list, and add it to the blocks this repository sets by hand."
resolution: "The configuration chapter already says ten blocks, lists status and names it among the blocks set by hand (prose corrected in 2854547b5); a test now holds that sentence's count, list and hand-set blocks to .abcd/config.json."
impact: internal
resolved_by:
  commit: "ed2337b57f1814ff3d55b59af32fa36cc22840e2"
---

`.abcd/development/brief/05-internals/03-configuration.md:241` says this repository's config carries nine blocks and lists them. `.abcd/config.json` carries ten: also `status`. The following sentence names only `facilitator` and `release` as set by hand, while :90-92 of the same chapter says `status.scheduled_workflow` is hand-set too.

## Grounds

- pursued: the chapter's sentence names exactly .abcd/config.json's top-level blocks and includes status among those set by hand; TestConfigurationChapterNamesThisRepositorysBlocks failing on the current tree, or a block added to the file without the sentence changing, would show it wrong
