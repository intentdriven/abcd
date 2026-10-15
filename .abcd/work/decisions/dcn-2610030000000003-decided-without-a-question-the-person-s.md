---
id: dcn-2610030000000003
slug: decided-without-a-question-the-person-s
date: 2026-10-03
---
2026-10-03 — Decided without a question (the person's ruling that an obvious answer is decided; recorded by abcd-e8): iss-2610032115023656, a SIGTSTP sent from outside stopping the plain-Terminal answer loop with the terminal raw, is won't-fix under its remedy A. An interactive job-control shell restores its own terminal state when a foreground job stops, and the session re-enters raw mode and redraws once on SIGCONT (pinned by the fix round's pty case). Remedy B, process-wide stop handling in internal/term with the kernel's orphaned-process-group rule reimplemented, risks a stopped abcd that nobody continues. The security review's own fix does not work in Go, because the runtime keeps its handler and drops the signal. Alternative not taken: remedy B.
