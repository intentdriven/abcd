---
schema_version: 1
id: "iss-2610032115023656"
slug: "a-sigtstp-sent-from-outside-kill-tstp-not-the-ctrl-z-key"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25: security review fix round e8term2fix, finding 3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/term/raw.go"
remedy: "Defer, or stop through SIGSTOP. Option A (no code): rely on the job-control shell, which saves the terminal state when a foreground job stops and restores its own, and on the session's raw re-entry and redraw at SIGCONT, which TestLongListRestoresTerminalOnInterrupt pins. Option B: internal/term owns one process-lifetime SIGTSTP listener installed with the first session; on a SIGTSTP it restores a live session, then stops the process with kill(getpid, SIGSTOP), and with no live session it stops at once, so Ctrl-Z in cooked mode keeps working; Suspend's kill(0, SIGTSTP) is then caught by it too. B must first reproduce the kernel's orphaned-process-group rule (POSIX: SIGTSTP is discarded for an orphaned group, but SIGSTOP is not), or an abcd run as a session leader without a job-control shell (ssh host abcd) would stop with nobody to continue it; grounds: POSIX.1-2017 System Interfaces, Signal Concepts, on SIGTSTP, SIGTTIN and SIGTTOU in an orphaned process group, and the Go runtime's signal_unix.go sigdisable."
---

A SIGTSTP sent from outside (kill -TSTP, not the Ctrl-Z key, which raw mode delivers as a byte) stops an abcd process that holds a term.RawSession with the terminal still raw; SIGTTIN and SIGTTOU sent from outside do the same. On SIGCONT the session enters raw mode again and redraws, so only the stopped interval is affected. The fix the security review of the answer loop proposed (catch SIGTSTP with signal.Notify, then signal.Reset and kill(self, SIGTSTP) to stop) does not work in Go: a scratch probe (Notify, Reset, then kill(getpid, SIGTSTP)) showed the process is never stopped, because the runtime keeps its own handler installed for SIGTSTP once it has been notified (runtime sigdisable leaves it while sigInstallGoHandler is true) and that handler drops an unwanted SIGTSTP. Catching it would therefore make SIGTSTP ignored for the rest of the process, Ctrl-Z in cooked mode after the interview included, and would swallow the kill(0, SIGTSTP) RawSession.Suspend stops with.
