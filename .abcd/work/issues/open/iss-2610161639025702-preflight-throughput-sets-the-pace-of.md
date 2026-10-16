---
schema_version: 1
id: "iss-2610161639025702"
slug: "preflight-throughput-sets-the-pace-of"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "product thinker, 2026-10-16 evening plan"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "First research the state of the art, with primary sources: merge-queue batching (GitHub merge queue groups, bors, Zuul), test impact analysis and selective testing (go list -deps per changed package, Bazel and Pants affected targets), build and test result caching, sharding, and splitting local preflight from CI. Then adopt what fits: (1) a bundled rule that a feature's reviewed step branches are merged into one integration branch and preflighted once, and that records-only branches are stacked on one branch; (2) a preflight whose test stages run only the packages a change can affect, plus the race lane's affected subset, with a full run kept for the release cut and CI; (3) records-only changes skip the Go test stages, under a check that proves no code changed. Pin each with a test and measure queue time before and after."
---

Preflight throughput sets the pace of all work on this machine, and nothing keeps it in bounds. On 2026-10-16 the load gate admitted one make preflight at a time, each took about 40 minutes, and about 20 waiters were queued across seven sessions. Every reviewed branch had its own preflight, so a seven-step feature needed seven full runs, and records-only branches paid the same full cost as code. Sessions waited hours, and the queue's ordering after the reboot starved late waiters (fixed separately on fix/load-queue-start-order). No rule tells sessions to batch work, and no part of preflight scales its cost to what a change touches.

## Evidence 2026-10-16 (abcd-a2, measured on this machine: 16 cores, macOS)

**What a preflight costs**
- **No model tokens.** `make preflight` is local work only: format, lint and record gates, build, vet, unit tests, race tests, and the two eval lanes, which are build-tagged Go tests of about five seconds each. Its cost is CPU, memory and wall-clock time.
- **Wall-clock, from today's logs.** Each figure is gate wait plus run.

  | Branch | Total | Waiting at the gate | Running |
  |---|---|---|---|
  | embargo store fix | 61 min | 37.5 min | about 23 min |
  | messaging step 4 | 86 min | 62 min | about 24 min |

  The run takes about 23 to 24 minutes; the wait is the larger part.

**What the queue costs**
- **Race runs queued alone, without a preflight, waited at the gate:**
  - 17 min (an independent reviewer's run);
  - 33 to 37 min (three sub-agents' checks);
  - one never opened inside its 2 h timeout (messaging step 3, fix round 2).
- **Depth.** About 20 waiters were queued across seven sessions at 17:30. Six of this session's branch preflights were then folded into two integration branches, to cut its share to two runs.
- **Load.** The gate's limit is 32 (2 × 16 cores). The one-minute load ran 38 to 75 through the afternoon and reached 164 after the reboot. Most of that was the operating system re-indexing (Spotlight, fseventsd, the VPN network extension), not test work, and it held the gate shut regardless.
- **Ordering.** After the reboot, process ids wrapped. Pid-ordered admission then starved waiters started earlier, a separate defect fixed on `fix/load-queue-start-order`.

**Where tokens actually go around preflights**
- **Waiting is cheap.** A session waiting on a gated preflight in the background reads one short notice and a few tail lines: hundreds of tokens.
- **Fix rounds cost the most.** A fix-round sub-agent that runs the full package suites itself used about 200,000 to 390,000 tokens per round. Most of that was reasoning and reading code; the test output was the smaller part.
- **Reviews cost less.** A review pair used about 60,000 to 240,000 tokens per reviewer.

**Next measurement.** A trial tonight runs two preflights at once, with the race stage kept to one at a time (`feat/two-preflights`). It will record each run's wall-clock, its peak one-minute load, and the gate waits, against these one-at-a-time figures. Resource-derived admission is iss-2610161756384596.
