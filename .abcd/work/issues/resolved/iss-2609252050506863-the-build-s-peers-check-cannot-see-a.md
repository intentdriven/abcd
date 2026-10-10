---
schema_version: 1
id: "iss-2609252050506863"
slug: "the-build-s-peers-check-cannot-see-a"
severity: "minor"
category: "architectural-insight"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "abcd build --session <id> claims the intent in the shared run state for a joined session when it creates the run, so another checkout's build of the same intent is refused at its peers check from the start; the session's own claim is not its peer. A build without --session holds no claim and now says so in its result."
impact: additive
resolved_by:
  commit: "bd6f6acdd"
---

The build's peers check cannot see a lane that has neither moved nor claimed the intent: the peer listing (peers.Report.Locate) sees a holding only when a sibling worktree or branch holds the intent in another bucket, and the claim half sees only a live claim in the shared run store, so a lane at steps 1 to 4 of any run (worktree made, brief written, implementer working, nothing committed that moves the intent) and a second clone of the repository are both invisible, and a second 'abcd build' of the same intent in another checkout starts a duplicate run. The other half of the same check, a peer named and not read being skipped silently while an unreadable claim refuses (fail-open on one side, fail-closed on the other), is iss-2609252049491342, fixed in the same lane. Closing this half needs Start to write a claim into the shared run store when it creates a run, which needs a session identity the host driver does not have yet.

## Grounds

- pursued: a second build of one intent from another checkout, the first started with --session, is refused as held; a duplicate run started past such a claim would show it wrong. A session-less build stays invisible by design and is named as such.
