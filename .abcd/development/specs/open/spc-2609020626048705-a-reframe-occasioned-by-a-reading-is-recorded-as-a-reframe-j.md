---
id: spc-2609020626048705
slug: a-reframe-occasioned-by-a-reading-is-recorded-as-a-reframe-j
intent: itd-2609020625402518
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The reframe record: `capture reframe` writes one `rfm-N` keyed on the committed fingerprints of the three frame surfaces before and after, joined to the reading item, disposition or surprise that occasioned it

## Summary

spc-2609020626048705 delivers
[itd-2609020625402518](../../intents/planned/itd-2609020625402518-a-reframe-occasioned-by-a-reading-is-recorded-as-a-reframe-j.md).
A new record family, `rfm-N`, lives flat under `.abcd/work/issues/reframes/`
beside the surprise family, one record per reframe occasioned by a reading. It
carries the occasion, the SHA-256 of each of the three frame surfaces as they
stood before and after the rewrite, which of the three changed, and the
grounds, and nothing of the abandoned framing's text, which stays local under
[adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md).
`abcd capture reframe` is its only writer. It reads the three surfaces itself,
at `HEAD` and in the working tree, so the operator supplies no hash; when the
record precedes the commit it writes the first half and says so, and a second
invocation completes it after the commit. The occasion is asserted by the
operator and checked in one respect, that it predates the rewrite. The family
is warm, reaches no reading, and its exclusion is asserted in every manifest.
`abcd rfm-N` reports the occasion, the fingerprints and which surfaces moved.
The frame this record keys on is the framing as it presently stands, which
adr-55 enumerates as three committed surfaces, in the form
[adr-2609021016288378](../../decisions/adrs/2609021016288378-a-reframe-occasioned-by-a-reading-is-a-committed-pointer-to.md)
adopted on 2026-09-02: The construal section of the framing chapter, the
committed glossary terms, and the committed scope. The single-section
narrowing an earlier draft of this spec flagged is withdrawn.

## Scope

In: The family's constants and schema in `internal/core/issueschema`, the
directory constant registered with the ledger's directory list; the writer,
the three surface readers and the fingerprint in
`internal/core/capture/reframe.go`; the
`capture reframe` sub-verb and its plugin page section; the record store in
`.abcd/record-lint.json` and `internal/core/lint/schema.go`; an exclusion-floor
row, the regenerated charter and a planted sentinel in the read-block eval; the
dispatcher in `internal/core/record`.

Out: The prior text of any surface; reframes no reading occasioned; a lint
over frame edits; the fourth audit verdict; a frame that lives at other paths
than the three named below.

## Approach

### The family: prefix, store and shape

The prefix is `rfm`, fixed here as the intent left it to the spec; it collides
with no family in `recordid`, `issueschema` or the record-lint stores, and
`grep` finds no `rfm-` in the tree. `issueschema/reframe.go` declares
`ReframeFamily = "rfm"`, `ReframesDir = "reframes"` (flat, like
`SurprisesDir`: The record is keyed by what it carries, not by a directory),
`ReframeRequired = ["schema_version", "id", "occasioned_by", "construal_before", "glossary_before", "scope_before", "grounds"]`,
`ReframeKnown` as that set plus `construal_after`, `glossary_after`,
`scope_after` and `changed`, and
`ReframeOccasionFamilies = [ReadingItemFamily, DispositionFamily, SurpriseFamily]`.
`ReframesDir` (registered in `issueschema.LedgerDirs()`, the one list the comparative exclusion rows and the scribe allow list derive from) is registered in the ledger's directory list in `issueschema`,
beside `ReadingsDir`, `DispositionsDir`, `AdmissionsDir` and `SurprisesDir`,
which is the list the comparative channel derives its per-directory exclusion
rows from at the comparative position and the scribe derives its allow list
from; registering the constant is what makes both pick the family up without
a literal edit in either. The record is frontmatter only, rendered through
`buildIssueText(fields, "")` as a disposition is: `schema_version: 1`, `id`,
`occasioned_by`, `construal_before`, `glossary_before`, `scope_before`, and
`construal_after`, `glossary_after`, `scope_after` (each absent while the
record is open, a 64-hex scalar once complete), `changed` (absent while open;
once complete a flow sequence naming the surfaces whose fingerprints differ,
drawn from `construal`, `glossary`, `scope`, and never empty, because a
completion in which no surface moved is refused), `grounds` (free text, held to
`grounds.ValidateText` over `grounds.Fold`, the floor every grounds-shaped
field is held to, then redacted through the ledger redactor before it is
written). The disclosure pair is not carried: It belongs to intent, spec and
issue records, and both pages say records of other families carry neither.
The store path is `.abcd/work/issues/reframes/rfm-<N>.md`, minted through
`recordid.Minter.Mint(issueschema.ReframeFamily)` under `withLedgerLock`,
provisioned through `safeMkdirLeaf`, and refused if the path already exists.

### The three frame surfaces and their fingerprints

The frame is the framing as it presently stands, which adr-55 enumerates as
three committed surfaces and adr-2609021016288378 fixes: The construal
section, the committed glossary terms and the committed scope, each at a
fixed path, per the intent's first scope condition. `capture` declares them as one table, `FrameSurfaces`, of
`{Name, Path, Heading string}`:

- `construal`: `.abcd/development/brief/01-product/06-framing.md`, the H2
  titled `Construal`. `capture.ConstrualFingerprint(doc string)` strips the
  frontmatter with `site.StripFrontmatter`, walks `site.Sections`, takes that
  section and the lines to the next heading of level two or lower or to the
  end of the file, normalises line endings to LF, trims blank edges, and
  returns the SHA-256 hex of those bytes. A chapter with no such section, or
  with two, is refused by name. The not-yet-real marker the chapter may open
  the section with is part of the section: A change to it is a change to what
  the construal states about itself, and the fingerprint says so rather than
  guessing.
- `glossary`: `.abcd/development/brief/glossary/`, the committed terms.
  `capture.GlossaryFingerprint(files map[string][]byte)` takes every tracked
  `.md` file under the directory except each `README.md` (the index is a
  render of the terms, held by `internal/core/glossary`) and the
  `_template.md` scaffold, sorts by path, normalises line endings, and returns
  the SHA-256 over the sequence of path, a NUL, content, a NUL. A term added,
  removed or edited moves it; an index regeneration does not.
- `scope`: `.abcd/development/brief/01-product/04-scope.md`, the committed
  scope, fingerprinted whole after frontmatter stripping and line-ending
  normalisation, by `capture.ScopeFingerprint(doc string)`.

Three readers share each function: The committed content through
`gitutil.RunCapped(repoRoot, issueschema.RecordReadLimit, "show", "HEAD:"+path)`
(for the glossary, `git ls-tree -r --name-only HEAD -- <dir>` then `show` per
file), the working-tree content through the guarded read the ledger already
uses, and prior states through `git log --format=%H -n 64 -- <paths>` over
all three paths together, followed by `git show <sha>:<path>` for each. A
frame state is the triple of fingerprints at one commit;
`capture.frameHistory` returns that bounded list of (commit, triple) pairs
from `HEAD` backwards, and both the whole write and the completion walk it;
sixty-four commits touching any of the three surfaces is the bound, and a
search that reaches it without finding what it looks for is refused rather
than extended. Two triples are distinct when any surface's fingerprint
differs, and the record's `changed` names which.

### The verb, and the two halves

`abcd capture reframe --occasioned-by <rdi-N|dsp-N|srp-N> --grounds "<why>" [--open]`
and `abcd capture reframe --complete <rfm-N>`, dispatching to
`capture.Reframe(ReframeRequest{RepoRoot, IssuesRoot, OccasionedBy, Grounds, Open bool, Complete string})`
returning `ReframeResult{ID, Path, OccasionedBy string; Before, After Frame; Changed []string; Half string; Commits int; Redacted int; Degraded string}`,
where `Frame` is the triple `{Construal, Glossary, Scope string}`.
`Half` is `whole`, `open` or `completed`, and both renderings print it, so
the verb says which half it wrote; `Commits` is how many commits the walk
crossed between the two hashes.

- **Whole, after the commit** (no flag): Each working-tree surface must
  fingerprint equal to `HEAD`'s, or the verb refuses naming the surface ("the
  glossary has uncommitted changes; commit the rewrite, or record the first
  half with --open"). It then walks the history to the previous distinct
  committed frame state, the first triple in which any surface differs from
  `HEAD`'s; none within the bound refuses ("the frame at HEAD matches no prior
  committed state, so there is no reframe to record"). It writes the three
  `*_before` fingerprints from that prior triple, the three `*_after` from
  `HEAD`'s, and `changed` as the surfaces that differ, in one record. The
  rewrite commit, for the predate check below, is the earliest commit in the
  walk whose triple equals `HEAD`'s.
- **First half, before the commit** (`--open`): The three `*_before`
  fingerprints are `HEAD`'s, the `*_after` fingerprints and `changed` are
  absent, and the render says "first half written; commit the rewrite, then
  `abcd capture reframe --complete rfm-N`". A second open record is refused
  while one is open, so a half can never be completed against the wrong
  rewrite. The rewrite is not yet committed, so the predate check requires the
  occasion to be committed at `HEAD`.
- **Second half** (`--complete rfm-N`): The record must be open, each
  working-tree surface must equal `HEAD`'s, and `HEAD`'s triple must differ
  from the record's `*_before` triple in at least one surface ("the frame at
  HEAD is still the state the record opened against; nothing was rewritten").
  Then the verb walks the surfaces' history back from `HEAD` until it finds
  the record's before triple, within the bound. Distinct states between the
  two are crossed and counted, not refused: A rewrite committed in two
  commits, or brought in by a merge commit rather than a squash, pairs the
  same way, so the merge strategy does not decide the outcome; `Commits`
  reports the crossing and the render says "completed across N commits". A
  walk that reaches the bound without finding the before triple refuses naming
  both triples: The surfaces' history no longer contains the state the record
  opened against, which is the one failure the intent's Mechanism names, and
  it is loud. On success the three `*_after` fingerprints and `changed` are
  set through `setScalarField` under the ledger lock, in place, atomically.

Whole and first-half writes validate and redact `grounds`, mint under the
lock, refuse an existing path, and validate the rendered record through
`validateReframeStrict` (required set, allow-list, fingerprint shape, the
`changed` vocabulary, occasion shape) before the write, so the writer refuses
what the gate would refuse.

### Resolving the occasion, and the one check on the join

The occasion is resolved through `readingitem.ResolveOccasion(issuesRoot, id, issueschema.ReframeOccasionFamilies...)`
in the `internal/core/readingitem` leaf, the one resolver the admission,
reframe and condition verbs share; it refuses an id outside the families it is
handed by shape before any path is built. An `rdi-N` resolves through
`readingitem.Locate`; a `dsp-N` through `readingitem.LocateDisposition`, which
walks `dispositions/<item>/dsp-N.md` across every item directory with
`refuseSymlinkedDir` at both levels, exactly as `readingitem.Paths` walks runs,
and refuses zero or more than one match; an `srp-N` is `surprises/srp-N.md`,
admitted only as a regular file by `Lstat`. Resolution is by presence in the
store, so a surprise written by hand today and by the sibling verb tomorrow
resolves the same way. `capture.findReadingItem` stays as a thin wrapper over
`readingitem.Locate` until every caller has moved; this spec calls the leaf.

The join from occasion to rewrite is operator-asserted: Nothing in the record
can prove that this reading item is why the construal was rewritten, and the
verb does not pretend to. It checks one thing, that the occasion predates the
rewrite, as adr-2609021016288378 states. `capture.occasionCommit` takes the resolved path and asks
`git log --diff-filter=A --format=%H -n 1 -- <path>` for the commit that added
the record; an occasion not yet committed refuses ("the occasion rdi-N is not
committed; a reframe cannot be occasioned by a record that does not yet
exist in history"). For the whole write and the completion, that commit must
be a strict ancestor of the rewrite commit (`git merge-base --is-ancestor`),
or the verb refuses naming both commits ("the occasion was committed after
the rewrite; a reframe cannot be occasioned by what came later"). For the
first half the rewrite commit does not yet exist, so the occasion must be
committed at `HEAD`, which the same check delivers with `HEAD` as the second
operand. The check is a floor, not a proof, and the capture page says so.

### Exclusion from every reading, asserted

The exclusion floor's row for `.abcd/work/issues` binds at the widening,
entailment and detection positions once the comparative channel narrows it,
and at the comparative position the floor's per-directory rows are derived
from the ledger's directory list, which now carries `ReframesDir`; so no
include row can reach a reframe at any position. Since the record is a new
family and the floor is a declaration a reader checks, `Exclusions` in
`internal/core/reading/include.go` gains one row:
`{Rule: "absent from the positive walk", Signal: "record type in a denied path", Detail: "the reframe record"}`,
beside the lapse log's. `Render()` moves, so `AssemblerVersion()` moves with it
and the charter is regenerated in the same change, which
`TestReadingsCharterCarriesTheRenderedIncludeTable` requires. The read-block
eval gains a plant: A reframe record carrying the sentinel class
`LEDGER-REFRAME` under `testdata/cold-reading/baseline/.abcd/work/issues/reframes/`,
its row in the oracle table citing this spec, and its row in the coverage
matrix, so `TestEveryAssemblerRuleHasAFalsifier` and the declared table sizes
move together. The pinned counts this spec moves: `sentinelClasses` by one,
`coverage` by one, and `excludedFamilies` by one at the comparative position
by derivation from the directory list; `excludedKeys` and `declaredGaps` do
not move.

### Versions

This spec makes a MINOR change to `AssemblerVersionCore`: An `Exclusions` row
is part of the assembly contract, and the contract's own comment says the
number moves when the contract moves. `SchemaVersion` on the manifest and
the bundle does not move: No artefact field and no kind changes. The merging
change sets the constant from the merged base and updates every pinned count
in the same diff; this spec names the class of bump and no number.

### Dispatch

`record.IDRe` becomes `^(iss|itd|spc|adr|adm|srp|rfm)-[0-9]+$`; the admission
and surprise spec makes that edit and the matching edit to the dispatch-family
list in `commands/abcd.md`, and this spec inherits both, since it lands after
that spec. `Describe` gains `describeReframe`, which reads
`.abcd/work/issues/reframes/rfm-N.md` through `readRecordHead`. `Family` is
`reframe`; `Status` is `open` when `construal_after` is absent and `complete`
otherwise; `Title` is "reframe occasioned by <id>"; `Links` carries
`occasioned_by`, the three `*_before` fingerprints and, when present, the
three `*_after` fingerprints and `changed`.
`NextMoves` on an open record is "commit the rewrite, then
`abcd capture reframe --complete rfm-N`", through a new
`verbCaptureReframe = "capture reframe"` in `RecommendedVerbPaths`, so the
anti-drift test covers it; a complete record reports none.

### The decisions, adopted

The two decisions an earlier draft flagged for the maintainer were adopted on
2026-09-02 as
[adr-2609021016288378](../../decisions/adrs/2609021016288378-a-reframe-occasioned-by-a-reading-is-a-committed-pointer-to.md),
which
[iss-2609020626118168](../../../work/issues/resolved/iss-2609020626118168-ruling-owed-whether-every-rewrite-of-the-construal-must-carr.md)
carried and is resolved by.

The first is whether every construal rewrite must carry a reframe record. The
ADR adds a pointer for the reading-occasioned case and no rule over every edit
to the frame, so the verb requires `--occasioned-by` and no lint judges a
frame edit. That is why the verb derives the `*_before` fingerprints from the
surfaces' history rather than from the last record's `*_after`: A chain would
refuse every reading-occasioned reframe that followed an unrecorded one. A
rule over every edit is adoptable later without changing this record's
shape, as the ADR says.

The second is what the frame is. The single-section form this spec first
carried, keyed on the `## Construal` section alone, was found to contradict
the design's landing table, which keeps the construal and the frame apart,
and adr-55's enumeration of the framing as three committed surfaces; it is
withdrawn. The frame is the framing as it presently stands, which adr-55
enumerates as the construal section, the committed glossary terms and the
committed scope; the record fingerprints all three surfaces before and after
and shows which changed, and a construal-only rewrite is one instance, not
the definition.

### Surfaces

`newCaptureCommand` in `internal/surface/cli/cli.go` gains the sub-verb with
flags `--occasioned-by`, `--grounds`, `--open`, `--complete`; its usage line
joins the capture page's `argument-hint`. `commands/capture.md` gains a
"Record a reframe" section stating the two halves, the three frame surfaces,
the three occasion families, the predate check and its limit, and that the
prior text of any surface never enters the record. `.abcd/development/brief/04-surfaces/06-capture.md`
gains the row `` `reframe` | — | shipped `` in its sub-verbs table, the
release surface snapshot is regenerated, and the CLI reference page is
regenerated with `go generate ./internal/surface/cli`.

### Landing order

The Iteration 2 set lands in two phases around the maintainer's readings, on
the corrections ruling of 2026-09-02. Phase A, before any reading runs: The
two-operand invocation (spc-2609021004075744), itd-194
(spc-2609021003136831), the preset windows (spc-2609020626048722), the
comparative channel (spc-2609020626039834) and the reading-occasioned origin
(spc-2609020626042168), in that order. The maintainer then runs the four
readings in the cycle's order, widening, entailment, comparative and
detection, and dispositions their outputs. Phase B, built at Step 5 after
those dispositions: The condition disposition (spc-2609020626046252), the
admission and surprise verbs (spc-2609020626040342), the reframe record
(spc-2609020626048705), the scribe verbs (spc-2609020626045177) and the
principles read object (spc-2609020626042471), in that order. No spec names a
target version number: Each names the class of bump it makes, and the merging
change sets each constant from the merged base and updates every pinned count
in the same diff.

This spec lands third in Phase B, after the admission and surprise verbs,
because it inherits their `record.IDRe` edit and resolves the surprise family
they write, and after the comparative channel in Phase A, because its
directory constant feeds the per-directory exclusion rows that channel
derives at the comparative position; it lands before the scribe so the
scribe's derived allow list carries the family from the start. A reframe a
reading occasions between the phases is hand-authored in the target format
until the record lands.

## How the Acceptance Criteria are satisfied

- **ac-1 (one record with occasion, the before and after fingerprints of
  each surface, which changed, and the ground).** The whole write after a
  committed rewrite: The `*_before` triple from the surfaces' previous distinct
  state, the `*_after` triple from `HEAD`, `changed` from their difference.
  Proved by `TestReframeRecordsACommittedRewrite`, which commits two states of
  the framing chapter in a fixture repository and compares the fingerprints
  to the three fingerprint functions over each committed text, and by
  `TestReframeRecordsAGlossaryRewrite` and `TestReframeRecordsAScopeRewrite`,
  which move one other surface each and assert `changed` names that surface
  alone.
- **ac-2 (no known prior state refuses, naming the mismatch).** The whole
  write refuses when no distinct prior frame state exists, and the completion
  refuses when the walk reaches its bound without finding the record's before
  triple, naming both triples. `TestReframeRefusesAFrameWithNoPriorState`
  and `TestCompleteRefusesARewriteItCannotPair`.
- **ac-3 (an unresolvable occasion refuses).** `readingitem.ResolveOccasion`;
  proved by `TestReframeRefusesAnUnresolvableOccasion` over an unknown
  `rdi-N`, a `dsp-N` in no item directory, an `srp-N` that is a symlink, and
  an `iss-N`.
- **ac-4 (no reframe reaches any reading; the manifest asserts it).** The
  denied segment, the derived per-directory row at comparative, the new
  exclusion row, and the planted sentinel. Proved by
  `TestReframeRecordsNeverReachTheBundle` in `internal/core/reading`, over all
  four positions, and by the read-block eval's baseline and holed runs over
  the new plant.
- **ac-5 (dispatch reports the occasion, the fingerprints and which surfaces
  changed).** `describeReframe`; proved by
  `TestDescribeReframeReportsOccasionAndFingerprints` and the root-dispatch
  JSON contract test.

## Tests

Watched fail before, pass after. `internal/core/capture`:
`TestConstrualFingerprintIsStableAcrossLineEndingsAndBlankEdges`,
`TestConstrualFingerprintRefusesAChapterWithoutTheSection`,
`TestGlossaryFingerprintIgnoresTheIndexAndMovesOnATerm`,
`TestScopeFingerprintIsStableAcrossLineEndings`,
`TestReframeRecordsACommittedRewrite`, `TestReframeRecordsAGlossaryRewrite`,
`TestReframeRecordsAScopeRewrite`,
`TestCompleteRefusesWhenNoSurfaceMoved`,
`TestReframeOpensAHalfBeforeTheCommit`,
`TestCompleteFinishesAnOpenRecord`,
`TestCompleteCrossesATwoCommitRewrite` (two commits between the halves, and
a merge commit bringing the rewrite in, both complete and report the
crossing), `TestCompleteRefusesARewriteItCannotPair`,
`TestReframeRefusesAFrameWithNoPriorState`,
`TestReframeRefusesUncommittedChangesWithoutOpen`,
`TestReframeRefusesASecondOpenRecord`,
`TestReframeHoldsTheGroundToTheFloor` (blank, and a value `ValidateText`
refuses), `TestReframeRedactsTheGround`,
`TestReframeRefusesAnUnresolvableOccasion`,
`TestReframeRefusesAnUncommittedOccasion`,
`TestReframeRefusesAnOccasionCommittedAfterTheRewrite`,
`TestReframeSerialisesOnTheLedgerLock`. `internal/core/readingitem`:
`TestLocateDispositionRefusesASymlinkedItemDir`,
`TestResolveOccasionRefusesAFamilyItIsNotHanded`. `internal/core/issueschema`:
`TestReframeKnownIsRequiredPlusAfter`, `TestLedgerDirectoriesCarryReframes`.
`internal/core/lint`: `TestRecordSchemaJudgesAReframeRecord` (blank ground,
a missing before-fingerprint, an unknown key, a mis-shaped fingerprint, a
`changed` value outside the three names, an occasion naming nothing). `internal/core/reading`: `TestReframeRecordsNeverReachTheBundle`,
`TestExclusionFloorNamesTheReframeRecord`. `internal/core/record`:
`TestDescribeReframeReportsOccasionAndFingerprints`, `TestRecommendedVerbPathsClosed`
(existing, extended). `internal/surface/cli`: `TestCaptureReframeSurface`
(both halves' renders name the half), `TestNextMoveVerbsResolveInLiveTree`
(existing). `evals`: The read-block lane over the new plant, run explicitly.

## Out of scope

- The prior text of any surface, and any path by which it could reach the
  record.
- Reframes no reading occasioned, and any lint over frame edits; adr-2609021016288378 adds
  a pointer for the reading-occasioned case and no rule over every edit.
- A proof that the occasion caused the rewrite. The record carries the
  operator's assertion and the one check that the occasion came first.
- The fourth audit verdict; a reframe changes what is built next and puts no
  delivered promise in question.
- A frame that lives at other paths than the three named; the paths and the
  heading are constants, per the intent's first scope condition.
- Writing the surprise family, and the `record.IDRe` and `commands/abcd.md`
  edits; the admission and surprise spec owns those, and this one inherits
  them and resolves what that verb wrote.
