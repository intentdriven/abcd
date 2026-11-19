# The pre-push check's reuse, measured over ten landings

Dated 2026-11-19. The measure that criterion 6 of itd-2611011325421304 asks
for, written as step 3 of spc-2611012047503720: ten landings' `make preflight`
reports, read after step 2 landed (5795e213f, merged in #1413 on 2026-11-05),
set against the 2026-11-01 baseline of no reuse. Every figure below is one the
preflight printed. A share is that printed count divided by another printed
count, and nothing else is estimated.

## The baseline

The baseline is in the body of iss-2611011343052330, captured on 2026-11-01:
"A full preflight on 2026-11-01 reported 0 cached results of 133 packages."
The same record says every preflight then ran all packages twice from scratch,
once in the unit stage and once under the race detector. That one count is
the whole of the baseline. No record from 2026-11-01 gives that check's time
or splits the 133 packages between the two stages.

## How the ten were chosen

- **A landing's report** is the last preflight that minted a push receipt for
  the head its pull request merged. Each receipt line (`push receipt minted for
  <sha>`) was matched to the `headRefOid` of a pull request merged since
  2026-11-05.
- **Only a run that ended `exit 0` with a receipt counts.** Nine earlier logs
  (#1446 to #1472) minted a receipt for a merged head, but they end on the
  receipt line with no exit line after it, so they are left out.
- **The ten are the first ten such reports after step 2**, #1473 to #1582,
  taken in merge order with none skipped.

## The ten reports

Each stage prints `reused N of M packages (A with no test run, B with only
their always-rerun tests run); R ran`. A package counted as reused either ran
no test (A) or ran only its always-rerun tests (B). A package not reused ran
in full. The unit stage prints two such lines: the `./cmd/...` packages, and
the tests named to run without the race detector. The table gives their sum,
which is the figure the receipt line prints.

| PR | Merged | Head | Unit reused of run | Race reused of run | Race: no test run / always-rerun only / in full | Always-rerun tests run, unit; race | Wall time |
|---|---|---|---|---|---|---|---|
| #1473 | 2026-11-09 | 3bc823513c40 | 9 of 14 | 96 of 128 | 63 / 33 / 32 | 17 in 5 packages; 1592 in 58 | not printed |
| #1474 | 2026-11-09 | 1d459ce94e91 | 9 of 14 | 117 of 128 | 63 / 54 / 11 | 17 in 5; 1591 in 58 | not printed |
| #1475 | 2026-11-10 | d974d92c9ebf | 4 of 14 | 56 of 128 | 35 / 21 / 72 | 17 in 5; 1596 in 58 | not printed |
| #1476 | 2026-11-10 | a4ffe1dc7ea8 | 4 of 14 | 62 of 128 | 41 / 21 / 66 | 17 in 5; 1596 in 58 | not printed |
| #1478 | 2026-11-10 | bab4b25292b4 | 4 of 14 | 43 of 128 | 34 / 9 / 85 | 17 in 5; 1597 in 58 | not printed |
| #1479 | 2026-11-10 | 233ce83220a3 | 9 of 14 | 117 of 128 | 63 / 54 / 11 | 17 in 5; 1600 in 58 | not printed |
| #1480 | 2026-11-10 | a019ef4b9e2b | 9 of 14 | 110 of 129 | 61 / 49 / 19 | 17 in 5; 1600 in 58 | not printed |
| #1481 | 2026-11-10 | 4c099853fb30 | 7 of 14 | 108 of 128 | 60 / 48 / 20 | 17 in 5; 1600 in 58 | not printed |
| #1482 | 2026-11-10 | 5e671f5b36b0 | 9 of 14 | 109 of 128 | 60 / 49 / 19 | 17 in 5; 1600 in 58 | not printed |
| #1582 | 2026-11-15 | 1b2de3d63edc | 8 of 14 | 120 of 131 | 60 / 60 / 11 | 17 in 5; 1783 in 64 | not printed |

In every report, the unit stage's second line (the tests named to run without
the race detector) reused 0 of 5 packages. Its five packages always run in
full, because each reaches the repository tree from a package that starts
subprocesses. The "always-rerun tests run" column sums the counts on the
report's `ran <package>: N always-rerun tests` lines for each stage.

## The saving the reports show

| Over the ten | Unit stage | Race stage | Both stages |
|---|---|---|---|
| Packages reused | 72 of 140 (51.4%) | 938 of 1284 (73.1%) | 1010 of 1424 (70.9%) |
| of which ran no test at all | 29 (20.7%) | 540 (42.1%) | 569 (40.0%) |
| of which ran only their always-rerun tests | 43 | 398 | 441 |
| Packages run in full | 68 (48.6%) | 346 (26.9%) | 414 (29.1%) |

Against the baseline, where 0 of 133 packages were reused, these ten checks
reused 1010 of 1424 package runs (70.9%). 569 of those (40.0%) ran no test at
all. The race stage, which holds almost all the packages, reused between 43 of
128 (33.6%, #1478) and 120 of 131 (91.6%, #1582) per check.

## What the numbers cannot show

- **The check's time.** No report prints its own wall time. The logs' file
  times do not stand in for it. One log (#1473's) was created at 14:17 UTC and
  last written at 17:44, but the secrets scan in that same run printed 17:11,
  partway through the run, so the file's creation time does not mark when the
  run started. No record from 2026-11-01 gives the baseline check's time
  either. So the check-time half of criterion 6 is not recorded here. What the
  logs do print is the wait at the load gate before each run (31s to 1h14m52s
  over the ten). That is waiting, not checking. The only timed comparison on
  record is step 2's own, on one commit and not one of these ten, in
  iss-2611011343052330's resolution.
- **Test time per package.** The counts are of packages, not of seconds. A
  reused package may be a quick one, and the always-rerun tests (1591 to 1783
  per race stage) and the packages run in full may hold most of the time.
- **Consecutive landings on the default branch.** These ten are consecutive
  among the reports this checkout keeps, not among everything that merged.
  98 other pull requests merged between #1473 and #1582: #1483, between #1481
  and #1482, and 97 between #1482 and #1582. This checkout's logs hold no
  report for any of them, so their reuse is not known.
- **A like-for-like baseline.** The baseline is one run's single count of 133
  packages, not split by stage. The ten count 142 to 145 package runs across
  the two stages, so they compare as shares, not as package-for-package runs.
- **The declared list's own part.** Every `ran` line for an always-rerun
  package names its first test as `undeclared`, which means the classifier
  found it, not the always-rerun list on the default branch
  (`.abcd/work/always-rerun-tests.txt`). The report does not separate the
  tests the list declared from the ones the classifier found.
- **That each reused result was right.** Reuse replays a cached pass. These
  reports cannot show that a reused package would have passed if it had been
  rerun. The hosted run, which passes `-count=1` on every test step, is the
  check for that.
