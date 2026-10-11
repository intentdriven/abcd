---
schema_version: 1
id: "iss-2608231409595789"
slug: "the-manifest-pins-the-inputs-but-the"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "v0.6.2 release-gate double run 2026-08-23"
found_at: ".abcd/development/release-gate/manifest.json"
resolution: "The manifest's comment, the release-gate README, the detector header, the currency test's doc comment and the phase-8 goal claim what the manifest pins — the inputs, and so the scope two runs of one tier examine — and say the findings, their count and their classes vary between runs; the README states that a receipt's failing count is one run's observation, not a metric. Record correction only; the gate's refusals are unchanged."
impact: internal
resolved_by:
  commit: "b2fc7f3b"
---

`.abcd/development/release-gate/manifest.json` describes itself as "the
reproducibility anchor (iss-122): two honest runs of the same tier mean the same
thing because the doc list, the directions, the checker count, and the prompt are
fixed here rather than chosen per run".

Three full-tier runs on 2026-08-23, same manifest, same 28 checkers:

| | run 1 (7a4ee00) | run 2 (a2c77e5) | run 3 (6e4d537) |
| --- | --- | --- | --- |
| total | 125 | 126 | 147 |
| false-claim | 58 | 48 | 64 |
| undocumented-surface | 38 | 50 | 48 |
| stale-count | 15 | 18 | 21 |
| criterion-violation | 7 | 3 | 3 |
| fictional-layout | 7 | 7 | 11 |

**Runs 2 and 3 are a controlled measurement.** The brief is byte-identical
between those two commits — `git diff a2c77e5 6e4d537 -- .abcd/development/brief/`
is empty — so the detector's entire subject was held constant. It returned 21
more findings, a 17% swing, on the same bytes. Per-file the swings are larger
than the total: `05-prompt-quality.md` 0 to 5, `02-verification-matrix.md` 1 to
7, `06-capture.md` 4 to 9, and in the other direction `11-history.md` 3 to 0 and
`18-ideate.md` 4 to 1.

What IS stable is roughly which chapters are implicated: 28 files appear in both
runs 2 and 3, with 4 unique to run 2 and 10 to run 3. So the detector broadly
agrees on WHICH chapters have drifted and disagrees on HOW MANY discrepancies
each holds and WHAT CLASS each is — and even the file set is only mostly stable,
not fixed.

The manifest is not wrong to exist and it is not lying about what it pins. It
pins the inputs, and it does: scope, depth, doc list, prompt. What it cannot pin
is the output, because the checkers are LLM agents. The defect is the sentence
claiming the consequence — "two honest runs of the same tier mean the same
thing" — which is a statement about outputs that the mechanism cannot deliver.

The consequence is practical, not theoretical:

1. A receipt's `failing` count is not a measurement. Comparing 37 (v0.6.1) to
   147 cannot support "drift got worse", and a later run showing 90 would not
   evidence improvement. The 0.6.2 receipt records 147 and says so in its own
   provenance, because that number will otherwise sit in the ledger beside
   v0.6.1's 37 and invite exactly this comparison.
2. The class distribution cannot drive scope decisions. An intent that scoped
   itself to "the 58 false-claims" would have scoped itself to 48 a few hours
   later.
3. The stable half is still useful, and is the half to build on: which chapters
   drift is reproducible, and that is what locates a generation seam.

Fix is a record correction, not a mechanism change: the manifest's comment
should claim what it delivers — comparable SCOPE, not comparable findings — and
the runbook should say a receipt's finding count is an observation from one run
rather than a metric. Refines iss-2608231346137587, whose corpus this governs;
that issue's own class counts carry the same caveat.

This is the enforcement-claims-are-facts class again (iss-2608230847432286): a
record stating an assurance the machinery underneath it does not provide. It was
found only because a release recovery happened to run the same gate twice in one
afternoon, which is not something the process would otherwise ever do.

## Grounds

- pursued: no record states that two runs of one tier yield comparable findings or that a receipt's count measures drift; a surviving sentence saying runs 'mean the same thing', or a receipt comparison presented as a trend, would show it wrong
