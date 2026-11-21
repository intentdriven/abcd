---
schema_version: 1
id: "iss-2610262037128325"
slug: "the-release-glossary-accepts-snapshot-as"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 706 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/distribution/release.md:5"
remedy: "Remove the bare 'snapshot' alias from release.md (keep 'published snapshot'), so snapshot is not an accepted name for both a release and a lab's pinned clone."
finding_ref: cfk-51df57a6afc4f748
resolution: "Removed the bare 'snapshot' alias from the release glossary entry, keeping 'published snapshot', so the word no longer names both a release and a lab's pinned clone; a repolint test pins the alias list."
impact: internal
resolved_by:
  commit: "59ebbd69d958ed9f3c25c409db05bbdb38e342a1"
---

The release glossary accepts snapshot as an alias for a release; the lab chapter defines a snapshot as a standalone clone pinned at one commit.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/distribution/release.md:5` — "aliases: ["published snapshot", "plugin release", "snapshot"]"
- End B: `.abcd/development/brief/04-surfaces/31-lab.md:48` — "lays down a **snapshot**: a standalone clone of the repository detached at the pin"

release.md lists snapshot among a release's aliases, so a snapshot is a published, version-tagged cut of abcd. 31-lab.md defines a lab's snapshot as a standalone clone of the repository detached at the pin. The word is an accepted name for two different things.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 706.

## Grounds

- pursued: release.md's aliases no longer accept bare 'snapshot' while 'published snapshot' stays; TestTheReleaseEntryDoesNotAcceptBareSnapshot would fail if either changed back
