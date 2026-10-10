---
schema_version: 1
id: "iss-2610041944562639"
slug: "after-ctrl-z-and-a-resume-the-plain"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "abcd-e8: merge-queue ejection of #818, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/term/raw.go"
remedy: "Diagnose first, on a scratch copy: run the subtest in a loop under the pseudo-terminal helper with the child's process group state and wait status logged after SIGCONT (WUNTRACED a second time shows a SIGTTOU re-stop). If SIGTTOU: make the session's terminal calls safe from a background group (block or ignore SIGTTOU around tcsetattr, as shells do for their own tcsetattr, or skip re-entry until the group is in the foreground), and make the test child the terminal's foreground group as a shell would. If a race: hand the continue to Suspend through one owner. Pin it with the subtest run 200 times under -race. Grounds: POSIX job control on tcsetattr from a background process group (SIGTTOU), and the CI dump."
resolution: "The kernel lost the continue, not the program: on darwin a SIGCONT sent the instant wait4 reports a self-inflicted SIGTSTP stop can arrive before the stop suspends the task, which then stays suspended while reported running (probe: 21 of 3,000 immediate continues lost, 0 with a 20 ms pause, every loss recovered by a second SIGCONT). SIGTTOU ruled out (the pty is not the child's controlling terminal) and the session's token race ruled out (the stalled child never returned from its kill). The test now continues again while the kernel reports no new stop, and fails on a stop it does report; a subtest drops the first continue to pin it. raw.go unchanged."
impact: internal
resolved_by:
  commit: "4f321039c"
---

After Ctrl-Z and a resume, the plain-Terminal answer loop can fail to come back: on the macOS merge-queue runner (run 37210244082, 2026-10-04) the subtest TestLongListRestoresTerminalOnInterrupt/ctrl-z restores, and SIGCONT re-enters and redraws saw the child stop and the terminal restored while stopped (both asserted), then after SIGCONT the child wrote nothing for 20 s, so the question was not redrawn and raw mode was not re-entered. A person would see Ctrl-Z, fg, and the question still frozen until Ctrl-C. Lead: RawSession.reenter's tcsetattr (internal/term/raw.go) runs from a process group that is not the terminal's foreground group, and the kernel answers with SIGTTOU, stopping the child again. Alternative: a race between Suspend's wait for its continue token and the watch goroutine's continued(). It ejected an unrelated pull request from the merge queue; the detector stays armed.

## Grounds

- pursued: a SIGCONT the kernel loses straight after the reported stop leaves the question frozen, and a resend while no new stop is reported brings it back; a stall with the kernel reporting the child stopped again, or one a second SIGCONT does not end, would show it wrong
