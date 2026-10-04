---
schema_version: 1
id: "iss-2610041944562639"
slug: "after-ctrl-z-and-a-resume-the-plain-terminal-answer-loop-can"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "abcd-e8: merge-queue ejection of #818, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/term/raw.go"
remedy: "Diagnose first, on a scratch copy: run the subtest in a loop under the pseudo-terminal helper with the child's process group state and wait status logged after SIGCONT (WUNTRACED a second time shows a SIGTTOU re-stop). If SIGTTOU: make the session's terminal calls safe from a background group (block or ignore SIGTTOU around tcsetattr, as shells do for their own tcsetattr, or skip re-entry until the group is in the foreground), and make the test child the terminal's foreground group as a shell would. If a race: hand the continue to Suspend through one owner. Pin it with the subtest run 200 times under -race. Grounds: POSIX job control on tcsetattr from a background process group (SIGTTOU), and the CI dump."
---

After Ctrl-Z and a resume, the plain-Terminal answer loop can fail to come back: on the macOS merge-queue runner (run 37210244082, 2026-10-04) the subtest TestLongListRestoresTerminalOnInterrupt/ctrl-z restores, and SIGCONT re-enters and redraws saw the child stop and the terminal restored while stopped (both asserted), then after SIGCONT the child wrote nothing for 20 s, so the question was not redrawn and raw mode was not re-entered. A person would see Ctrl-Z, fg, and the question still frozen until Ctrl-C. Lead: RawSession.reenter's tcsetattr (internal/term/raw.go) runs from a process group that is not the terminal's foreground group, and the kernel answers with SIGTTOU, stopping the child again. Alternative: a race between Suspend's wait for its continue token and the watch goroutine's continued(). It ejected an unrelated pull request from the merge queue; the detector stays armed.
