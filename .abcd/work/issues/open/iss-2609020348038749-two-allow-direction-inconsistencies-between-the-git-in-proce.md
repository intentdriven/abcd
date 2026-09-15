---
schema_version: 1
id: "iss-2609020348038749"
slug: "two-allow-direction-inconsistencies-between-the-git-in-proce"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/gitconfig.go"
---

Two allow-direction inconsistencies between the git in-process alias pre-pass and the rest of the guard matcher, surfaced by the security review of the shell-guard advisory branch. (1) expandGitAliases in internal/core/guard/gitconfig.go compared the command word with `git` literally, while matchSegment compares a globbed command word as the pattern it is, so a glob-spelled git built no alias rewrite and the declared alias went unread. (2) A bang-prefixed alias body is handed to shellInspect and then to expandPayloads, but the resulting segments never go back through expandGitAliases, so an alias declared inside a bang body is not resolved and its rewrite is never checked. Half (1) is closed in the same change that records this, pinned by TestGlobSpelledGitReachesTheAliasPrePass; half (2) stays open because re-entering the pre-pass needs a recursion depth budget and a repeat guard of its own, which is a shape to choose rather than a line to change. Both are allow-direction: the guard says nothing where the matcher's own reading elsewhere would have said something.
