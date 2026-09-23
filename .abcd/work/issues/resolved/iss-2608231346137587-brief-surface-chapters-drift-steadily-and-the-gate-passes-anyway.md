---
schema_version: 1
id: "iss-2608231346137587"
slug: "brief-surface-chapters-drift-steadily-and-the-gate-passes-anyway"
severity: "major"
category: "process"
source: "user-observation"
found_during: "v0.6.2 release-gate crosscheck 2026-08-23"
found_at: ".abcd/development/brief/04-surfaces"
resolution: "The brief's surface chapters carry a generated appendix of flags and sub-verbs derived from the command tree, drift-tested, with the hand-written shape prose retired and a check keeping it out (itd-147, spc-2609020906356450)."
impact: internal
resolved_by:
  commit: "86a440d55fe11d407221c1865ad6d1c4469e6f61"
---

A full-tier `iss35-brief-surface-crosscheck` at content commit `7a4ee00`
returned 125 discrepancies: 58 false-claim, 38 undocumented-surface, 15
stale-count, 7 criterion-violation, 7 fictional-layout. Exactly one points at a
file the release bundle carries (`commands/memory.md`, fixed in this cut); the
other 124 are `.abcd/development/brief/`, which ships in no release (adr-28), so
no installer is misled. Contributors reading the record are.

**The class counts above are one run's observation, not a measurement.** A
second full-tier run hours later, on a tree differing only by seven shipped-doc
fixes, returned 126 findings with false-claim at 48 and undocumented-surface at
50 — the same chapters, redistributed. See iss-2608231409595789. What is
reproducible is WHICH chapters drift (29 of 33 files common to both runs); the
counts and classes are not. Scope the intent on the stable half.

The corpus is preserved at
`.abcd/.work.local/scratch/brief-drift-2026-08-23/crosscheck-findings.json` with
the run's content commit, manifest hash and tier. It earns promotion to a dated
research note when the intent below is filed. **Do not fix the 124 by hand
before mining it** — the whole value of this run is that it is evidence.

## What the corpus already says

124 of 125 findings resolve to a blaming commit via `git blame` on their
`file:line`. The dates are the finding: 66 of them were last written in
2026-07 and 58 in 2026-08 — evenly spread, in every class. So this is not one bad merge that outran its
record; it is a steady rate. Roughly half the false claims were written in July
and survived four releases — v0.4.2, v0.5.1, v0.6.0, v0.6.1 — each of which
recorded a PROMOTE receipt against this same detector, at 26, 28 and 37 findings
respectively.

That is the real defect, and it is the proxy-gate class again
(`iss-2608230847432286`): the gate ran, reported, went green, and the thing it
measures got steadily worse. `receipt_gate` never coupled the verdict to the
finding count, and `surface_coverage` — the deterministic half — checks that a
registered sub-verb has a ROW in the chapter's table, never that the prose
beside the row is true. It blocked a commit today for a missing `history drain`
row while 124 false prose claims sat green beside it.

## Diagnosis

The brief mixes two kinds of content with two different failure modes in the
same paragraphs:

- **Shape** — flags, sub-verbs, exit codes, schema fields, counts, file layouts.
  Derivable from the command tree, therefore checkable, therefore should never
  be hand-written. `false-claim` and `stale-count` are shape by definition, and
  most `undocumented-surface` is the binary growing something the chapter's
  shape section never learned.
- **Intent** — why a surface exists, what it refuses to do, which trade was
  made. Not derivable, and cannot drift against code, because it is not a claim
  about code shape.

The comparison that makes it concrete, from the same repo on the same day:
`docs/reference/cli/commands.md` is generated and drift-tested, and the same
reviewers found ZERO discrepancies across its 775 lines. The brief is
hand-written prose and returned 124. The brief holds a second, hand-maintained
copy of a fact the command tree already owns — `one-canonical-primitive`
violated at two copies rather than three.

## Direction, not yet decided

Generate or deterministically check the shape; let the prose carry only the why.
A chapter becomes a generated shape block plus hand-written rationale, and the
shape then cannot drift.

Explicitly rejected: a "you changed a surface, so touch its brief chapter" gate.
It forces an edit without forcing correctness, which is a phantom gate — the
exact class this issue is an instance of.

## Routing

This warrants an intent, and the maintainer has said so. The four-piece split
below was hand-run and **confirmed by the maintainer on 2026-08-23**, with the
intent to be filed after the 0.6.2 cut — so the next session files from this
routing rather than re-running the interview:

- capability -> intent: the brief's surface chapters become a generated
  reflection of the shipped surface;
- trust rule -> ADR + brief invariant: shape claims are derived, never
  hand-authored;
- stance -> principle: none new, `one-canonical-primitive` already says it;
- plumbing -> brief: the generation mechanism and where the seam sits.

Verdict: SPLIT, confirmed. Open questions the corpus should answer first: did the surface change and the
brief change land in the same commit (the hypothesis that locates the seam);
which chapters drift fastest normalised by surface churn; and how many releases
each false claim survived.

## Grounds

- pursued: a shape claim nobody writes by hand cannot drift; a false-claim or stale-count finding about a flag or sub-verb a generated appendix covers, in the next full-tier crosscheck, would show it wrong
