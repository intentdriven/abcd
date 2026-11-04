---
schema_version: 1
id: "iss-2611012144422634"
slug: "a-planning-interview-the-product-thinker"
severity: "major"
category: "bug"
source: "manual-test"
found_during: "the product thinker's planning interview of itd-2610262140595217, 2026-11-01 21:41"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview"
remedy: "Scope a running interview's tripwire to what its own contract may change and what it reads: the refs of its own checkout's branch and the records it edits, never every branch ref of the repository's shared .git; a commit on another worktree's branch is not the interview's concern. Where a change the interview must not ignore does happen, tell the person in plain words what changed and offer to continue, never stop silently before the first question."
deferred_after: "v0.16.0"
deferral_reason: "Deferred by the product thinker on 2026-11-03 for this cut, v0.16.1 as the records derive it: scoping the interview's tripwire is a few hours' work that would hold the cut past tonight."
---

A planning interview the product thinker started at 21:41 on 2026-11-01 stopped before its first question, with its answers record written as answers: null. Its tripwire fired on refs/heads/records/api-route-lessons and refs/heads/records/planning-1101, branches two other sessions were committing to in their own worktrees. Every worktree of a checkout shares one .git, so the interview watches every branch ref, and any peer's ordinary commit anywhere in the repository ends the person's interview. The peers had to hold every git ref change for half an hour so the person could retry. With several sessions working one repository, as abcd's own development does, an interview can hardly run at all.
