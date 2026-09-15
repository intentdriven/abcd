# Brief-to-surface crosscheck — measured corpus, 2026-08-23

Three full-tier runs of the `iss35-brief-surface-crosscheck` detector against
the brief's surface chapters, kept as data rather than as prose. This is the
empirical base
[`itd-147`](../../../intents/planned/itd-147-the-brief-s-surface-chapters-are-a-generated-reflection-of-t.md)
rests on, and the measurement behind
[`iss-2608231346137587`](../../../../work/issues/open/iss-2608231346137587-brief-surface-chapters-drift-steadily-and-the-gate-passes-anyway.md).

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
