---
schema_version: 1
id: "iss-2609091956001547"
slug: "the-brief-surface-crosscheck-returns-a-fresh-nonzero-sample"
severity: "major"
category: "process"
source: "review-followup"
found_during: "v0.8.0 release gate crosscheck rounds 1 and 2"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/"
deferred_after: "v0.7.1"
deferral_reason: "The finding is about the shape of the work rather than any one claim, and the evidence for it was only complete once the second round returned. Fixing it inside the release it was found in would mean another sampling round, which is the thing it says does not converge. Recorded here so the next cycle starts from the measurement instead of rediscovering it. The waiver lapses at v0.8.0 and the finding returns to the gate."
---

The iss-35 brief-surface crosscheck does not converge on a clean run. Three
armings against this repository, each at full tier over the pinned 37 checkers:

| Round | Findings | After |
| --- | --- | --- |
| 1 | 246 | the prose rewrite |
| 2 | 137 | 134 fixed across 39 files |
| 3 | 109 | this record |

Round 3 ran against a tree in which round 2's findings had been fixed and
verified, so the 109 are very largely claims no previous round reported. The
detector is sampling a large prose corpus, not enumerating a fixed defect list,
and each honest run draws a different sample.

## Fixing rounds are not monotonic

Round 3 found a false claim that round 2 introduced. Round 2's finding 8 said
that `disembark review` "takes and reads a source repo" and that no test
fingerprints it. The first half is false: `ReviewLifeboat` gates the source as a
real directory and takes `filepath.Base` for the attestation, and its own header
says the content is never read
(`internal/core/lifeboat/synthesis_review.go:11-14`, `:90-100`), which
`commands/disembark.md:211-213` states too. The round-2 fix corrected the
evidence half and carried the false half into the record. It is repaired in the
same change that files this record.

That is the load-bearing observation. A round of fixes closes most of what it
touches and opens a little, so the tail is not simply long: it is fed.

## What this is not

Not a defect in the detector. Its findings verify: across rounds 2 and 3 the
confirmed rate was above 95 per cent, with three refutations in 137 and every
refutation caught by adversarial triage rather than by the detector recanting.

Not a gate that fails to gate, either. `iss-122` pinned scope and depth and
deliberately left the pass threshold out: `receipt_gate` refuses a manifest
mismatch, an insufficient tier, and a finding with **no** disposition, and routes
confirmed findings to the maintainer, whose PROMOTE with recorded dispositions is
the gate. Three releases have shipped that way. The design is sound and the
release path is not blocked by this record.

## What is actually owed

A systematic pass over the design record rather than another sampling round. The
brief's surface chapters were largely written ahead of the code and have been
corrected reactively ever since, which is why a fresh sample keeps finding
material. Candidate shapes, in rising order of cost:

- Grind the corpus chapter by chapter with the binary open, one chapter to a
  session, until a chapter's claims are all checked rather than all sampled.
- Move the checkable claims out of prose into generated tables, so a verb list, a
  flag list or a count cannot drift by being retyped.
- Mark, per chapter, the date its claims were last verified end to end, so a
  reader can tell a checked chapter from an unchecked one.

## Acceptance

- **Given** the design record after the pass, **when** the crosscheck runs at
  full tier, **then** the finding count is stable across two consecutive armings
  against an unchanged tree, rather than drawing a fresh sample each time.
- **Given** a chapter that has had the pass, **when** a reader opens it, **then**
  they can tell when its claims were last checked against the binary.
