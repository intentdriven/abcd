---
schema_version: 1
id: "iss-2611251238397006"
slug: "merge-gate-does-not-read-the-reviewers"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "code reading of the landing gate, 2026-11-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/facilitator.go"
remedy: "Have the merge gate read the reviewer's account type from the reviews answer it already requests, and count a review only when the type is `User`, as the landings reader does, keeping the name-shape check, so that an app, a bot, and an absent or unread type each count for nothing; treat a review whose type is read and is not `User` as a read answer that counts for nothing, not as an unread one, and leave any other counting review to count; pin the field in the REST contract and add a test that names each case, using the type an app's review carries on that call."
---

Two readers of the same forge reviews disagree on which reviewer counts. The landings history reader (`internal/core/landings/forge.go`) reads each reviewer's account type and counts a review only when the type is `User`, and its tests name a bot, an app's account and a login in a bot's shape. The merge gate (`countedApproval` in `internal/core/landing/facilitator.go`) reads the login alone: it counts an approving review of the head from any login other than the author that has write access, for a maintainer's pull request in human mode and for a contributor's in either mode. An app's review is not counted at the gate, but as a side effect and by no rule of its own: the name-shape check that runs before the permission read (`ownerName` in `internal/core/landing/ruleset.go`) refuses a login with a bracketed suffix, which an app's login usually carries, and the reviewer then reads as unverified, which, where no other approval counts, marks the gate's answer as unread and not as a refusal. No test in `internal/core/landing` covers a review from an app, what the gate answers for an app whose login passes that check is not tested, and the account type is not in the pinned REST contract (`internal/ghcontract/rest-fields.txt`). This record is not about an agent-driven account of type `User`, which both readers count as a person: dcn-2610300955154851 already rules that human mode's approval is a rule agents keep and not a proof of who approved, and `commands/land.md` says so. Assessed with the question in `.abcd/development/principles/abcd-fixes-what-it-adds.md`, this is a gap in a check abcd adds and not a danger abcd adds: a repository without abcd has no gate reading approvals before a merge is armed. Read at 383208fc0.
