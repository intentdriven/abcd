# The pre-push check's reuse and time, measured over ten more landings

Dated 2026-11-20. A second reading of the measure that criterion 6 of
itd-2611011325421304 asks for, written as step 3 of spc-2611012047503720. The
note of 2026-11-19 (`2026-11-19-preflight-reuse-over-ten-landings.md`) read
ten landings from 2026-11-09 to 2026-11-15 and could not state a check time,
because no report printed one. Since then the preflight prints its time, so
this note reads the ten most recent landings whose preflight this checkout kept
a log of, and sets both reuse and time against the 2026-11-01 baseline. Two
accounts land on main and each keeps its own logs, so the set is not ten
consecutive landings on main. The span runs from #1657 to #1710, and the merged
pull requests inside it with no log here are #1663 to #1671, #1673 to #1701,
and #1703, #1704 and #1708. Every figure below is one a preflight printed, or a
share of two printed counts. Nothing is estimated.

## The baseline

The baseline is in the body of iss-2611011343052330, captured on 2026-11-01:
"A full preflight on 2026-11-01 reported 0 cached results of 133 packages."
That count is the whole of the 2026-11-01 baseline. No record from that day
gives the check's time or splits the 133 packages between the unit and race
stages.

The only timed figures on record are in the resolution of the same issue and
come from step 2's own run, not from 2026-11-01: on one commit through the
load gate at a load of about 10, a cold race stage reused 0 of 126 packages
in 1811s, and the same commit from a fresh worktree reused 125 of 126 in
800s. That is one commit and one machine load, so this note uses it as a
reference and not as a baseline.

## How the ten were chosen

- **A landing's report** is the preflight log that ended `exit 0` and
  contains `push receipt minted`, for a head that a merged pull request
  carries. The heads come from the receipt line and the file name, and each
  was matched to the `headRefOid` of a merged pull request.
- **The ten are consecutive** among this checkout's logs, taken newest first
  by file modification time. A log whose head no merged pull request carries
  is skipped: `preflight-headings-f324e7d2.log` is the newest log and is
  skipped on that ground.
- **Two of the ten took the records-only path.** #1659 and #1702 changed only
  record files, so the preflight skipped the unit and race stages and printed
  no reuse counts. They count as landings and appear in the table, and they
  are left out of the reuse shares, which therefore cover eight landings.
- **Before 2026-11-19 16:18, the logs print no check time.** The reports for
  #1657, #1659 and #1662 print reuse counts but no total and no stage time, so
  their time cells read "not printed".

## The ten reports

The unit column sums the preflight's two unit lines, the `./cmd/...`
packages and the six packages named to run without the race detector, as
the receipt line does. The race stage held 132 packages in every report. Times
are as printed; the merge time is UTC.

| PR | Merged | Head | Unit reused | Race reused | Race stage time | Checked in | Waited (load gate, race gate, check lock) |
|---|---|---|---|---|---|---|---|
| #1657 | 2026-11-19 05:45 | d37c28ce | 8 of 16 | 110 of 132 | not printed | not printed | not printed |
| #1659 | 2026-11-19 05:56 | f7c6414c | records-only | records-only | records-only | not printed | not printed |
| #1662 | 2026-11-19 09:42 | 6049a67d | 5 of 16 | 105 of 132 | not printed | not printed | not printed |
| #1672 | 2026-11-19 17:30 | e7ff5e2e | 6 of 16 | 115 of 132 | 38m41s | 44m00s | 1m37s |
| #1702 | 2026-11-20 15:55 | 81739e55 | records-only | records-only | records-only | 4m09s | 7m54s |
| #1705 | 2026-11-20 18:04 | 36c3d866 | 4 of 16 | 65 of 132 | 43m16s | 48m52s | 8m23s |
| #1706 | 2026-11-20 18:41 | d0596e67 | 5 of 16 | 101 of 132 | 40m46s | 46m53s | 48m27s |
| #1707 | 2026-11-20 18:21 | 402f6e00 | 10 of 16 | 120 of 132 | 11m20s | 16m49s | 52m15s |
| #1709 | 2026-11-20 19:29 | eb62a371 | 8 of 16 | 119 of 132 | 39m35s | 44m53s | 33m44s |
| #1710 | 2026-11-20 20:02 | cf0370c9 | 6 of 16 | 115 of 132 | 39m04s | 44m38s | 42m14s |

The rows are in the order of the logs' modification times, oldest first. #1706
merged after #1707 although its check finished first.

## The reuse the reports show

| Over the eight test-running landings | Unit stage | Race stage |
|---|---|---|
| Packages reused, pooled | 52 of 128 (40.6%) | 850 of 1056 (80.5%) |
| Reuse share per landing, median | 37.5% | 85.2% |
| Reuse share per landing, range | 25.0% (#1705) to 62.5% (#1707) | 49.2% (#1705) to 90.9% (#1707) |

Against the baseline, where 0 of 133 packages were reused, the race stage of
every one of the eight checks reused at least 65 of 132 packages. Taking the
two stages together, 902 of 1184 package runs (76.2%) were reused. The unit
stage's second line reused 0 of its 6 packages in every report here, as it did
in the earlier ten (0 of 5 there), so its share is bounded: 6 of the 16
packages in that stage always run.

For scale, the earlier ten landings (read on 2026-11-19) pooled to 73.1% in
the race stage and 51.4% in the unit stage. The race share here, 80.5%, is
higher and the unit share, 40.6%, is lower. Both ranges are wide, and #1705 is
the low case in each, so this note draws
no trend from the pair.

## The time the reports show

Six test-running landings printed a time. Their medians and ranges:

| Over the six timed landings | Median | Range |
|---|---|---|
| Race stage | 39m19s | 11m20s (#1707) to 43m16s (#1705) |
| Checked in, whole preflight | 44m45s | 16m49s (#1707) to 48m52s (#1705) |
| Waited (load gate, race gate, check lock) | 37m59s | 1m37s (#1672) to 52m15s (#1707) |

The records-only landing #1702 checked in 4m09s after waiting 7m54s.

## What the numbers support

- **A reuse saving in packages run.** Over the eight landings that ran test
  stages, between 49% and 91% of the race stage's packages and 25% to 63% of
  the unit stage's were reused, against 0 of 133 on 2026-11-01. This holds on
  this machine, for these eight heads.
- **No saving in time against the 2026-11-01 baseline.** That baseline has no
  time, so no difference of times can be stated. The check time of the six
  timed landings is a figure by itself and not a comparison.

## What the numbers cannot show

- **A saving in time.** The race stage's time does not follow its reuse share
  in these six. #1707 reused 120 of 132 and took 11m20s. #1709 reused 119 of
  132 and took 39m35s. #1672 and #1710 each reused 115 of 132 and took 38m41s
  and 39m04s. Time depends on which packages ran, on whether the always-rerun
  tests ran, and on how busy the machine
  was. The reports do not separate these, so a faster check cannot be credited
  to reuse from these six alone. The one pair that isolates reuse is step 2's
  own, a cold run of 1811s against a fresh-worktree run of 800s on the same
  commit, which is one pair and not one of these ten.
- **The waits.** Waiting at the load gate and the race gate ranged from 1m37s
  to 52m15s and exceeded the check itself in #1706 and #1707. The
  column also counts the lock on the check directory, which the reuse
  mechanism adds: 4m01s of #1707's wait and 2m44s of #1709's. The rest
  reflects how many other checks and runs were on the machine. The wait is
  kept out of any claim above.
- **Consecutive landings on the default branch.** The ten are consecutive among
  the logs this checkout keeps, not among everything that merged. Pull requests
  merged between them whose preflight this checkout did not run have no log
  here, and their reuse is not known.
- **A like-for-like baseline.** The baseline is one run's count of 133
  packages, not split by stage, and these reports count 132 race packages and
  16 unit packages. They compare as shares of reused packages, not as
  package-for-package runs.
- **That each reused result was right.** Reuse replays a cached pass. The
  reports cannot show that a reused package would have passed if it had been
  rerun. The hosted run, which passes `-count=1` on every test step, is the
  check for that.
- **Criterion 6 as a whole.** The reuse half is measured over two sets of
  ten landings. The time half has six timed landings and no timed baseline,
  so the criterion's comparison of check time is not met by this note.

## What this does not settle

Criterion 6 of itd-2611011325421304 is not met as written. The set is not ten
consecutive landings on main, and the race stage's time does not fall with its
reuse share. So the intent stays planned. The decision that records this is
dcn-2611202032152249, and the issue that holds the open question of the race
stage's time is iss-2611202032159749.
