---
schema_version: 1
id: "iss-342"
slug: "sourcecontext-listdir-opens-directory"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/core/lifeboat/probe.go"
resolution: "SourceContext.ListDir now opens with O_RDONLY|nonBlock so a planted FIFO returns promptly instead of hanging probe/plan/pack (test TestListDirDoesNotBlockOnFifo)"
impact: fix
---

SourceContext.ListDir opens directory entries with root.Open (no O_NONBLOCK), so a statically-planted FIFO named at a path a probe adapter lists (e.g. docs) hangs abcd disembark probe/plan/pack forever over an untrusted target repo — no race required, breaking the package's stated no-hang-on-FIFO invariant

## Evidence

- `internal/core/lifeboat/probe.go:286` -- `c.root.Open(rel)` (no `nonBlock`, no Lstat), unlike the exemplar `SourceContext.ReadFile` at `probe.go:230`.
- 18 call sites via adapters (`sources_conventions.go`, `sources_native.go`, `plan.go`, `graveyard_abandoned.go`); `cli.go:515/596/644` pass an arbitrary user-named target repo.
- `nonblock_unix.go:16` defines `nonBlock = O_NOFOLLOW | O_NONBLOCK`.

## Verifier verdict -- CONFIRMED (substantive, no race)

Live exploit against the HEAD build: a target repo containing `README.md` + a FIFO named `docs` hangs `abcd disembark probe/plan/pack` (EXIT 124 under timeout; control repo 0.031s). SIGQUIT dump shows adapter goroutines wedged in ListDir. Breaks the package's stated invariant (`probe.go:128-129` "cannot ... hang on a FIFO"). Fix (verified): `c.root.OpenFile(rel, os.O_RDONLY|nonBlock, 0)` -- ListDir's existing `len(entries)==0 -> nil` turns the resulting `not a directory` into absent; directory listing unaffected; self-probe byte-identical.
