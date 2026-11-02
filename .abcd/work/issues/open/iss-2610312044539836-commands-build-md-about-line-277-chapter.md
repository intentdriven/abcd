---
schema_version: 1
id: "iss-2610312044539836"
slug: "commands-build-md-about-line-277-chapter"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "landing PR #1301 (run-2610310228179894, lane-1)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/build.md"
remedy: "Name the flags the digest counts (intent-to-add, skip-worktree, assume-unchanged) and say the fsmonitor-valid bit is not counted, in commands/build.md, chapter 34 and the indexDigest comment."
---

commands/build.md (about line 277), chapter 34 of the brief and the indexDigest comment say the auto-plan's checkout digest covers every index entry with its flags, but the git ls-files -t -v listing it hashes leaves out the fsmonitor-valid bit, by design, since git rewrites that bit on every status.
