---
schema_version: 1
id: "iss-347"
slug: "readtranscript-opens-the-hook-supplied"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/surface/cli/cli.go"
resolution: "readTranscript routed through fsutil.ReadGuarded; symlink refused, over-cap refused whole"
impact: fix
---

readTranscript opens the hook-supplied transcript path without O_NOFOLLOW or an over-cap probe, the one guarded read left off fsutil.ReadGuarded, so a symlinked transcript_path is followed and an over-cap file is stored silently truncated
## Evidence

- `internal/surface/cli/cli.go:1211` opens with `os.O_RDONLY|syscall.O_NONBLOCK` and no `O_NOFOLLOW`; the `IsRegular` check at `:1221` fstats the already-followed target. Reproduced end-to-end: a symlinked `transcript_path` was followed and the target bytes persisted to the history store. The size check at `:1224` then reads via `LimitReader` with no cap+1 probe, so a file crossing the cap between fstat and read stores silently truncated — contradicting spc-4's refuse-whole invariant.
- Every sibling guarded read carries `O_NOFOLLOW` (`internal/fsutil/fsutil.go:48`, spec/rules/record stores); the user-typed operand path into the same `history.Capture` sink is already `fsutil.ReadGuarded` via `readGuardedOperand` (`cli.go:917`) — the more-trusted input is guarded, the hook input is not.
- Refuter verdict: CONFIRMED substantive-minor (CWE-59 hardening/consistency; same-identity write, no privilege boundary). Leaf `O_NOFOLLOW` costs nothing legitimate (symlinked ancestors still resolve).
