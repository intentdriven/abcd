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
remedy: "Finish the systematic pass the body names, one chapter to a session with the binary open, changing a claim only after checking it at file:line against the code, so a fix cannot feed the tail. Then the orchestrator arms the full-tier brief-surface crosscheck twice against one unchanged tree, with the person's opt-in, and the record resolves when the two finding counts agree. Grounds: of the sixteen findings the v0.10.0 classification deferred here, eleven were still false of the prose at ad2b3f0d4, so the drift is real and checkable claim by claim; two armings against one tree are the acceptance criterion this record already states."
deferred_after: v0.11.1
deferral_reason: "the 16 deferred findings are paid (lane drainBrief); owed: two stable full-tier crosscheck armings, run by the orchestrator with the person's opt-in"
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

## Deferral 2026-09-29

Deferred past v0.11.1: the 16 deferred findings are paid (lane drainBrief); owed: two stable full-tier crosscheck armings, run by the orchestrator with the person's opt-in

## Deferred findings paid 2026-09-29

The v0.10.0 gate's receipt
(`.abcd/work/reviews/64ea8f62201970b9b242b59d0b7e7aab4c3d5baa/iss35-brief-surface-crosscheck.json`)
does not mark which findings its classification deferred here, so lane drainBrief
(autonomous run A) reconstructed the set by that classification's own test: a
`false-claim` or `stale-count` about the existence, name or set of a flag or
sub-verb, in 01-product, 02-constraints, 05-internals or the glossary. Exactly
sixteen findings meet it. Each was checked against the code at ad2b3f0d4; the
fixes are c6199530e, and the file:line cites the tree after it.

| Finding | Claim | Disposition |
|---|---|---|
| x-120 | `disembark <source-repo> to <dest>` | Already fixed before the base: every cited site reads `disembark pack <source-repo> <dest>`, the shape `internal/surface/cli/cli.go:968` registers |
| x-121 | a `launch dry-run` sub-verb | Fixed: `01-product/01-press-release.md:41`, `05-internals/09-provenance-substrate.md:83` name the `--dry-run` flag (`cli.go:380-388`; the preview exits 0, `cli.go:425`) |
| x-122 | bare launch shows status and help; a `dry-run` sub-verb | Fixed: `01-product/02-context.md:10` (bare launch refuses, `cli.go:388`) |
| x-123 | embark `scan` sub-verb | Fixed: `01-product/02-context.md:9` (embark registers `from` and `probe`, `cli.go:1218`, `cli.go:1236`) |
| x-124 | the seven-command "ships" list, `intent grill`, no `hold`/`unhold` | Fixed: `01-product/02-context.md:5` points at the generated CLI reference, `:11` and `:15` (`hold` and `unhold` at `cli.go:2464`, `cli.go:2491`) |
| x-125 | press release: `intent consistency`, `grill`, `refine`, `shape` shipped | True at the base: `01-product/01-press-release.md:26` marks `grill` and `shape` designed and not built, and `consistency` is registered (`internal/surface/cli/intent_consistency.go:26`) and files one capture per finding (`:16-21`) |
| x-127 | `embark from --archive`, `embark scan --deep` | Fixed: `02-constraints/01-platform.md:23`, `:28-29` (`from` takes no flags of its own, `cli.go:1236-1237`); the embark voyage record in the same paragraph was false too, and is marked a design target with them |
| x-128 | `memory ingest --reingest` | Fixed: `05-internals/09-provenance-substrate.md:48` (the flags are `--keep-original` and `--pages-json`, `cli.go:5526-5530`) |
| x-129 | a bundle is planned by `intent plan itd-A itd-B --bundle <name>` | True at the base: `cli.go:2259-2263`, `internal/core/intent/bundle.go:88`; the stale "itd-34, planned" markers beside it are fixed at `glossary/core/bundle.md:18` and `glossary/core/record-families.md:25` |
| x-130 | the family table names `intent reclassify`, `abcd build` and `drain`; the step entry names `abcd build` and `implement step` | True at the base: `cli.go:2588`, `internal/surface/cli/build.go:83`, `internal/surface/cli/drain.go:27`, `build.go:304`, and the family table no longer names them; `glossary/core/step.md:18` said the build lands steps, which the loop does not yet carry (`internal/core/implement/loop/loop.go:134-138`), fixed |
| x-136 | `intent grill` under the "ships" heading | Fixed: `01-product/02-context.md:15` marks it designed and not built |
| x-137 | press release: `consistency` and `shape` | True at the base, as x-125 |
| x-138 | `disembark ... to <dest>`; a `dry-run` sub-verb | Fixed: `01-product/02-context.md:8` names `plan` (`cli.go:943`); the same claim in `glossary/core/disembark.md:33`, fixed |
| x-139 | bare embark shows status and help; `scan` | Fixed: `01-product/02-context.md:9` |
| x-140 | bare launch; `/abcd:launch dry-run` in an acceptance criterion | Fixed: `01-product/02-context.md:10`, `01-product/01-press-release.md:41` |
| x-141 | intent's shipped sub-verbs omit `hold` and `unhold` | Fixed: `01-product/02-context.md:11` |

Found beside them and fixed: `glossary/core/voyage.md:71` named a file list, an
oracle backend and a verdict on each voyage line (ef1c8fee9; the line is
`internal/core/lifeboat/voyage.go:31-41`), and `glossary/distribution/release.md`
named a `launch ship --force` that the command does not have (c80832f97).

Seen and not paid here, each outside the sixteen: the `intent grill` usage
examples in `glossary/core/intent.md:37-38` and `glossary/core/persona.md:37`;
"11 adapters" in `01-product/04-scope.md`; bare `/abcd:ahoy` "shows status+help"
and `launch ship` "update the marketplace entry" in `01-product/02-context.md`;
and the other forty-two v0.10.0 findings in these four directories, among them
x-083 to x-100 in `02-constraints/04-naming.md` and
`05-internals/03-configuration.md`, which other branches were editing at the
time. The two stable armings the acceptance asks for are not run: they fan out
the pinned checkers and are the orchestrator's to arm, with the person's opt-in.
