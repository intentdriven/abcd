# The renamed home's measured receipt, 2026-10-28

## TL;DR

- D6 of itd-2610030720038073 (ac-6), taken in the real renamed store, `~/.abcd.noindex/worktrees/<root-sha>/`: eight working copies created in 8 s cost the desktop search indexer **0.56%** CPU combined on average over the minute after (peak 0.8%), against **3.28%** (peak 21.9%) for the same eight in the old shape, a dot-folder directly in the home. The threshold is the after set under 20%: **PASS**, and well inside it.
- Taken on macOS 27.0.1 (build 26A434), 2026-10-28, ten cores, after the rename shipped.
- The old shape showed a burst (two samples, 11.9% and 21.9%, a few seconds after the copies appeared) and the renamed store showed none. That is the direction the rename predicts, but the old shape's mean is itself under the bar and nowhere near the 2026-10-02 burst (corespotlightd near 114%, mds_stores near 70%), so the comparison supports the suffix's effect without reproducing the original incident.
- Caveat on the precondition: `mdutil -s /` reported "Index is read-only." on this run, not "Indexing enabled." as on 2026-10-04. The indexer still reacted to the old-shape set, so it was working, but the criterion's "indexing on" is read here as that state, and a reader who needs a stricter reading should repeat the run on a volume reporting enabled.
- The run-log half cannot be checked: see below.

## Method

- The measurement ran through the load gate and waited for a quiet machine itself: one-minute load under 10 and no compiled test binary running (load 4.94 at the start). The load average falling through the run (15-minute figure 67.8 to 57.9) is the preceding queue draining, not work during the samples.
- Each set is eight `git worktree add --detach` of the repository's `origin/main`, one after another, then thirteen samples five seconds apart (`top -l 13 -s 5`), summing the CPU of corespotlightd, mds_stores and every mdworker per sample. top's first sample has no delta, so the figures are over samples 2 to 13.
- The old shape is `~/.d6-index-probe/worktrees/<root-sha>/`, a dot-folder directly in the home without the suffix, as the 2026-10-02 run's shape; recreating `~/.abcd` itself would trip the stop this release ships. The renamed shape is the real store path, `~/.abcd.noindex/worktrees/<root-sha>/`.
- Every working copy and the probe folder were removed after, proven by the worktree list and the probe folder's absence.

## Samples (combined %CPU per sample, 1 to 13)

| set | samples | mean 2-13 | max 2-13 |
| --- | --- | --- | --- |
| baseline (no copies) | 0.0 0.5 0.4 0.6 0.6 0.6 0.4 0.6 0.6 0.6 0.5 3.4 0.6 | 0.78 | 3.4 |
| old shape | 0.0 0.5 0.7 0.4 0.6 11.9 21.9 0.5 0.6 0.6 0.6 0.4 0.6 | 3.28 | 21.9 |
| `~/.abcd.noindex` (real store) | 0.0 0.6 0.6 0.5 0.8 0.5 0.5 0.5 0.5 0.6 0.5 0.6 0.5 | 0.56 | 0.8 |

Endpoint-security agent processes (eleven present) were summed alongside and stayed under 1.0% in every sample of every set, so they do not explain either result.

## The run log

The run-log half of D6, that no run log carries a `stop` naming indexing, cannot be checked: neither `~/.abcd.noindex/runs` nor `~/.abcd/runs` exists on this machine, so there is no run log to read. It stays unrecorded, as the 2026-10-04 receipt left it, and the next autonomous run that opens its lanes in the renamed store is the first log that can be read against it.

## Source

The raw `top` output and the run log this note is rendered from are kept in the local tier of the checkout that ran it (`.abcd/.work.local/scratch/d6-home/`), never committed.
