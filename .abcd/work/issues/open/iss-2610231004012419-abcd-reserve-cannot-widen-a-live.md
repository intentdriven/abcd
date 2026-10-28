---
schema_version: 1
id: "iss-2610231004012419"
slug: "abcd-reserve-cannot-widen-a-live"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "hand-rolled coordination log, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/remote.go"
remedy: "Add reserve --add-footprint <path>, which widens a live reservation its caller already holds from any of that person's copies without moving the hold or touching its lease, and tell a sibling copy on this machine apart from another computer (same machine identity in the copy key), naming it as another of your copies here; test that a sibling worktree widens the footprint, the owning run's state is unchanged, and a copy on another machine is still refused"
---

abcd reserve cannot widen a live reservation from another copy of the same person on the same machine. A second worktree that reserves the same id with more footprint paths is refused as if the hold sat on another computer (internal/core/claim/remote.go:403, 'you are already building it on another computer; pass --take-over'), and a take-over moves the hold, which can disturb the run that owns it. Two sessions hit this on 2026-10-23 in the hand-rolled coordination log: abcd-54 at 10:04 could not add the worktree store's step 3 and 4 files to its live itd-2609091014076309 hold from a second copy, and borrowed an unrelated open record's reservation to announce them; abcd-f1 at 10:26 and 10:41 had its own hold read from a sibling worktree as another computer, and added the paths from the reserving worktree instead. Related: iss-2610161457209287 (holds do not say which of a person's copies has them).
