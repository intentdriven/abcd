---
schema_version: 1
id: "iss-2609230720193756"
slug: "the-implement-run-state-has-two-same-uid"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
resolution: "AppendLineIn refuses a symlinked or non-regular leaf; readClaim treats a claim whose session or lane is not a name as unreadable; lock files are created 0600; the command page and brief say the bounds rest on a cooperative, unauthenticated role."
impact: fix
resolved_by:
  commit: "f85d271ac"
---

The implement run state has two same-uid hygiene gaps the security review of itd-2609221656373558 rated LOW and shipped past on the record: fsutil.AppendLineIn's openAppendIn (internal/fsutil/fsutil.go) opens the log leaf without the Lstat and symlink refusal its read twin ReadGuardedInRoot applies, so an in-root symlink leaf redirects every appended line (demonstrated: a log symlinked onto a claim file makes the claim unreadable); and readClaim (internal/core/implement/claim.go) checks a claim file's session and lane only for non-empty before they reach HeldError and the release refusal, printed to stderr unsanitised, so a hand-edited claim can put terminal escapes on the operator's screen. Remedy, each test-held: refuse a symlink leaf in openAppendIn as the read primitive does, and run validName on the claim's session and lane in readClaim treating failure as unreadable. Also: .lock is created 0644 beside 0600 run files (internal/fsutil/flock.go), and commands/implement.md does not say the release refusal rests on a cooperative, unauthenticated role. The next lane that extends internal/core/implement (the build verb, itd-2609201916151817) takes it.

## Grounds

- pursued: an in-root symlink leaf can no longer redirect an appended log line and a hand-edited claim can no longer put escapes on the operator's screen; a log append landing in a claim file, or an escape in a HeldError, would show it wrong.
