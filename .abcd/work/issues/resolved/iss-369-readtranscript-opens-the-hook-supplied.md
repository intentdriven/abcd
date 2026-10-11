---
schema_version: 1
id: "iss-369"
slug: "readtranscript-opens-the-hook-supplied"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "internal/surface/cli/cli.go"
resolution: "readTranscript routes through fsutil.ReadGuarded: O_NOFOLLOW plus regular-file check on the open descriptor plus size cap; a symlinked transcript is refused rather than followed"
impact: fix
resolved_by:
  commit: "c1ef016bb231e89352bd9dbef029507ac6781a92"
---

readTranscript opens the hook-supplied transcript path with O_RDONLY|O_NONBLOCK but no O_NOFOLLOW, the last bespoke external-input read in the CLI not routed through fsutil.ReadGuarded; defence-in-depth consolidation (no reachable exploit: the path is harness-supplied and no path policy exists)
## Evidence

- `internal/surface/cli/cli.go:1211` — `os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)`, no `O_NOFOLLOW`, unlike `fsutil.ReadGuarded` (`fsutil/fsutil.go:48`) and `readGuardedOperand` (`cli.go:917-918`).
- The regular-file check and cap are on the returned fd (`:1217-1226`) — no leaf TOCTOU, but a followed symlink's fstat reports the target, so a symlink-to-regular-file is not refused.
- `transcript_path` is harness-supplied (`cli.go:943-945`), outside the target repo; content lands in `~/.abcd/history` and is shown by `abcd history show` (sanitised, iss-340).

## Adversarial verdict

CONFIRMED (nitpick / defence-in-depth). The exfiltration framing is REFUTED: the path is attacker-chosen by design with no path policy, so a direct path already reads any file — `O_NOFOLLOW` closes nothing reachable. It is the last bespoke external-input read not routed through the shared primitive. RECORDED, not fixed this round (touches the SessionEnd hook error path; pure defence-in-depth, no reachable exploit). Fix (future): route through `fsutil.ReadGuarded`.
