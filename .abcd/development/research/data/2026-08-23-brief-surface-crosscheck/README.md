# Brief-to-surface crosscheck — measured corpus, 2026-08-23

Three full-tier runs of the `iss35-brief-surface-crosscheck` detector against
the brief's surface chapters, kept as data rather than as prose. This is the
empirical base
[`itd-147`](../../../intents/shipped/itd-147-the-brief-s-surface-chapters-are-a-generated-reflection-of-t.md)
rests on, and the measurement behind
[`iss-2608231346137587`](../../../../work/issues/resolved/iss-2608231346137587-brief-surface-chapters-drift-steadily-and-the-gate-passes-anyway.md).

## Why the files are here rather than in a scratch directory

The dispositions from run 3 already survive as a semantic-gate receipt at
[`work/reviews/6e4d5377…/iss35-brief-surface-crosscheck.json`](../../../../work/reviews/6e4d5377de8020df2dbb1ed6b47eccfd20a0cb80/iss35-brief-surface-crosscheck.json).
That receipt carries `where`, `class`, `item` and a `disposition` for each of
the 147 entries — enough to prove the gate ran and what it decided, which is
what a receipt is for.

It does not carry `claim` and `reality`: the quoted brief sentence and the
observed binary behaviour that make each entry checkable by someone who was not
in the room. Those two fields are the evidence, and they existed in one
gitignored directory on one machine. Runs 1 and 2 have no receipt at all.

Detector output is not a verdict, so it does not belong in the decision record;
it is measurement, so it belongs with the research it grounds.

## The runs

| File | Content commit | Findings | What it measures |
|---|---|---|---|
| [`run-1-7a4ee003.json`](run-1-7a4ee003.json) | `7a4ee003` | 125 | The baseline sweep, at the 0.6.2 release-gate content commit |
| [`run-2-a2c77e5e.json`](run-2-a2c77e5e.json) | `a2c77e5e` | 126 | Same manifest, tree differing only by seven shipped-doc fixes |
| [`run-3-6e4d5377.json`](run-3-6e4d5377.json) | `6e4d5377` | 147 | Brief byte-identical to run 2's tree |

All three are tier `full`, 28 checkers, manifest
`sha256:20b7f07e…`, pinned by
[`release-gate/manifest.json`](../../../release-gate/manifest.json) and produced
by [`release-gate/brief-surface-crosscheck.js`](../../../release-gate/brief-surface-crosscheck.js).

## The property that makes runs 2 and 3 worth keeping together

`git diff a2c77e5..6e4d537 -- .abcd/development/brief/` is empty: the detector's
subject is byte-identical across those two runs. The counts are 126 and 147, and
the class distribution moves in every class:

| Class | Run 2 | Run 3 |
|---|---|---|
| `false-claim` | 48 | 64 |
| `undocumented-surface` | 50 | 48 |
| `stale-count` | 18 | 21 |
| `fictional-layout` | 7 | 11 |
| `criterion-violation` | 3 | 3 |

So the pair is a controlled measurement of the **detector**, not only of the
brief: a host-delegated checker reading unchanged text returns a different set
each time. Any claim that rests on a single run's count — including a claim
about how much the brief drifted — inherits that spread. Reading run 3 alone
would hide it, which is the reason to keep all three rather than the largest.

## How to use this

Mine it before fixing anything by hand. A hand-fixed brief starts drifting again
at the same rate and destroys the dataset that would locate the seam; that
argument is `itd-147`'s, and this corpus is what supports it.

The entries are detector output: no verdict, no dispositions, and no claim that
any individual finding is correct. Run 3's dispositions live in the receipt
linked above.

## Chapters implicated: the baseline itd-147 is assessed against

itd-147 moves each surface chapter's flags and sub-verbs into a generated
appendix. Because the counts do not reproduce (iss-2608231409595789), progress
is assessed by **which chapters are implicated**, not by how many findings there
are (itd-147 ac-8). Each cell below gives all findings in that chapter, then the
`false-claim` and `stale-count` findings among them. It is derived from the three
files above by grouping each entry's `where` on its chapter.

| Chapter | Run 1 | Run 2 | Run 3 |
|---|---|---|---|
| `01-ahoy.md` | 9 / 4 | 12 / 6 | 10 / 5 |
| `02-disembark.md` | 5 / 2 | 5 / 1 | 3 / 1 |
| `03-embark.md` | 3 / 3 | 4 / 1 | 3 / 2 |
| `04-launch.md` | 3 / 1 | 7 / 6 | 4 / 2 |
| `05-intent.md` | 9 / 7 | 9 / 7 | 10 / 8 |
| `06-capture.md` | 4 / 1 | 4 / 1 | 9 / 4 |
| `07-memory.md` | 4 / 4 | 6 / 5 | 5 / 3 |
| `08-abcd.md` | 4 / 1 | 3 / 2 | 5 / 3 |
| `09-reflect.md` | 6 / 3 | 3 / 1 | 4 / 1 |
| `10-docs.md` | 3 / 2 | 1 / 1 | 4 / 3 |
| `11-history.md` | 3 / 3 | 3 / 2 | 0 / 0 |
| `12-version.md` | 4 / 4 | 3 / 2 | 2 / 2 |
| `13-consult.md` | 2 / 1 | 3 / 0 | 3 / 0 |
| `14-ingest.md` | 2 / 0 | 3 / 3 | 5 / 2 |
| `15-prepare-this-repo.md` | 6 / 4 | 4 / 1 | 4 / 3 |
| `16-lint.md` | 5 / 4 | 4 / 3 | 4 / 3 |
| `17-guard.md` | 7 / 5 | 8 / 5 | 5 / 4 |
| `18-ideate.md` | 2 / 0 | 4 / 0 | 1 / 0 |
| `19-identity.md` | 1 / 0 | 4 / 1 | 3 / 0 |
| `20-banlist.md` | 5 / 3 | 4 / 1 | 5 / 3 |
| `21-update.md` | 5 / 1 | 3 / 0 | 6 / 3 |
| `22-site.md` | 4 / 1 | 6 / 1 | 7 / 2 |
| `README.md` | 6 / 5 | 7 / 6 | 10 / 8 |

Twenty-three files are implicated across the three runs. Twenty-two of them
appear in all three, and `11-history.md` in the first two. The per-chapter
counts move from run to run over an unchanged brief, so the stable signal is
the set of files, not the numbers. Two entries whose `where` named the register
or a chapter with a trailing note are counted under that file.

What would show the seam wrong is a `false-claim` or `stale-count` finding, in a
full-tier run after itd-147 lands, about a flag or sub-verb a chapter's
generated appendix covers. That run is the next release gate's crosscheck.
Classify its findings against this table chapter by chapter, and record the
result against itd-147 when it has run. Whether a chapter is still implicated
for exit codes, output fields or behaviour is out of the seam's reach by design,
and does not count against it.
