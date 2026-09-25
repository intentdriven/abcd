---
id: itd-147
slug: the-brief-s-surface-chapters-are-a-generated-reflection-of-t
spec_id: spc-2609020906356450
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: additive
---

# The brief's surface chapters are a generated reflection of the shipped surface, so a shape claim cannot drift

## Press Release

> _Sequenced 2026-09-01 as a delivery rung of
> [Phase 8 — the brief is the shipped state](../../roadmap/phases/phase-8-brief-currency.md);
> the phase records the sync rule and its one legitimate lead._


> **The design record stops being able to lie about the shipped surface.** Each
> surface chapter in the brief carries a generated block — the verb's flags,
> sub-verbs, exit codes, schema fields and counts, derived from the command tree
> at build time and drift-tested like the CLI reference already is. The prose
> around it keeps doing what only prose can: saying why the surface exists, what
> it refuses to do, and which trade was made. A shape claim can no longer drift,
> because nobody writes one by hand.
>
> The measurement that prompted this is uncomfortable. A full-tier
> brief-to-surface crosscheck at the 0.6.2 release gate returned 147
> discrepancies across 23 chapters. Of those, exactly one touched a file the
> release bundle carries. The rest were the record describing a product that had
> moved: a PATH entry documented as a symlink when it has been an owned copy
> since spc-35, a user-scope directory tree missing two of the three directories
> that actually exist, a detection pass enumerating twelve steps where fourteen
> run.
>
> The same day, the same reviewers found **zero** discrepancies across the 775
> lines of `docs/reference/cli/commands.md`. That file is generated and
> drift-tested. That is the whole argument.
>
> "I read the brief to learn what the tool does before I touch it," said Maya, an
> autonomous-development practitioner whose agents read the record the same way.
> "When a chapter says the PATH entry is a symlink and the binary writes a copy,
> I do not discover a documentation bug. I write code against a surface that does
> not exist, and the mistake looks like mine."
>
> "We already knew how to fix this and had done it once," said Kira, who
> maintains the surface. "The CLI reference is generated, so it is right. The
> brief is typed, so it rots. We were keeping two copies of the same fact and
> hand-maintaining the one nobody could check."

## Why This Matters

The brief drifts steadily, not in bursts. Of 124 findings resolvable to a
blaming commit, 66 were last written in one month and 58 in the next — an even
rate, in every class. Roughly half the false claims survived four releases, each
of which recorded a PROMOTE receipt against this same detector at 26, 28 and 37
findings.

So the gate has been running and passing while the record got worse. That is the
proxy-gate class (iss-2608230847432286) with a demonstration attached, and the
deterministic half makes it sharper: `surface_coverage` blocked a commit during
this very release for a missing `history drain` **table row**, and passed green
while false prose claims sat beside those rows. Row presence stands in for
chapter correctness.

The diagnosis is that a chapter mixes two kinds of content with two different
failure modes in the same paragraphs:

- **Shape** — flags, sub-verbs, exit codes, schema fields, counts, file layouts.
  Derivable from the command tree, therefore checkable, therefore never worth
  hand-writing. `false-claim` and `stale-count` are shape by definition, and most
  `undocumented-surface` is the binary growing something a chapter's shape
  section never learned.
- **Intent** — why a surface exists, what it refuses to do, which trade was
  made. Not derivable, and structurally unable to drift against code, because it
  is not a claim about code shape.

Two copies of one fact, one of them authoritative: `one-canonical-primitive`
violated at two copies rather than three.

## Decisions (grilled 2026-09-01)

The maintainer resolved the open questions at the planning interview; these are
commitments, not options.

- **One generated appendix per chapter.** The block sits at the end of the
  chapter under a marker; the prose above it never states a flag or sub-verb.
- **A staged chapter gets an empty block that says unbuilt.** Every chapter
  carries a block; one whose surface has not shipped reads that there is no
  shipped surface, so the reader learns it from the same place as the flags.
- **Flags and sub-verbs only, first.** Exit codes and JSON output fields stay
  prose until the binary records them somewhere a generator can read.
- **The external-state category stays out.** The two-way split (derivable from
  the tree, or rationale) holds; a claim about external state is neither and is
  not this intent's to check.
- **Sequenced as Phase 8**, the brief is the shipped state, with itd-60 as the
  other rung; standalone kind, its own spec.

## What's In Scope

- **One generated appendix per chapter.** Each surface chapter carries a single
  generated block at its end, under a marker. The block is derived from the same
  command tree that already produces `docs/reference/cli/commands.md` and
  `.abcd/development/release/surface.json`, and a drift test fails when the
  committed block and the tree disagree.
- **Flags and sub-verbs, first.** The block carries the verb's flags and its
  sub-verbs, because those are what the command tree records today. Exit codes
  and JSON output fields stay prose until the binary records them somewhere a
  generator can read; the seam is built so admitting them later is a generator
  change, not a re-cut.
- **Every chapter carries a block, including a staged one.** A chapter whose
  surface has not shipped gets an empty block that says so — there is no shipped
  surface — so a reader learns the surface is unbuilt from the same place they
  read the flags, rather than from the block's absence.
- **A marker that keeps prose hand-written.** The generated region is delimited,
  so a chapter's rationale above the marker is never machine-authored and never
  clobbered, and the prose above the marker states no flag and no sub-verb.
- **The trust rule as an ADR plus a brief invariant:** shape claims are derived,
  never hand-authored.
- **Retiring the hand-written shape prose** the generated block replaces, chapter
  by chapter, so the 124 findings are closed by construction rather than by a
  sweep that starts rotting the next day.
- **`surface_coverage` stays, and says what it is.** It remains the row-level
  presence check over the surfaces index, labelled as such, so nothing reads it
  as a chapter-correctness gate.

## What's Out of Scope

- **A claim about external state.** A third category surfaced while this was
  drafted and fits neither half of the split: a claim that is checkable in
  principle but has no local source to check against — the worked example is
  `wrangler.jsonc` recording a dashboard setting with no durable local home,
  where eight of eight pull requests on 2026-08-23 built through an integration
  the record says is off. It is the most dangerous of the three shapes, because
  the plan to change the external state and the assertion that it had changed
  were written by the same hand in the same file, and it reads as a completed
  decision rather than as an unchecked claim. Neither generation nor prose
  reaches it. The two-way split stands for this intent; the category is held by
  iss-2608231607594913 (refining iss-2608220150157502) and a remedy that reads
  the real external state is that record's to propose.
- **A "you changed a surface, so touch its chapter" gate.** It forces an edit
  without forcing correctness, which is a phantom gate — worse than none, and the
  precise class this intent is an instance of.
- **A new principle.** `one-canonical-primitive` already says it; this is an
  application, and the confirmed decomposition records "none new" deliberately.
- **Generating the rationale.** Prose about why is the half that cannot drift and
  the half worth a human. This intent therefore does not make a hand-maintained
  behavioural claim safe: ordering guarantees, failure semantics and what a verb
  refuses stay prose and stay hand-maintained.
- **Fixing the 147 by hand ahead of the seam.** The corpus is evidence; a hand
  sweep destroys the dataset and buys a clean brief that drifts again at the
  measured rate.

## Mechanism

> _Facilitator-seeded from the measurement this intent rests on. The maintainer
> stated no mechanism claim at the interview; strike this line and the paragraph
> under it if it is not the claim being made._

We expect this to work because it has already worked here, once, on the same
class of content: `docs/reference/cli/commands.md` is generated from the command
tree and pinned by `TestReferenceMatchesCommittedPage`, and it returned zero
discrepancies on the day the hand-written brief returned 147. The mechanism needs
no new infrastructure — the generator, the snapshot and the drift test all exist
and run every CI pass — and a `false-claim` or `stale-count` finding about a
generated block in the next crosscheck run is what would show it wrong.

## Scope Conditions

None stated.

## SOTA

Doc-generation from a command tree is the presumptive answer and is what cobra
ecosystems do by default. The adversary-filtered question is not whether to
generate but where to cut, because a fully generated chapter loses the rationale
that makes the brief worth reading. The cut proposed here — generated shape,
hand-written why — is the narrow version, chosen because the measurement says
shape is where the drift is.

## Acceptance Criteria

- **Given** a surface chapter with a generated appendix under its marker,
  **when** a verb gains a flag or a sub-verb and the appendix is not regenerated,
  **then** the drift test fails and names the chapter and the missing claim.
- **Given** a chapter's hand-written rationale above the marker, **when** the
  appendix is regenerated, **then** nothing above the marker changes.
- **Given** a chapter whose surface has not shipped, **when** the generator runs,
  **then** the chapter carries an empty block under the same marker stating there
  is no shipped surface, so the reader learns the surface is unbuilt from where
  they would have read the flags.
- **Given** the generated appendix, **when** it is composed, **then** it carries
  the verb's flags and sub-verbs and nothing else — an exit code or a JSON output
  field stays prose until the binary records it where a generator can read it.
- **Given** a surface chapter's prose above the marker, **when** it is checked,
  **then** it states no flag and no sub-verb; the shape claims live only in the
  appendix.
- **Given** a full-tier brief-to-surface crosscheck run after the seam lands,
  **when** its findings are classified, **then** no finding is a `false-claim` or
  `stale-count` about a flag or a sub-verb the generated appendix covers.
- **Given** `surface_coverage`, **when** it reports, **then** it names itself as
  the row-level presence check over the surfaces index and claims nothing about
  whether a chapter's prose is correct.
- **Given** the crosscheck's non-reproducibility (iss-2608231409595789),
  **when** progress is assessed, **then** it is assessed on which chapters are
  implicated rather than on a finding count, because the count is not a metric.

## Open Questions

> _Every question below was put to the maintainer at the planning interview on
> 2026-09-01 and answered there; the answers are the rulings in the Decisions
> section above. The section is kept so a later reader sees what was asked._

- **Resolved — where the seam is.** One generated appendix per chapter, at the
  end, under a marker. Not per section: the per-section cut preserves the most
  rationale and costs the most machinery, and the appendix keeps the prose
  hand-written for a fraction of it.
- **Resolved — whether the snapshot carries enough.** Flags and sub-verbs only,
  first. Exit codes and JSON output fields are the gaps, and they stay prose
  until the binary records them somewhere a generator can read, rather than being
  invented into the snapshot to fill the block.
- **Resolved — a chapter documenting a staged surface.** It gets an empty block
  that says there is no shipped surface. Every chapter carries a block, so a
  reader never has to infer anything from a block's absence.
- **Resolved — what happens to `surface_coverage`.** Left as the row-level check
  it is, and labelled as that. Leaving it was always defensible; leaving it
  unlabelled is what let it read as a chapter-correctness gate.
- **Resolved — the third, external-state category.** Recorded as out of scope
  above. The two-way split stands for this intent, and the category is held by
  iss-2608231607594913, refining iss-2608220150157502.

## Audit Notes

Filed from `iss-2608231346137587`, whose Routing section carries the four-piece
decomposition confirmed by the maintainer on 2026-08-23 (verdict SPLIT), graded
into the dated decomposition-calibration corpus. Capability here; trust rule to
an ADR plus a brief invariant; stance deliberately none new; plumbing to the
brief.

The evidence base is three full-tier crosscheck runs preserved out-of-tree with
their content commits, manifest hash and tier. Read it with
`iss-2608231409595789` in hand: the runs returned 125, 126 and 147, and runs 2
and 3 are a controlled comparison — the brief is byte-identical between their
commits, so the detector's whole subject was held constant and it still returned
17% more findings. Which chapters are implicated is broadly stable; how many and
of what class is not.

<!-- abcd-review: INGESTED receipt=rcp-2769c7a59830 -->
Fidelity review — receipt rcp-2769c7a59830 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:f360a0544725b3e1f3d6e1cd643a8db5c614b5927043c3e33f22721caf04a0db
Input attestations: diff:4480c5a8^1..4480c5a8 (PR #671 merge, reachable from e2ab1ddc)@sha256:f5b5a456f7bc0f072a578eb663a9d4c3843ec8ae412aaf0abed0264d0a5a2221;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 1

Per-criterion verdicts:
- ac-1 — MET: TestSurfaceAppendicesMatchCommandTree regenerates every chapter from the live tree and Drift() names the chapter and each missing line; on a scratch copy of HEAD, adding a `colour` flag to `version` without regenerating failed the test with `04-surfaces/12-version.md` and `missing: | --colour | bool |`
  evidence: internal/surface/cli/brief_appendix_test.go:15 — "func TestSurfaceAppendicesMatchCommandTree(t *testing.T)"
  evidence: internal/core/surface/appendix.go:643 — "the generated appendix disagrees with the command tree"
  evidence: internal/core/surface/appendix_test.go:199 — "func TestAppendixDriftNamesTheMissingClaim"
- ac-2 — MET: RenderChapter returns the bytes above the opening marker unchanged plus the regenerated region, markerLines refuses prose after the end marker, and the idempotence test asserts the prose prefix is byte-identical across two regenerations
  evidence: internal/core/surface/appendix.go:243 — "func RenderChapter(text, appendix string) (string, error)"
  evidence: internal/core/surface/appendix_test.go:147 — "func TestRenderChapterPreservesProseAndIsIdempotent"
- ac-3 — MET_WITH_CONCERNS: The staged reflect chapter carries the markers around the declared unbuilt sentence and a test pins the exact text; the concern is that the same sentence is emitted for the three host-delegated commands whose register rows read `shipped` (consult, ingest, prepare-this-repo), so a generated block states there is no shipped surface where the register says there is one
  evidence: .abcd/development/brief/04-surfaces/09-reflect.md:119 — "There is no shipped surface: the command tree registers no `abcd reflect` verb"
  evidence: internal/core/surface/appendix.go:71 — "func UnbuiltSentence(path string) string"
  evidence: .abcd/development/brief/04-surfaces/13-consult.md:144 — "There is no shipped surface: the command tree registers no `abcd consult` verb"
  evidence: .abcd/development/brief/04-surfaces/README.md:28 — "| 13 | `/abcd:consult` | shipped |"
- ac-4 — MET: ComposeAppendix is handed paths, flags and sub-verbs only, and the cobra-tree test asserts an exit-code annotation, a JSON schema and an example in a verb's metadata never reach the appendix
  evidence: internal/core/surface/appendix.go:87 — "func ComposeAppendix(paths []string, tree []Command) string"
  evidence: internal/surface/cli/brief_appendix_test.go:84 — ""Exit codes:", "3 refused", "schema_version", "widgets.v1""
  evidence: internal/core/surface/appendix_test.go:128 — "func TestComposeAppendixCarriesOnlyFlagsAndSubVerbs"
- ac-5 — MET_WITH_CONCERNS: TestSurfaceChapterProseStatesNoShape passes over all 25 chapters and, on a scratch copy, fired at `12-version.md:8` when `--check` returned to the prose; two ruled narrowings remain: the `## Sub-verbs` table above the marker still states sub-verbs by hand (exempt, checked by surface_coverage), and the check fires only on flags the tree registers, so a flag abcd never or no longer registers passes as another program's
  evidence: internal/surface/cli/brief_appendix_test.go:35 — "func TestSurfaceChapterProseStatesNoShape(t *testing.T)"
  evidence: internal/core/surface/appendix.go:357 — "if inSubVerbs && (strings.HasPrefix(t, "|") || strings.HasPrefix(t, ">"))"
  evidence: internal/core/surface/appendix.go:367 — "if s := text[m[4]:m[5]]; longs[s]"
  evidence: .abcd/development/decisions/adrs/2609231028044006-surface-chapter-shape-claims-are-derived-never-hand-authored.md:60 — "The chapter's `## Sub-verbs` table and its standard note are the one exception"
- ac-6 — INCONCLUSIVE: No full-tier brief-to-surface crosscheck has run after the seam landed; the delivery records the run as owed at the next release gate and nothing in the diff or the tree classifies post-seam findings, so the outcome cannot be verified from the inputs
  evidence: .abcd/development/release-gate/README.md:63 — "Owed at the next release gate: itd-147's ac-6."
  evidence: .abcd/development/research/data/2026-08-23-brief-surface-crosscheck/README.md:110 — "That run is the next release gate's crosscheck."
- ac-7 — MET: Every surface_coverage finding from both the index pass and the sub-verb pass is prefixed with SurfaceCoverageLabel, which names the row-level presence check and disclaims chapter prose, and two tests pin the prefix and the label's wording
  evidence: internal/core/lint/lint.go:725 — "const SurfaceCoverageLabel = "row-level presence check over the surfaces index and each chapter's sub-verb table (it judges rows, not whether a chapter's prose is correct): ""
  evidence: internal/core/lint/lint.go:430 — "findings = append(findings, labelSurfaceCoverage(sc)...)"
  evidence: internal/core/lint/subverbs_test.go:361 — "func TestSubVerbFindingsCarryTheRowLevelLabel"
- ac-8 — MET_WITH_CONCERNS: The research note records the baseline as a per-chapter table across the three runs and states that the stable signal is the set of files rather than the numbers; the concern is that the post-seam assessment itself has not been made, because it depends on the crosscheck run ac-6 still owes
  evidence: .abcd/development/research/data/2026-08-23-brief-surface-crosscheck/README.md:69 — "## Chapters implicated: the baseline itd-147 is assessed against"
  evidence: .abcd/development/research/data/2026-08-23-brief-surface-crosscheck/README.md:104 — "Twenty-three files are implicated across the three runs."

Gap audit:
- honoured:
  - one generated appendix per chapter, at the end, under a marker pair; all 25 chapter files carry exactly one begin and one end marker
    evidence: internal/core/surface/appendix.go:40 — "AppendixBegin = "< !-- surface-appendix:begin"
    evidence: .abcd/development/brief/04-surfaces/12-version.md:73 — "< !-- surface-appendix:begin"
  - the appendix is derived from the same command-tree walk that builds the compatibility snapshot, and one generator writes both
    evidence: internal/surface/cli/brief_appendix.go:17 — "surface.RegenerateChapters(dir, commandSurface(NewRootCommand()))"
    evidence: cmd/abcd-gen-surface/main.go:60 — "chapters, refused := cli.SurfaceChapters(root)"
  - a drift test fails `go test` when the committed block and the tree disagree, naming the chapter and the claim
    evidence: internal/surface/cli/brief_appendix_test.go:24 — "if d := ch.Drift(); d != "" {"
  - the hand-written flag and sub-verb prose was retired chapter by chapter and a check keeps it out
    evidence: .abcd/development/brief/04-surfaces/12-version.md:8 — "The one exception is the opt-in online check"
    evidence: internal/surface/cli/brief_appendix_test.go:47 — "for _, c := range surface.ProseShapeClaims(prose, ch.Commands, tree)"
  - the trust rule is recorded as an ADR and as brief invariant 18
    evidence: .abcd/development/decisions/adrs/2609231028044006-surface-chapter-shape-claims-are-derived-never-hand-authored.md:38 — "Every shape claim the brief's surface chapters"
    evidence: .abcd/development/brief/02-constraints/03-invariants.md:49 — "18. **Shape claims in the brief's surface chapters are derived, never hand-authored**"
  - surface_coverage stays and names itself the row-level presence check
    evidence: internal/core/lint/lint.go:725 — "row-level presence check over the surfaces index"
    evidence: .abcd/development/brief/04-surfaces/README.md:48 — "the row-level presence check over this index"
  - a staged chapter carries an empty block that says there is no shipped surface
    evidence: .abcd/development/brief/04-surfaces/09-reflect.md:119 — "There is no shipped surface"
  - a malformed or unregistered chapter is refused by name and the rest are still regenerated
    evidence: internal/core/surface/appendix_test.go:362 — "func TestRegenerateChaptersSkipsAndReportsARefusedChapter"
- diverged:
  - the unbuilt sentence was promised for a chapter whose surface has not shipped; it is also emitted for the three shipped host-delegated commands, so their generated block contradicts their `shipped` register row
    evidence: .abcd/development/brief/04-surfaces/13-consult.md:144 — "There is no shipped surface: the command tree registers no `abcd consult` verb"
    evidence: .abcd/development/brief/04-surfaces/14-ingest.md:95 — "There is no shipped surface"
    evidence: .abcd/development/brief/04-surfaces/15-prepare-this-repo.md:153 — "There is no shipped surface"
    evidence: .abcd/development/brief/04-surfaces/README.md:191 — "They are **host-delegated** commands, with a command page and no Go verb"
  - the prose above the marker was promised to state no sub-verb; the `## Sub-verbs` table and its note above the marker still state sub-verbs by hand, as a ruled exception checked by surface_coverage
    evidence: .abcd/development/decisions/adrs/2609231028044006-surface-chapter-shape-claims-are-derived-never-hand-authored.md:60 — "table and its standard note are the one exception"
    evidence: internal/core/surface/appendix.go:307 — "The `## Sub-verbs` section's table and its standard blockquote note are exempt"
  - the prose check fires only on flags the command tree registers, so a flag spelt in prose that abcd never or no longer registers is treated as another program's and passes
    evidence: internal/core/surface/appendix.go:367 — "if s := text[m[4]:m[5]]; longs[s]"
    evidence: .abcd/development/decisions/adrs/2609231028044006-surface-chapter-shape-claims-are-derived-never-hand-authored.md:59 — "Another program's flag (git's `--force`) and the same words as"
  - every chapter carries a block holds for files in 04-surfaces only; the staged worktree surface documented in the internals chapter carries none
    evidence: .abcd/development/brief/04-surfaces/README.md:40 — "| 25 | `/abcd:worktree` | staged |"
    evidence: internal/core/surface/appendix.go:521 — "!strings.ContainsAny(lt[1], "/#")"
- missing:
  - a full-tier crosscheck run after the seam, classified so that no finding is a false-claim or stale-count about a covered flag or sub-verb (ac-6), and the chapter-implicated assessment against it (ac-8)
    evidence: .abcd/development/release-gate/README.md:63 — "Owed at the next release gate: itd-147's ac-6."
    evidence: .abcd/development/release-gate/README.md:70 — "Delete this paragraph in the change that records the result."

### ac-6 outcome check at the v0.10.0 release gate (2026-09-24)

The first full-tier brief-surface cross-check after the generated appendix
landed ran its 40 pinned checkers over fa744b41 (content commit 64ea8f62,
receipt `.abcd/work/reviews/64ea8f62201970b9b242b59d0b7e7aab4c3d5baa/iss35-brief-surface-crosscheck.json`).
Of its 154 unique findings, an independent classification found 17 that are a
`false-claim` or `stale-count` about the existence, name or set of a flag or
sub-verb of a command the appendix lists. **ac-6 is NOT MET**: one of the 17
(x-049, captured as iss-2609240519422232) sits in `04-surfaces/14-ingest.md`
above the appendix marker, where this intent's rule that the prose states no
flag and no sub-verb applies, and the prose there enumerates ingest sub-verbs
without `history ingest`. The other 16 sit in 01-product, 02-constraints,
05-internals and the glossary, which carry no appendix, so they are outside
the seam this intent installed; they are deferred with the rest of the brief
drift to iss-2609091956001547. The seam held in 34 of the 35 brief documents
it covers; the drift its press release claims to end persists in the chapters
it does not reach.

### ac-6 met by the ingest chapter fix (2026-09-25)

**ac-6 is met by commit 7dafb541**, which resolves iss-2609240519422232. The
one finding of the 17 that sat inside the seam (x-049) was the ingest chapter's
prose above its appendix marker enumerating ingest sub-verbs without
`history ingest`. The fix does not add the missing verb, because this intent's
rule is that the prose there states no sub-verb: the paragraph names none
and defers the list to the generated CLI reference. With it, none of the 17
false-claim or stale-count findings the v0.10.0 classification found sits in a
chapter's prose above an appendix marker. The other 16 stay outside the seam
and deferred to iss-2609091956001547. The next full-tier crosscheck is what
would show this wrong: a false-claim or stale-count about a flag or sub-verb
the appendix covers, found above a marker.

## Grounds

- pursued: a shape claim nobody writes by hand cannot drift; a false-claim or stale-count finding about a generated block in the next crosscheck run shows it was wrong
