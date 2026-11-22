---
schema_version: 1
id: "iss-2611220942419392"
slug: "github-clears-a-pull-request-s"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous fix run 2026-11-21/22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing"
remedy: "Have abcd land (bare, on a landed branch) or abcd peers report the merge-queue entry's state and position and, when the pull request left the queue, the removal event and its reason, read from the forge timeline, so a heartbeat tells queued from dropped with one command."
---

GitHub clears a pull request's auto-merge request when the merge queue takes it, so a run that checks the armed state reads a queued pull request as dropped. On 22 Nov 2026 an autonomous run had to query the forge's GraphQL timeline by hand to tell that its pull request was second in the queue, not removed.
