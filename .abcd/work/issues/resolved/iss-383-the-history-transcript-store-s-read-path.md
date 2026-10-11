---
schema_version: 1
id: "iss-383"
slug: "the-history-transcript-store-s-read-path"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "internal/core/history/store.go"
resolution: "route listRecords and Read through fsutil.ReadGuarded; watched-fail TestListSkipsSymlinkedRecord"
impact: fix
---

the history transcript store's read path is unguarded — listRecords and Read use raw os.ReadFile, so a planted FIFO wedges history list and history capture while the store flock is held, and a symlinked record is followed
## Evidence

- `internal/core/history/store.go:264` (`listRecords`) and `internal/core/history/history.go:232` (`Read`) use raw `os.ReadFile` over `~/.abcd/history/<rootSHA>/transcripts/*.md` — no `O_NOFOLLOW`, no `O_NONBLOCK`, no regular-file check, no size cap — while the write path is fully hardened (`ownedDirsReal`, `repoLock` with `O_NOFOLLOW` at `store.go:83-84`, `WriteFileAtomic`, and the iss-347 `ReadGuarded` hook read).
- Reproduced: a FIFO `hang.md` in the transcripts dir wedges `abcd history list` (exit 124 under timeout) and — worse — `history capture`, which calls `listRecords` at `history.go:117` **while holding the store flock** taken at `history.go:102`, the same lock-held-hang shape that made iss-202 critical. A symlinked `*.md` is followed, though disclosure does not reproduce (unparseable files are skipped by `parseRecord`), so availability is the confirmed hazard.
- Refuter verdict: CONFIRMED (minor, security/hardening) — the same-identity HOME-store severity band matches resolved iss-347's adjudication; the missing git-committable delivery vector (unlike iss-202's `pii.json`) is what caps it at minor. Remedy: route both reads through `fsutil.ReadGuarded` with a size cap, keeping `listRecords`' per-file skip tolerance.
