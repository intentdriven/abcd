# The checklist's crash lab: the kill loop

Date: 2026-11-25. Tree: `surge-checklist-pr1`, from main `b8250514c`. Machine: macOS,
16 online cores, under the surge's shared load. This note records the kill half of
spc-2610230805429429 step 7 (technical criterion 2 of itd-2610170638184486). The power-cut
half has not run; it is owed in the remainder spec, and nothing below speaks for it.

## What the lab claims

A writer killed at any point of its write cycle leaves the reader a list it can use:

- the checklist itself, whole, never a torn file and never last-good or "nothing" once a
  write has finished;
- never older than the last write the writer acknowledged: either that version or, when
  the kill fell after the rename and before the acknowledgement, the next one;
- last-good whole, and never newer than the checklist;
- no byte of a refused secret-bearing write anywhere in the tier, a torn temp file included;
- a temp file a kill left is never read, and the next write removes it once it is more than
  a minute old.

It does not claim anything about a power cut. A process kill leaves the kernel's page cache
in place, so the flushes (`F_FULLFSYNC` on macOS, in `internal/fsutil/flush.go`) are not
tested by it; two hard power cuts, one on macOS and one on Linux, are what would.

## How it runs

`TestCrashLabKilledWriterLeavesAReadableList` in `internal/core/checklist/crashlab_test.go`
re-executes the test binary as the writer (`TestCrashLabWriterProcess`). The writer runs the
real path, `checklist.Update`, with its lock, its read, its scan and its two atomic writes, and
flushes as a shipped binary does, whatever the run's own `ABCD_TEST_SKIP_FLUSH`. It writes
version after version, each of a different size (20 to 79 items), attempts a write carrying a
token beside every fifth version, which the scanner must refuse, and prints `wrote <n>` after
each cycle returns.

The parent waits for the writer's first acknowledged write, lets it run for a random 0 to
150 ms (about two cycles), and sends SIGKILL to the writer's process group. The writer leads
its own group (`procgroup.Lead`), and the kill is addressed through the pid its own exec
handle holds while the child is unreaped; before signalling, the test checks that the handle
is unreaped and that the pid still leads its group, and after the reap it checks with signal 0
that no member of the group still runs. Nothing is killed by pattern, and nothing outside the
group the test started is signalled. Then it reads, and checks every claim above.

`TestCrashLabTornTempIsNeverRead` writes out every outcome a kill between a temp file's
creation and its rename can leave: a temp file cut at each of the first and last 33 lengths of
an encoded list and at every 97th byte between, with and without a checklist beside it.

The default run makes 25 kills, about five seconds, and is part of the ordinary suite.
`ABCD_CRASH_LAB_KILLS=1000` runs the full lab, and `ABCD_CRASH_LAB_SEED=<n>` replays a run's
delays.

## The run

One full run, through the load gate as a narrow run:
`ABCD_CRASH_LAB_KILLS=1000 go test -trimpath -count=1 -run TestCrashLab ./internal/core/checklist/`.

| Measure | Value |
|---|---|
| Kills | 1,000 |
| Seed | 1795632066181716000 |
| Time | 3 min 37 s (about 217 ms a kill, the writer's start-up included) |
| Writes acknowledged before the kills | 1,724 |
| Kills after a rename and before its acknowledgement (the newer version read) | 214 |
| Kills that left a torn temp file (between a temp file's creation and its rename) | 146 |
| Reads that fell back to last-good or "nothing" | 0 |
| Reads older than the last acknowledged write | 0 |
| Files in the tier holding any part of the refused token | 0 |
| Torn temp files left after the closing write (aged past a minute) | 0 |

The torn-temp test passed beside it.

## Watched failing

Both tests were run against a broken tree before they were kept:

- the checklist written in place (truncate, write half, flush, write the rest, flush) instead
  of by a rename: the kill loop failed at its fifth kill, the reader falling back to last-good
  because `checklist.json` did not decode;
- the torn file planted at `checklist.last-good.json` instead of a temp name: the torn-temp
  test failed at the whole-file cut, the reader showing it as last-good.

## What it found

No defect in the store. One test-side lesson: a check for four-character runs of a random
token over tens of kilobytes of version numbers matches by chance (the first 1,000-kill run
reported a false echo), so the tier check uses eight-character runs and the token's prefix.
