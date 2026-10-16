---
id: dcn-2608080000000002
slug: bug-hunt-loop-round-9-state-issue-197-a
date: 2026-08-08
---
2026-08-08 — bug-hunt loop round 9 (state issue #197): a multi-angle sweep (5
blind hunt angles plus independent adversarial verification) surfaced a
complete, silent bypass of every guard blocker via `env -S`/`--split-string`
(GNU env re-splits and runs that value as the command; the guard's wrapper
table treated it as the wrapper's own value and stepped past it). A fix
attempt (branch `bugfix/env-split-string-guard-bypass`, closing the
separate-token spelling `env -S git push ...`) was BLOCKed at pre-PR review by
two independent reviewers (correctness and security), who each independently
found the fix incomplete: the glued spellings `env -Sgit ...` and
`env --split-string=git ...` execute identically to the closed form and
remained a live bypass. Per protocol this is a first-attempt BLOCK, not a
repeat of the unrelated iss-189/190 scanner root cause, so it stops here
rather than escalating — nothing from that branch was pushed. The complete bug
(all three spellings, plus the reviewers' converging suggested repair —
handle env's `-S<value>`/`--split-string=<value>` forms alongside the
separate-token form in `commandOf`/`skipWrapperArgs`, no nested re-tokenizing
needed) is captured fresh as iss-200, since iss-196 (the narrower, incomplete
framing) only ever existed on the abandoned, unpushed branch and was never
merged; treat iss-196 as superseded by iss-200 the way iss-191 was re-scoped
into iss-195.

The same sweep also confirmed three further bugs, captured here record-only
(no behaviour changes in this commit): iss-201 (`abcd guard hook`'s stdin read
has no overflow check, so a payload padded past the 1 MiB cap silently
truncates, fails JSON parsing, and fail-opens with a diagnostic that
misleadingly blames the host instead of reporting the cap overflow — major,
not critical, since the practical exploitation cost is a single tool_input
exceeding roughly 1 MiB of model-generated content, and the fail-open is loud,
never silent); iss-202 (`scanner.New` reads `.abcd/config/pii.json` with a
bare, unguarded `os.ReadFile` — no symlink guard, no size cap — unlike every
sibling trust-boundary config reader in this codebase; a FIFO hangs it forever
and a git-committable symlink to a device file grows it toward OOM, both
reachable automatically via the SessionEnd hook, and the hang also wedges
`history.repoLock`'s unbounded flock, permanently disabling transcript
capture for that repo — the same bug class already fixed once under iss-97,
which this instance escaped); and iss-203 (`abcd audit`'s privacy rule guards
on `scanner.New`'s error return expecting a fallback to the built-in pattern
set, but `scanner.New` never returns a non-nil error, so the guard is always
true, the documented fallback is dead code, and a broken `pii.json` override
silently drops a repo's raised severities and downgrades the audit exit code
from 2 to 1 with no diagnostic).

Also reconfirmed this round (captured round 7 as iss-195, previously flagged
as needing adversarial verification before fix-eligibility): the scanner's
rigid/open-ended heuristic false-positive `hard_fail`-flags RFC 3849 reserved
documentation addresses and silently corrupts them via `Redact`, on top of
the previously-known cost regression (up to ~54x on colon-hex content, worse
than the ledger's original 1.9x-27x table) — worse than its `minor` label
suggests. Its own fix is out of scope for this round, but iss-195 is a
distinct bug in the scanner's rigid/open-ended heuristic, not part of the
shelved scanner adjacency-recovery mechanism (iss-189/190/191, shelved per
the maintainer's 2026-08-07 ruling); iss-195 itself remains fix-eligible, not
shelved. No ledger or code change made for iss-195 this round beyond this
reconfirmation note.
