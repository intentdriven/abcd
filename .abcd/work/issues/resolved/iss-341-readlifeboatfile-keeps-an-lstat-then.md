---
schema_version: 1
id: "iss-341"
slug: "readlifeboatfile-keeps-an-lstat-then"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/core/lifeboat/embark.go"
resolution: "readLifeboatFile routes through fsutil.ReadGuardedInRoot (O_NONBLOCK + fstat-on-fd + SameFile), keeping the Lstat symlink message; unused abs param dropped (test TestReadLifeboatFileDoesNotBlockOnFifo)"
impact: fix
---

readLifeboatFile keeps an Lstat then root.Open window over an untrusted lifeboat (embark), bypassing fsutil.ReadGuardedInRoot — a vetted regular file swapped for a FIFO blocks the open forever, or an in-root symlink is read/hashed in place of the vetted file

## Evidence

- `internal/core/lifeboat/embark.go:563-589` -- `root.Lstat(rel)` then a separate `root.Open(rel)`, no `O_NONBLOCK`, no `os.SameFile` fd re-check. Callers: `VerifyManifest` (`embark.go:146`), the planner (`embark.go:246`), `synthesis_review.go:105`.
- `internal/fsutil/fsutil.go:92-133` `ReadGuardedInRoot` -- canonical primitive naming both dropped guarantees; `probe.go:226-247` `SourceContext.ReadFile` is the hardened sibling.

## Refuter verdict -- CONFIRMED (substantive)

On the pinned go1.25.6, `root.Open` follows in-root symlinks and blocks on a FIFO. The lifeboat dir is documented untrusted (`embark.go:4-8`). Incremental hazard is the FIFO hang; a content swap inside VerifyManifest fails closed on hash mismatch. Same shape as the burst-2 readGraveyardFile fix, invisible to a `Lstat`->`ReadFile` grep. Fix: route through `fsutil.ReadGuardedInRoot`, drop the unused `abs` param, keep the Lstat symlink check for its distinct message.
