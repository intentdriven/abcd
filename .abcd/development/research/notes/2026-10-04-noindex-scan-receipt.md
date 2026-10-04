# The noindex home's scan receipt, 2026-10-04

## TL;DR

- D6 of itd-2610030720038073 (spc-2610031309233367 open question 4): eight working copies in a `.noindex`-suffixed probe folder shaped as the renamed store, `~/.d6-probe.noindex/worktrees/<root-sha>/`, cost the desktop search indexer **2.8%** CPU combined on average over the minute after (peak 6.1%), against **4.4%** (peak 21.4%) for the same eight in a dot-folder without the suffix. The pass is the after set under 20% and no run-log stop naming indexing: **PASS**.
- What it does not show: the old-shape set did not reproduce the 2026-10-02 burst that motivated the rename (corespotlightd about 114% with mds_stores about 70%); its mean (4.4%) is itself under the bar, and only its peak (21.4%) shows indexing work. So the receipt passes D6's threshold, and the comparison is inconclusive about the suffix's effect, because the old shape did not burst. Possible reasons, conjecture only: a fresh probe folder, the indexer's state, eight copies of an already-indexed tree, a quieter machine. The stronger evidence is the intent's audit and the next autonomous run's log in the real renamed store.
- Taken on macOS 27.0 (build 26A428), 2026-10-04, with the product thinker's consent ("Yes, run it", 2026-10-04), before the release that renames the home, while the release was cut.

## Method

- Taken before the rename, by the product thinker's ruling of 2026-10-04 ("Yes, that order": the release is cut first, the folder renamed on their return): the after set lives in a probe folder carrying the same suffix, `~/.d6-probe.noindex/worktrees/<root-sha>/`, directly in the home folder as the renamed store will be, because the indexer's rule turns on the `.noindex` suffix. The store itself is measured by the next run's log after the rename.
- Indexing on the home volume: enabled (`mdutil -s` reported "Indexing enabled.").
- The machine was quiet first (one-minute load under 8); the load at the start was {. 16 cores.
- Each set is eight `git worktree add --detach` of abcd's own HEAD, one after another, then thirteen samples five seconds apart (`top -l 13 -s 5`), summing the CPU of corespotlightd, mds_stores and every mdworker per sample. top's first logging sample has no delta, so the figures are over samples 2 to 13.
- The old shape is a dot-folder directly in the home folder, `~/.d6-index-probe`, rather than `~/.abcd` itself: recreating `~/.abcd` would have tripped the stop this release ships, and the indexer's rule turns on the `.noindex` suffix, not on the folder's name. Burst 3 s for the old set.
- Nothing ran in the background; every working copy and the probe folder were removed after, proven by the worktree list and the folder's absence (clean).

## Samples (combined %CPU per sample, 1 to 13)

| set | samples | mean 2-13 | max 2-13 |
| --- | --- | --- | --- |
| baseline (no burst) | 0.0 0.1 0.5 0.0 0.1 2.9 0.4 0.2 0.5 0.2 3.0 0.0 0.2 | 0.7 | 3.0 |
| old shape | 0.0 0.5 0.1 0.3 0.2 0.2 0.3 0.2 21.4 4.9 7.0 0.2 17.6 | 4.4 | 21.4 |
| `~/.d6-probe.noindex` | 0.0 0.0 0.2 0.3 4.9 4.4 5.3 1.9 1.7 3.1 2.4 6.1 3.1 | 2.8 | 6.1 |

## The run log

The run-log half of D6 cannot be read before the rename: no run has opened its lanes in the renamed store. The audit reads the next run's log against this criterion.

## Source

The results file this note is rendered from is kept in the local tier of the checkout that ran it (`.abcd/.work.local/scratch/d6/`), never committed.
