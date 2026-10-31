---
schema_version: 1
id: "iss-2610312044536324"
slug: "commands-build-md-about-line-281-and"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "landing PR #1301 (run-2610310228179894, lane-1)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/build.md"
remedy: "Say that such an index is not used and the lane is handed back, rather than that it is not read, in both places and in the indexDigest comment."
---

commands/build.md (about line 281) and chapter 34 of the brief (about line 1012) say an index past 1 GiB is not read, but indexDigest reads up to 1 GiB and one byte through its LimitReader before it refuses the copy.
