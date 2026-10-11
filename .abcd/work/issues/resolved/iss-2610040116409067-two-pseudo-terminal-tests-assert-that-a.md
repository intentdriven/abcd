---
schema_version: 1
id: "iss-2610040116409067"
slug: "two-pseudo-terminal-tests-assert-that-a"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "abcd-e8 build of plain-Terminal interviews, 2026-10-04: a peer's PR ejected from the merge queue"
origin: researcher-authored
production_mode: hand-written
remedy: "Wait for the message with ptytest's WaitFor, which polls the drained output up to the test's deadline, instead of reading Output() once. The parent holds its own copy of the terminal end, so the bytes stay readable after the child exits and only the read's timing is at fault. Grounds: the failing run's output and the helper's drain loop."
resolution: "Both panic cases wait for the message with WaitFor instead of reading the drained output once."
impact: internal
resolved_by:
  commit: "c729b7fbdd76b774a31b16915c99b18827482985"
---

Two pseudo-terminal tests assert that a panic's message reached the terminal by reading the drained output once, straight after the child exits (internal/surface/cli/ask/pty_test.go, the a-panic-inside-the-loop case; internal/term/raw_child_test.go, the panic-in-a-hook case). The drain goroutine may not yet have read the child's last bytes, so the assertion races it. It failed on the macOS merge-queue runner on 2026-10-03 with 'the panic did not reach the terminal' and ejected an unrelated pull request from the merge queue.

## Grounds

- pursued: the panic text reaches the test whenever it reaches the terminal; reproduced on a scratch copy with a short trace and the message's chunk delayed, the read-once form failing and the wait passing. A failure of either case now names a message that truly never arrived.
