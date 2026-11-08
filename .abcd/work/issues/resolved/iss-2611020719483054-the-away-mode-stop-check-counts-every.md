---
schema_version: 1
id: "iss-2611020719483054"
slug: "the-away-mode-stop-check-counts-every"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "parallel sessions in sibling worktrees, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Have the stop check's owed-work read count only the runs and lanes the stopping session drives (by its session claim, or the worktree it stands in), and name another session's live run as theirs rather than owed here."
resolution: "The owed read takes the stopping session's id from the Stop payload and leaves out what another live session holds, naming each holder once: the lanes of a run whose control record names that session as its driver, a branch the guard recorded it creating, a branch checked out in the worktree it started in, and a reservation only its runs are for. A session's own run's lanes and reservation stay owed. A lane of a run no session drives stays owed unless a live session (a drive's stage agent, say) works in that lane's worktree, when it is that session's. A peer's branch still serves as the base of this session's branch stacked on it, so the clearing rule still judges that branch."
impact: fix
resolved_by:
  commit: "05dfe5816dda15075cfc3980163dee867eb82f5a"
---

The away-mode stop check counts every live run's lanes as this checkout's own: with several sessions building in sibling worktrees of one repository, a session's turn end was refused more than fifty times on 2026-11-01 for lanes other sessions owned on their own branches, because the owed-work read takes every run the checkout's local tier lists rather than the runs this session drives.

## Grounds

- pursued: a session in a copy a live peer shares is no longer held at its stop for the peer's run lanes (TestPeerSessionWorkNamedNotOwed), while its own run's lane and reservation still block (TestOwnWorkStaysOwedBesidePeer) and its branch stacked on a peer's is still owed by the clearing rule (TestStackedOnPeerBranchStaysStacked, which failed before this round); shown wrong if a session's own lane, renewal or stacked branch ever drops out of its owed set, or a live peer's run lane is still named as owed here.
