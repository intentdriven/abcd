---
schema_version: 1
id: "iss-2609100519122086"
slug: "which-session-holds-which-worktree-branch-or-record-is-coord"
severity: "major"
category: "future-work-seed"
source: "agent-finding"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work"
---

Which session holds which worktree, branch or record is coordinated entirely by conversation, so every new session repeats a handshake that nothing records. A session joining work in progress has no way to ask what is already claimed: it messages the peers it can see, waits for replies, and rebuilds a picture that the sessions before it had already built and did not write down. One measured encounter cost four messages and about fifteen minutes before any work began, and the picture it produced is not durable, so the session after that pays again. The convention that a diff you did not make is a peer's work depends on knowing who the peers are and what they hold, which is precisely the thing no artefact carries. The repository already records this gap for the narrow case of detecting a peer session before mutating git state; the wider case is claim rather than presence, and the two want the same substrate. Whatever holds it should be as cheap to write as it is to read, because a coordination record nobody updates is worse than the chat it replaced.
