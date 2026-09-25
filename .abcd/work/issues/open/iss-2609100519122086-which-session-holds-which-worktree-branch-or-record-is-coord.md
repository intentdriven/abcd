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
related_intents: [itd-2609150819440345]
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Session register transport: the code host, a per-machine helper, or both?"
---

Which session holds which worktree, branch or record is coordinated entirely by conversation, so every new session repeats a handshake that nothing records. A session joining work in progress has no way to ask what is already claimed: it messages the peers it can see, waits for replies, and rebuilds a picture that the sessions before it had already built and did not write down. One measured encounter cost four messages and about fifteen minutes before any work began, and the picture it produced is not durable, so the session after that pays again. The convention that a diff you did not make is a peer's work depends on knowing who the peers are and what they hold, which is precisely the thing no artefact carries. The repository already records this gap for the narrow case of detecting a peer session before mutating git state; the wider case is claim rather than presence, and the two want the same substrate. Whatever holds it should be as cheap to write as it is to read, because a coordination record nobody updates is worse than the chat it replaced.

Corroborated 2026-09-12, from inside this repository rather than a managed one, and more sharply than the original evidence. Two agents working the same checkout each reported that a peer session was editing it, each listed the other's files accurately, and each correctly declined to touch them. Neither was a peer: they were each other, plus a third agent of the same run. Both were reduced to judging their gates on a clean clone made outside the checkout, because the shared tree was transiently broken by work that was not theirs and they had no way to tell whether it would be fixed.

The original evidence was a session paying four messages and about fifteen minutes to rebuild the picture by conversation. This is worse in one respect and better in another. Worse: there was nobody to ask, so the picture could not be rebuilt at all, and both agents inferred a foreign session from file timestamps, right about the files and wrong about who held them. Better: the convention held anyway. Each one left the other's work untouched on the strength of the rule alone. That is the argument for the substrate rather than against it: restraint worked, and it cost two full verification clones and a wrong belief about who else was in the tree.

## Grounds

- pursued: we expect a claim record keyed on the root-commit SHA beside the worktree store to remove the handshake, because the worktree store is already the machine-scoped place a session's lane lives and a claim is one more fact about that lane; it is shown wrong if claims go stale faster than sessions release them, in which case a record nobody updates is worse than the conversation it replaced
