---
schema_version: 1
id: "iss-2609090722466403"
slug: "a-staged-raw-transcript-can-live-indefinitely-and-the-store"
severity: "critical"
category: "security"
source: "agent-finding"
found_during: "sub-agent transcript capture conceptual review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/staging.go"
resolution: "Staging gains four limbs. A drain now runs while a session is live, from the prompt hook, with a budget small enough not to delay a prompt and placed ahead of the rules work so an unparseable rules file cannot switch redaction off. An age limit buys priority and reporting rather than removal: an overdue entry sorts to the front of every drain and is named in the notices, and nothing deletes it, because a file is old precisely when nothing ran rather than when redaction was tried and failed. A backlog survey walks every repository in the store, so a pile in a repository nobody is standing in is reported at any session start, carrying counts and sizes only and never another repository's identifiers. And a deterministic redaction refusal is separated from a retryable failure and quarantined with its reason, so it stops being re-read every pass and stops being filed as awaiting redaction. Quarantine changes state, not exposure: those bytes stay on disk until a human discards them, which is the honest trade against never destroying the only copy."
impact: additive
resolved_by:
  intent: "itd-2609090559376002"
  spec: "spc-2609090624222051"
---

A staged raw transcript can live indefinitely, and the store's own comment says it cannot. Staging is the one place abcd holds unredacted transcript text on purpose, and its contract is that a staged file survives only until the next session starts. That holds only for a repository someone opens again. The drain runs from the session-start hook of the repository the staged file belongs to, so a repository that is finished with, or merely quiet, keeps its raw transcripts forever. On this machine right now there are four staged files totalling about thirteen megabytes of unredacted transcript, the oldest fourteen days old, and the per-repo status verb reports nothing from any other repository, so standing in one checkout cannot reveal a pile in another. A drain failure compounds it: the staged file is deliberately left in place, correctly, because deleting the only copy would be worse, but nothing ever retires it, so the population of permanently raw files only grows. Three things would close it: a drain that can run while a session is live rather than only at its start, a maximum staged age after which a file is redacted or deleted with a notice, and a cross-repository notice at session start so the pile in the repository nobody is standing in is still visible.

## Grounds

- pursued: we expect the pile to be caused by a drain that only ever runs at the owning repository's session start, so moving a small drain into the live prompt hook and surveying every repository at session start should empty it in normal use; it is shown wrong if transcripts still accumulate in a repository being worked in, if the per-prompt budget proves noticeable, or if the quarantined population grows, which would mean redaction is failing on real transcripts rather than the drain failing to run
