---
id: spc-64
slug: the-read-block-eval-falsifies-the-firewall-planted-warm-cont
intent: itd-186
---
# The read-block eval falsifies the firewall

## Summary

spc-64 delivers [itd-186](../../intents/shipped/itd-186-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md):
a repository eval that plants sentinel warm content across every warm location
class in a fixture repository state, runs the cold-reading input assembler
([itd-183](../../intents/shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md),
spec [spc-61](../closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md))
over it, and asserts that no sentinel and no excluded field reaches the
assembled input. It is the only component in the workstream capable of
falsifying the read-block rather than restating it, so its oracle is the planted
corpus, never the assembler's include table.

The eval lands in the existing smoke harness package
([`evals/`](../../../../evals/README.md)), behind the `smoke` build tag, so
`make smoke` and CI's `smoke` job pick it up with no change to
[`Makefile`](../../../../Makefile) or
[`.github/workflows/ci.yml`](../../../../.github/workflows/ci.yml). Passing in
CI is a cycle exit criterion. spc-61 is being written concurrently; where it is
still a stub, this spec designs against itd-183's own text and the assembler row
of the cycle build plan, and states under [Approach](#approach) the four demands
it places on the assembler's interface, which are deliberately the whole of the
coupling.

## Scope

In: the sentinel fixture corpus and its plants, one per warm location class; the
independent oracle and the import guard that keeps it independent; the
field-level and content-level assertions over the assembled input at every
position the assembler accepts; the negative-control fixture that proves the
eval fails when the firewall is holed; the anti-vacuity guard that proves every
plant is still planted. Out: everything under
[Out of scope](#out-of-scope), and in particular any assertion about what a
reading *produced*, which belongs to the output contract
([itd-185](../../intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md)).

## Approach

### The oracle, and how it stays independent of the include table

Four mechanisms, each structural rather than promised:

- **Out of process.** The eval invokes the assembler through the built binary,
  reusing the harness's existing `TestMain` build and `run` helper in
  [`evals/smoke_test.go`](../../../../evals/smoke_test.go). It never links the
  assembler's package, so it cannot read the include table even by accident.
- **An import guard.** `TestOracleImportsNothingFromTheAssembler` parses every
  Go file in `evals/` with `go/parser` and fails if any import path names the
  assembler's package or `internal/core/launch`'s include list. The
  independence claim is therefore checked, not asserted in a comment.
- **A hand-transcribed exclusion table.** The eval's table of excluded
  frontmatter keys, headings, and record families is transcribed by hand from
  itd-183's exclusion list, brief invariants 14 and 15
  ([`03-invariants.md`](../../brief/02-constraints/03-invariants.md)), and
  [adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md).
  Transcription is the point: a generated table would agree with the assembler
  by construction and could only ever confirm it.
- **A content oracle, not a path oracle.** Every assertion is over the bytes and
  the parsed fields of the assembled input. A plant that moves is still caught,
  because nothing in the assertion mentions where the plant was.

### The fixture corpus

Fixtures live under `evals/testdata/cold-reading/`, not `evals/data/`: the
latter is reserved for the auto-discovered per-command shape of itd-75, and
`testdata` is the tree the Go toolchain already excludes from builds. Three
variants, all inert on disk and materialised into a temporary directory per run:

- `baseline/`: a whole miniature repository state, every plant in its canonical
  home. The eval copies it out, runs `git init`, and makes one commit under a
  fixture identity in a reserved example domain, so the manifest's commit
  reference resolves. `HOME` is redirected to a temporary home carrying a
  planted transcript store at `.abcd/history/<root-sha>/`, keyed on the
  fixture's own root-commit SHA, so the transcript class of invariant 15 is
  genuinely reachable if the assembler ever walked it.
- `holed/`: the negative control. Two plants are relocated into positively
  included material, one into a `01-product/` brief chapter and one into
  `docs/`, which the shipped tree includes wholesale. A correct assembler passes
  both through, and the eval must report exactly those two violations.
- `order/`: shared with spc-65, which owns its contents.

Plants, one per warm location class, each carrying a distinct token of the form
`ABCD-EVAL-SENTINEL-<CLASS>` so a failure names the class that leaked:

| Warm location class | Where the baseline plants it |
|---|---|
| Framing traces, declined construals, and the local ledger side | `.abcd/.work.local/ledger/` |
| Transcript class | `.abcd/.work.local/scratch/`, and the fixture home's `.abcd/history/<root-sha>/transcripts/` |
| Decisions, the issue ledger in all three states, a wontfix reason, and a lapse entry | `.abcd/work/DECISIONS.md`, an ADR, and `.abcd/work/issues/{open,resolved,wontfix}/` |
| Deliberation chapters and superseded records | `brief/03-evidence/`, `intents/superseded/`, `plans/`, `research/notes/`, and `roadmap/rfcs/` |
| Warm fields on an *included* record type | An Audit Notes heading and a scope-condition disposition on a shipped intent |
| Warm frontmatter keys on *included* record types | `origin` on a spec and a discipline, and the production-mode key on a brief chapter |
| The instrument's own exhaust | A prior manifest under `.abcd/development/readings/<run-id>/`, a prior reading record under `.abcd/work/issues/readings/<run-id>/`, and its disposition |
| Grounds records | Admission and selection grounds on the selection surfaces |
| Draft-intent body, position-dependent | `intents/drafts/`, carrying a second token in its `origin` key |

### Assertions

Per position, over the assembled input the assembler wrote:

1. **Sentinel absence:** No planted token appears in the raw serialisation.
2. **Field absence:** A recursive walk of the parsed document reports no key on
   the excluded-key list at any depth, and no excluded heading in any projected
   body. Depth-agnostic key matching is what makes a warm field landing in a new
   place a failure, which a path assertion would miss.
3. **Family absence:** No item's recorded path lies inside an excluded record
   family, checked against the eval's own list of families.

The draft plant is the one position-sensitive case: its body token is cold at
entailment and warm at widening, so assertion 1 exempts it at entailment alone,
while its `origin` token stays banned everywhere. That is itd-183's drafts
asymmetry, tested rather than trusted.

### Decisions this spec makes that the record did not

- **The eval binds to the verb, not the package**, for the independence reason
  above. The verb's name lives in one constant at the top of the eval file, the
  single place a rename touches.
- **Four demands on spc-61's interface**, stated here so spc-61 can meet them:
  a dry-run invocation that writes the assembled input and the manifest as two
  separate artefacts into an operator-named output directory; a JSON
  serialisation of the assembled input, per brief invariant 4; selection of the
  position and the target state as flags, with no free text, per itd-183's ruled
  interface; and a non-zero exit on failure.
- **Positions exercised: the full set of assertions at every position that
  assembles.** Widening, entailment and detection each carry all three
  assertions over every sentinel class, with only the draft body's
  position-dependent exemption above carved out. Carving a whole position out
  of the bundle assertions is what this spec's first review round refused: it
  left six of the ten sentinel classes unasserted at that position and left the
  oracle's own drafts-at-comparative exclusion a row that could never fire, so
  an assembler admitting the local ledger tier at that one position read as
  green. The comparative position is outside the set for a different reason —
  it does not assemble at all. Its object is the widening reading's
  pre-admission output, which is not repository material and which no channel
  supplies, so the verb refuses
  ([itd-199](../../intents/shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md)).
  Holding that position is `TestComparativeRefusesToAssemble` in
  [`evals/coldreading_test.go`](../../../../evals/coldreading_test.go), which
  asserts the refusal itself and asserts that neither bundle nor manifest is
  written: a bundle that is never written cannot leak, which is strictly
  stronger than an absence assertion over emitted bytes, and the test fails
  loudly if the position starts assembling again without this eval being told.
  The two position sets are named once, as `assemblingPositions` and
  `fullyAsserted` in
  [`evals/coldreading_fixture_test.go`](../../../../evals/coldreading_fixture_test.go),
  so a table that enumerates positions cannot silently drop one.
- **The corpus is owned here** and shared with spc-65, so the two evals cannot
  drift apart in the repository state they exercise.

### Wiring

The eval is Go test files in package `evals` with `//go:build smoke`, so
`make smoke` (`go test -tags smoke ./evals/...`) runs it and CI's `smoke` job,
a required status check on the protected branch, runs `make smoke`. No Makefile
or workflow edit is required. One caveat is recorded rather than worked around:
the diff classifier stands the `smoke` job down on a pull request confined to
`docs/`, `.abcd/development/`, `.abcd/work/`, and the root prose files, so a
record-only pull request does not exercise this eval. The merge-queue entry runs
the full set on the would-be merge result, so the property still gates the merge.

**Ruled by the maintainer, 2026-08-30 — the evals get their own always-run CI
job.** The `smoke` job stands down on a pull request confined to `docs/`,
`.abcd/development/`, `.abcd/work/` and the root prose files, and those are
precisely the paths these evals read: a record-only change is the diff most able
to introduce warm content into material the assembler includes, so standing the
eval down there is anti-correlated with the risk. A stood-down job still reports
its check context green, which is a green for work that did not happen.

The remedy follows the reasoning `ci.yml` already documents for the ubuntu unit
lane, which never stands down because its tests read the live tree under the
allowlist. So: a small CI job carrying no `inert` condition, running these two
evals alone behind their own make target, while the rest of the smoke harness
keeps standing down. The workflow edit lands with this spec's build, reviewed
alongside the evals it serves; the merge-queue entry continues to run everything
regardless.

## Acceptance criteria mapping

The criteria were split and sharpened on 2026-08-31, before this spec was
built. The numbering below is the positional authority ac-1..ac-6.

| itd-186 criterion | How spc-64 satisfies it | Pinned by |
|---|---|---|
| ac-1 — the baseline passes only if no planted warm content and no excluded field reaches the output | Assertions 1 to 3 run over the baseline fixture at every asserted position | `TestReadBlockBaselineIsClean` |
| ac-2 — a relocated plant fails with a non-zero exit and a message naming the class token and the position | The content oracle names no path, so a relocated plant is caught wherever it lands; the `holed/` variant relocates two plants into included material and the failure message carries the class token and position | `TestReadBlockCatchesAHoledFirewall` |
| ac-3 — a warm field on an already-included record type fails | The `origin` and production-mode plants sit on included record types, and the Audit Notes plant sits inside an included file; each is a real leak path if the key filter or the projection is absent | `TestReadBlockCatchesWarmFieldsOnIncludedTypes` |
| ac-4 — no prior-run manifest or reading record reaches the output | The exhaust plants, asserted at every position that assembles; comparative, the position most likely to be handed the instrument's own output, refuses instead and is held by `TestComparativeRefusesToAssemble` | `TestPriorRunExhaustNeverReaches` |
| ac-5 — no file under `evals/` imports the assembler's package or its include list | `go/parser` over every Go file in `evals/`, so the independence claim is checked rather than asserted in a comment | `TestOracleImportsNothingFromTheAssembler` |
| ac-6 — every declared sentinel class is planted its declared number of times | The anti-vacuity guard: an absence assertion cannot see a corpus that lost a plant, so the corpus is asserted separately from the absence | `TestEverySentinelIsPlanted` |

ac-5 and ac-6 were added when the criteria were split: the press release claims
oracle independence and the spec makes it structural, but no criterion measured
it, and an absence eval with no anti-vacuity guard is the failure mode this
workstream has already found five times.


## Tests

Every test below is watched red before the change and green after.

- `TestReadBlockBaselineIsClean` is watched red against an assembler whose
  exclusion of the local ledger tier is temporarily removed by a one-line local
  patch. The sentinel that must be caught is
  `ABCD-EVAL-SENTINEL-LEDGER-FRAMING`; the run is recorded in the pull request
  body, and the patch is reverted before the branch is pushed.
- `TestReadBlockCatchesAHoledFirewall` is red until the `holed/` variant exists,
  and it is the permanent proof that the assertion can fail: it demands exactly
  two violations, naming the classes, so an assertion that stops detecting
  anything fails here rather than passing quietly.
- `TestEverySentinelIsPlanted` walks the materialised fixture and fails if any
  class's token is absent or planted more than its declared number of times. It
  is the anti-vacuity guard: without it, a corpus that lost a plant would pass
  silently, the one failure mode an absence assertion cannot see.
- `TestOracleImportsNothingFromTheAssembler` is red the moment the eval's own
  package imports the assembler, which keeps the independence claim true after
  this spec closes.
- `TestPriorRunExhaustNeverReaches` and
  `TestReadBlockCatchesWarmFieldsOnIncludedTypes` are watched red against the
  same temporary hole technique, each removing the single exclusion it covers.

## Out of scope

- The assembler itself, its include table, its projection, and its manifest
  (spc-61); the four reading definitions and the blindness core
  ([spc-62](../closed/spc-62-four-cold-reading-definitions-one-blindness-core-each-positi.md));
  validating what a reading was licensed to produce
  ([spc-63](spc-63-one-ingest-verb-validates-every-cold-reading-output-includin.md));
  and determinism of the assembled input
  ([spc-65](spc-65-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md)).
- Prose-borne warmth inside an included chapter, which carries no structural
  signal. The eval catches a planted token wherever it sits, so it is stronger
  than the assembler's filter here, but unplanted prose warmth stays the
  disclosed residue itd-183 records, and so does the timing and
  target-selection leakage that intent discloses.
