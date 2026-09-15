---
id: itd-194
slug: the-reading-include-table-admits-only-what-the-exclusion-flo
spec_id: spc-2609021003136831
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: [itd-183]
severity: major
impact: fix
origin: researcher-authored
production_mode: hand-written
---

# The reading include table admits only what the exclusion floor can read

Typed links: `builds_on` [itd-183](../shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md) (the assembler, the include table and the exclusion floor); `refines` [itd-183](../shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md) (the floor refuses what it cannot resolve and the manifest asserts per item) and [itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md) (a preset opts source and tests in, never the table by default); related [adr-56](../../decisions/adrs/0056-an-exclusion-control-asserts-only-what-it-can-prove.md) (an exclusion control asserts only what it can prove).

## Press Release

> **abcd ships with an assembler whose manifest says exactly what was
> examined.** The cold-reading include table and the exclusion floor now
> describe one set, item by item: a markdown document the floor cannot resolve
> is refused at admission and named, never passed through unscanned, and a
> source or test file, which the floor does not parse, is admitted only when a
> committed preset entry opts it in, travels whole, and is marked `unscanned` in the
> manifest. The manifest's exclusion assertion is made per item and only for
> the items the floor parsed, so a scan that ran and a scan that never ran no
> longer produce the same artefact. The widening position stops receiving the
> shipped intents, which neither design document lists in its object.
>
> The floor previously recognised a construct by the shape of the container it
> sat in. A record-shaped document inside a Go fixture was not markdown, so it
> was not scanned, so it travelled intact while the manifest asserted its
> headings had been refused. Nine rounds of adding patterns closed nine
> spellings of that mistake. This closes the tenth by refusing what cannot be
> resolved and disclosing what was not examined, instead of enumerating what
> may walk down the silent path.

## Why This Matters

An exclusion floor is a control, and a manifest is its attestation. When the
two disagree, the manifest is the side that is wrong, and a reader holding the
bundle and the manifest has no way to know it. That is the defect the itd-183
audit found live on this repository's own corpus, not in a crafted document:
the assembler admitted `internal/core/site/fixture_test.go` at item `itm-0736`
carrying three literal `## Audit Notes` sections with their acceptance
rollups, while the same run's manifest asserted `Audit Notes` was refused
(iss-2608301450065320).

Ten records stand against the floor. They collapse into one mechanism and
three consequences of it. The mechanism is container-shape gating: the scan
is switched off for a whole region or file because the container does not
match an expected shape, and the manifest goes on asserting exclusion anyway.
The file extension is one container (iss-2608301450065320). A fenced region
wrongly extended over the frontmatter is another, and it switches off the very
refusal that exists to catch keys the field reader cannot see
(iss-2608301350533102, the set's only critical). A frontmatter block that
never opens because line 0 is not a delimiter is a third
(iss-2608301237456350).

The rest are the same error at smaller grain. A key that is nested rather than
line-anchored is invisible to patterns anchored to the line
(iss-2608301237450573, iss-2608301251398360, which states outright that the
two want one fix and not two). A character the file's other readers treat as
whitespace, but the scan's character class does not, mis-parses a heading
opener and admits the heading under it (iss-2608301350534164,
iss-2608301421380392).

Three of those are verified leaking on this round and on its parent. A fourth
is live on the corpus at HEAD. The records are unanimous about what that
means: `a spelling arms race needs a design, not more patterns`
(iss-2608301251398360), and `fixing them one at a time is how this floor has
spent nine rounds`.

The maintainer ruled on 2026-09-02, checked against the design framework and
the readings companion, which both name code, tests, documentation and
configuration as the shipped tree a reading may see: source and tests reach
a reading only where a committed preset entry names them, an item so named
travels whole and marked unscanned, and the floor refuses markdown it cannot
resolve
and marks what it does not parse, so the manifest asserts the exclusion only
for the items it parsed (brief invariant 16). The strict alternative, removing
the test kind from the table, was declined because it contradicts both
documents; widening the floor to scan every type stays declined (ruled
2026-08-30). The same interview settled iss-2609012259587904 from the
documents: the framework's widening object and the companion's section 5.2
both list the widening object without the shipped intents, so the shipped
intent projection row's positions exclude widening, and that change lands
here with the include-table work.

## Mechanism

We expect refusal-at-admission for markdown, and an `unscanned` mark for what
the floor does not parse, to close this class where nine rounds of added
patterns did not, because every leak in the set shares one shape rather than
ten: the floor declined to examine something, and nothing downstream knew the
examination had not run. Enumerating what may pass leaves the silent-admission
path in place and re-opens on the next unenumerated spelling. Refusing a
markdown document the floor cannot resolve removes the path for the material
the floor claims to cover, and marking every unparsed item makes the
manifest's claim a fact about each item rather than a claim about the run.

This is falsifiable in two moves. Find a leak in a document the floor parsed
and the manifest marked as parsed: such a leak would show the defect was in
the floor's patterns after all, and that refusal bought nothing. Or find an
item the floor did not parse arriving in a bundle without the mark: that
would show the manifest still asserting more than the examination behind it.

## Scope Conditions

- The corpus is this repository's committed record. A reading assembled over <!-- cond: cond-2609021003132586 -->
  an external or mixed corpus is a different population and is not claimed
  here.
- The floor's parseable set is markdown whose frontmatter and headings it can <!-- cond: cond-2609021003134464 -->
  resolve. Every other admitted kind is disclosed as unscanned rather than
  examined; widening the parseable set is a separate change, and this intent's
  claim does not survive it unexamined.
- The committed detection and widening entries name the `source` and `test` kinds, <!-- cond: cond-2609021003130127 -->
  on the object-set ruling of 2026-09-02, and every such item arrives whole
  and marked `unscanned`; the entailment entry names neither. An entry that
  names either kind by commit accepts the unscanned items it selects,
  disclosed by the mark.
- No reading runs before this intent ships; every criterion below is judged <!-- cond: cond-2609021003139989 -->
  against the assembler's output and its manifest.

## What's In Scope

- The `source` and `test` kinds stay in the include table, so the object stays
  as the design documents state it, which for the detection and widening
  positions closes with the shipped tree: The committed detection and widening
  entries name both kinds, on the object-set ruling of 2026-09-02, and the
  entailment entry names neither. An entry that names one gets each item
  whole, and the manifest marks each such item `unscanned`.
- The floor refuses a markdown document it cannot resolve, naming the document
  and the shape, and never admits one unscanned: a fence delimiter inside the
  frontmatter block; a delimited block that does not open at line 0 but would
  to a reader of the bundle, because only blank lines, whitespace or an HTML
  comment precede it; a compact mapping nested in a block sequence and an
  explicit key in a flow mapping, which the line-anchored key reader cannot
  see; an attribute whose value opens on the line after its equals sign,
  which declines the markup mask; and a raw heading opener that reaches the
  end of the document with no bound, which is how a CRLF document's blank line
  was read over. Each extends the answer
  `unresolvableFrontmatterShape` already gives for one class to the whole
  set.
- The manifest's exclusion assertion is made per item, only for the items the
  floor parsed: an item carries a closed mark saying whether the key and
  heading signals were examined over it, and the assertion rests on that
  mark rather than on the run.
- The narrowing the include table performs is stated in the table itself:
  each row declares whether the floor parses what it admits, the rendered
  charter carries the declaration, and admission and examination are one
  declaration rather than a row on one side and an extension test on the
  other (iss-2608301450065320 names this as the undeclared scope decision
  underneath the whole class).
- The shipped intent projection row's positions exclude widening, the
  exclusion floor asserts it at that position, and the widening definition's
  admitted sources are regenerated (iss-2609012259587904, resolved by this
  intent).
- The include table admits the brief's meta, product, constraints, surfaces,
  internals and delivery chapters as brief sections, one row per chapter, at
  every position that reads the brief, because the design documents name
  "brief current text" as a reading's object and the table admits two chapters
  of it today. The rows are named individually because rule 1 forbids naming
  `brief/` whole: The directory contains the glossary, which keeps its own row.
  The evidence chapter stays excluded as verdict material, on the ground the
  Audit Notes exclusion rests on, with that ground stated in the table's rule
  text, and the manifest asserts the exclusion (iss-2609021153264023, resolved
  by this intent on the corrections ruling of 2026-09-02).

## What's Out of Scope

Four findings in the evidence set are inherited by this intent and fixed by
none of it. They are recorded here so a later reader does not mistake their
survival for an oversight:

- The quadratic title render on an unbounded heading opener, and the test
  whose name over-claims the linearity it does not cover
  (iss-2608301421382564). A cost finding about the scan's algorithm, not
  about what it excludes; the refusal of an unbounded opener removes that
  shape's admission and claims nothing about the scan's cost, and the test's
  name and its coverage stay with the issue.
- The two hand-written definitions of what opens a tag
  (iss-2608301251394412). Tech debt under the one-canonical-primitive rule;
  the record asks only that this intent inherit it.
- The doc comment describing fence behaviour the code does not have
  (iss-2608301237450573, second half).
- The escaped-key refusal whose message asserts two things that need not be
  true (iss-2608301421381157). The refusal is correct; only its stated reason
  is wrong.

Widening the floor's parseable set is also out of scope. This intent makes
admission and comprehension agree by disclosing at admission; moving
comprehension instead is the alternative the itd-183 audit named and the
maintainer did not choose (ruled 2026-08-30, confirmed 2026-09-02). Removing
the `test` kind from the table is out of scope for the opposite reason: it was
declined because it contradicts both design documents. The presets themselves
are not recalibrated here; that is the preset-windows intent's work, and
which committed entries name the two tree kinds is that intent's to record:
The detection and widening entries name both, the entailment entry neither.

## Acceptance Criteria

- **Given** a markdown document the include table admits whose frontmatter or
  headings the exclusion floor cannot resolve, **when** the assembler runs,
  **then** the document is refused, the refusal names the document and the
  shape, and no part of it reaches the bundle.
- **Given** a document whose frontmatter the floor cannot resolve, including a
  fence delimiter inside the frontmatter block, a delimited block preceded
  only by blank lines, whitespace or an HTML comment, a compact mapping nested
  in a block sequence, and an explicit key in a flow mapping, **when** the
  floor scans it, **then** the document is refused rather than admitted, and
  the same holds for an attribute whose value opens on the line after its
  equals sign and for a raw heading opener that reaches the end of the
  document with no bound, once a CRLF blank line counts as one.
- **Given** a committed preset entry that opts the `source` or `test` kind in,
  **when** the assembler runs, **then** each such item travels whole and its
  manifest entry is marked `unscanned`, and no item the floor parsed carries
  that mark.
- **Given** a bundle and its manifest, **when** the manifest asserts a
  construct was excluded, **then** the assertion is made per item and only for
  the items marked as parsed, so it rests on a scan that ran rather than on
  one that may have declined, and a manifest whose item carries no mark is
  refused by the decoder.
- **Given** this repository's corpus at HEAD and the committed presets,
  **when** the assembler runs under the committed entry at any assembling
  position, **then** the Go fixture carrying three `## Audit Notes` sections
  is absent from the bundle, and **when** an entry opts tests in, **then** the
  item recorded at `itm-0736` arrives marked `unscanned`, so the leak does not
  reproduce as a leak.
- **Given** the include table, **when** it is inspected or rendered into the
  charter, **then** each row states whether the floor parses what it admits,
  and a reader can see which documents the floor is claimed to cover without
  reading the floor.
- **Given** the widening position, **when** the assembler runs, **then** no
  shipped intent reaches the bundle, the manifest asserts the exclusion at
  that position, and the entailment and detection positions still receive the
  shipped intents projected as before.
- **Given** the include table, **when** the assembler runs at any position
  that reads the brief, **then** a document under each of the brief's meta,
  product, constraints, surfaces, internals and delivery chapters is admitted
  as a brief section, a document under the evidence chapter is refused, and
  the manifest asserts that exclusion with the verdict-material ground stated
  in the table's rule text.

## Open Questions

None. The two questions the draft carried are answered by the ruling. The
narrowing costs the committed entries nothing they do not disclose: The
detection and widening entries name source and tests, on the object-set ruling
of the same day, and every such item arrives marked `unscanned`, while the
entailment entry names neither; the fixture leak at `itm-0736` is absent from
every committed entry, whose paths do not reach `internal/core/site`, and
disclosed as unscanned wherever an entry opts that path's tests in. A refusal
is per document for markdown the floor cannot resolve, which keeps a reading
possible over the rest of the corpus and names the one document that stopped
it; an opted-in non-markdown item is never refused, because refusing it would
contradict the object the design documents state, and is marked unscanned
instead.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-11891aee84a6 -->
Fidelity review — receipt rcp-11891aee84a6 (verifier intent-auditor claude-opus-5[1m]).

Provenance: intent-auditor@claude-opus-5[1m] · rubric_hash sha256:44f19418b0b8d56d7b17cbecbd5acd5d2cbed5abc4347428680ce46647f381d5 · prompt_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5
Input attestations: diff:c0baaff4~1..c1be0000@sha256:0913f8672a2568fe0f7970a3f03189bc2df67f3d953cf2c18c349009f18bdf89;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: verifyRedaction raises each unresolvable shape as a refusal naming the path and the shape and collect returns nil on the error, so the assembly stops and no item of the document reaches the bundle; the test asserts both the naming and the empty result
  evidence: internal/core/reading/project.go:359 — "if line, shape, ok := displacedFrontmatter(lines); ok {"
  evidence: internal/core/reading/assemble.go:874 — "doc, err = redactExcluded(rel, doc, exclusions)"
  evidence: internal/core/reading/project_test.go:341 — "if len(res.Bundle.Items) != 0 || len(res.Manifest.Items) != 0 {"
- ac-2 — MET: all six shapes plus the CRLF blank-line bound are implemented and each has a unit test that I watched pass, and six refusal plants run the same shapes through the eval at every assembling position
  evidence: internal/core/reading/project.go:1031 — "return i + 1, "a fence delimiter inside the frontmatter block", true"
  evidence: internal/core/reading/project.go:1134 — "a frontmatter block displaced from line 0 by %d line(s)"
  evidence: internal/core/reading/project.go:1037 — "return i + 1, "a mapping nested in a block sequence", true"
  evidence: internal/core/reading/project.go:1039 — "return i + 1, "an explicit key in a flow mapping", true"
  evidence: internal/core/reading/project.go:835 — "shape = "an attribute value that opens on the line after its equals sign""
  evidence: internal/core/reading/project.go:200 — "rawHeadingBoundRe = regexp.MustCompile(`(?is)</([a-z] [a-z0-9-]*)\s*>|<h[1-6] (?:\s[^>]*)?/?>|\n[ \t\r]*\n`)"
  evidence: internal/core/reading/project_test.go:288 — "func TestAnUnboundedRawHeadingRefusesAndACRLFBlankLineBounds(t *testing.T) {"
- ac-3 — MET: collect calls redactExcluded only for a ScanParsed row and passes an unscanned row's document through untouched, stamping the row's Scan onto each candidate; the two tests assert byte identity plus the unscanned mark, and that no .md item is marked unscanned
  evidence: internal/core/reading/assemble.go:873 — "if row.Scan == ScanParsed {"
  evidence: internal/core/reading/size_test.go:637 — "func TestOptedInSourceAndTestsTravelWholeMarkedUnscanned(t *testing.T) {"
  evidence: internal/core/reading/size_test.go:690 — "func TestNoParsedItemCarriesTheUnscannedMark(t *testing.T) {"
- ac-4 — MET_WITH_CONCERNS: DecodeManifest refuses an item whose scan mark is absent or outside the closed vocabulary, and each item now carries the mark; the concern is that the emitted exclusions array is unchanged in shape and carries no per-item scoping of its own, so the per-item narrowing lives in include.go's doc comment and in the eval oracle rather than on the artefact's face
  evidence: internal/core/reading/manifest.go:263 — "if it.Scan == "" {"
  evidence: internal/core/reading/manifest.go:141 — "Scan Scan `json:"scan"`"
  evidence: internal/core/reading/include.go:469 — "asserted for the items the manifest marks `parsed` and for no other"
  evidence: internal/core/reading/include.go:545 — "Positions []Position `json:"-"`"
  evidence: evals/coldreading_oracle_test.go:800 — "func checkFieldAbsence(a assembled) []violation {"
- ac-5 — MET_WITH_CONCERNS: both halves are demonstrated over a detached clone of this repository's HEAD and the eval lane passes, but three caveats are owed: the assembler could not run over this corpus at the intent's own commit at all, the proving test landed one commit later, and the item is identified by path rather than by the run-scoped key itm-0736 the criterion names
  evidence: evals/coldreading_test.go:768 — "func TestTheFixtureLeakIsAbsentUnderEveryCommittedPreset(t *testing.T) {"
  evidence: evals/coldreading_test.go:811 — "if it.Scan != "unscanned" {"
  evidence: internal/core/reading/scope_test.go:1058 — "func TestOnlyTheTreePositionsNameSourceOrTest(t *testing.T) {"
  evidence: .abcd/development/brief/glossary/core/construal.md:11 — "the attribution comment repaired below the frontmatter in 255543c1; assemble refused every position before it"
- ac-6 — MET: Row gains Scan, every row declares one, Render emits a Floor column between Kind and Admitting rule, and the regenerated charter carries it, so a reader sees the claimed coverage without opening the floor
  evidence: internal/core/reading/include.go:230 — "Scan Scan"
  evidence: internal/core/reading/include.go:580 — "| Positions | Source | Matches | Suffixes | Fields | Store | Bucket | Kind | Floor | Admitting rule |"
  evidence: .abcd/development/readings/README.md:120 — "| Positions | Source | Matches | Suffixes | Fields | Store | Bucket | Kind | Floor | Admitting rule |"
  evidence: internal/core/reading/include_test.go:429 — "func TestParsedRowsAdmitOnlyMarkdown(t *testing.T) {"
- ac-7 — MET: the shipped intent row's Positions drop widening, a widening-scoped exclusion row asserts the withdrawal into the manifest, the widening definition loses its shipped line, and both the unit test and the eval assert the family is absent at widening and present at entailment and detection
  evidence: internal/core/reading/include.go:363 — "Positions: []Position{PositionEntailment, PositionComparative, PositionDetection},"
  evidence: internal/core/reading/include.go:527 — "Rule: "the widening object as the design documents state it","
  evidence: evals/coldreading_test.go:657 — "func TestWideningNeverSeesTheShippedIntents(t *testing.T) {"
  evidence: internal/core/reading/include_test.go:195 — "func TestWideningExcludesDraftsAndPlannedEntailmentIncludesThem(t *testing.T) {"
- ac-8 — MET: all six brief chapters are admitted as brief sections at every position the brief rows admit, the meta row reaches 00-meta.md alone, and the evidence chapter's exclusion row now states the verdict-material ground in the table's own rule text
  evidence: internal/core/reading/include.go:296 — "Source: ".abcd/development/brief/04-surfaces","
  evidence: internal/core/reading/include.go:337 — "Match: []string{"00-meta.md"},"
  evidence: internal/core/reading/include.go:487 — "Rule: "verdict material: a prior verdict is revision history, the ground the Audit Notes exclusion rests on","
  evidence: internal/core/reading/include_test.go:492 — "func TestBriefChaptersAreAdmittedAsBriefSections(t *testing.T) {"
  evidence: internal/core/reading/include_test.go:549 — "func TestTheEvidenceChapterIsExcludedAsVerdictMaterial(t *testing.T) {"

Gap audit:
- honoured:
  - a markdown document the floor cannot resolve is refused at admission and named, never passed through unscanned
    evidence: internal/core/reading/project.go:364 — "if line, shape, ok := unresolvableFrontmatterShape(lines, fenced); ok {"
    evidence: internal/core/reading/project_test.go:320 — "func TestAnUnresolvableDocumentIsRefusedByName(t *testing.T) {"
  - admission and examination are one declaration: the extension test is gone and the floor runs over the row's Scan
    evidence: internal/core/reading/project.go:50 — "There is no file-extension test here any more, and its absence is the point."
    evidence: internal/core/reading/assemble.go:873 — "if row.Scan == ScanParsed {"
  - a source or test file is admitted only where a committed preset entry opts it in, travels whole, and is marked unscanned in the manifest
    evidence: internal/core/reading/include.go:419 — "Scan: ScanUnscanned,"
    evidence: .abcd/config/reading-presets.json:92 — ""kinds": ["brief-section", "glossary-term", "discipline", "spec", "intent-projection", "doc", "config", "source", "test"],"
    evidence: internal/core/reading/size_test.go:637 — "func TestOptedInSourceAndTestsTravelWholeMarkedUnscanned(t *testing.T) {"
  - a scan that ran and a scan that never ran no longer produce byte-identical attestations: every manifest item carries a closed scan mark and the decoder refuses one without it
    evidence: internal/core/reading/manifest.go:141 — "Scan Scan `json:"scan"`"
    evidence: internal/core/reading/manifest.go:268 — "if !knownScans[it.Scan] {"
  - the widening position stops receiving the shipped intents, which neither design document lists in its object, and the floor asserts the withdrawal there
    evidence: internal/core/reading/include.go:524 — "Detail: ".abcd/development/intents/shipped","
    evidence: agents/cold-reading-widening.md:8 — "prompt_version moved PATCH and the shipped line removed from the admitted-sources list"
  - the include table admits the brief's six chapters as brief sections and keeps the evidence chapter out as verdict material, with the ground stated in the table's rule text
    evidence: internal/core/reading/include.go:330 — "this one contains the glossary — which keeps its own row above."
    evidence: internal/core/reading/include.go:487 — "Rule: "verdict material: a prior verdict is revision history, the ground the Audit Notes exclusion rests on","
  - the narrowing the include table performs is stated in the table itself and carried into the rendered charter and the plugin page
    evidence: .abcd/development/readings/README.md:134 — "| `test` | `unscanned` | Admitted where a committed preset entry names this kind, and never examined"
    evidence: commands/reading.md:120 — "every manifest item carries a `scan` mark saying `parsed` or `unscanned`."
  - the size report counts what was not examined and the CLI names it above zero
    evidence: internal/core/reading/assemble.go:116 — "Unscanned int `json:"unscanned"`"
    evidence: internal/surface/cli/reading.go:391 — "unscanned: %d item(s) travel whole, not examined by the exclusion floor"
- diverged:
  - the spec's assertion that none of the six shapes appears in a committed record the include table admits, checked over the record at the named tree
    evidence: .abcd/development/specs/closed/spc-2609021003136831-the-reading-include-table-admits-only-what-the-exclusion-flo.md:211 — "None of the six shapes appears in a committed record the include table admits"
    evidence: .abcd/development/brief/glossary/core/construal.md:11 — "21 glossary records carried the displaced-block shape; assemble refused this corpus at every position until 255543c1 moved the attribution comment below the frontmatter (iss-2609021457186209)"
  - ac-5's second half, proved in the intent's own change
    evidence: evals/coldreading_test.go:768 — "TestTheFixtureLeakIsAbsentUnderEveryCommittedPreset landed in c1be0000, one commit after c0baaff4, because the committed entries carried no paths until the preset-windows intent"
  - the eval's carriers move +2 and the spec names no move of materialClasses
    evidence: evals/coldreading_oracle_test.go:244 — "{"carriers", len(carriers), 17},"
    evidence: evals/coldreading_oracle_test.go:245 — "{"materialClasses", len(materialClasses), 10},"
  - only the widening definition's prompt_version moves PATCH
    evidence: agents/cold-reading-detection.md:7 — "prompt_version 0.1.1 to 0.1.2; all four definitions gained the four brief-chapter sources and all four moved PATCH"
    evidence: internal/core/reading/definitions_test.go:252 — "TestDefinitionHoldsItsFiveParts holds every definition's admitted sources against the table, so the four moved together"
  - each refusal plant's Names name the path, the shape's wording and the excluded thing it hides
    evidence: evals/coldreading_fixture_test.go:378 — "the plants' Names carry the path and the shape wording only; the hidden key moved to Why, because naming it would mean resolving it"
  - the manifest's exclusion assertion is made per item
    evidence: internal/core/reading/include.go:545 — "Positions []Position `json:"-"`"
    evidence: internal/core/reading/include.go:467 — "What each entry asserts, and over WHICH items, is fixed here rather than left to a reader"
- missing:
  - the measurement adr-56 assigns to itd-194: how much corpus the narrowing costs, taken before a reading runs
    evidence: .abcd/development/decisions/adrs/0056-an-exclusion-control-asserts-only-what-it-can-prove.md:101 — "The corpus a reading sees gets smaller, by an amount nobody has measured."
    evidence: .abcd/development/intents/shipped/itd-194-the-reading-include-table-admits-only-what-the-exclusion-flo.md:249 — "None. The two questions the draft carried are answered by the ruling."

Scope-condition dispositions:
- cond-2609021003132586 — narrowed: the corpus was this repository's committed record, but the record had to move for the claim to hold: 21 glossary records carried the displaced-block shape and the assembler refused every assembling position over the corpus as it stood at the intent's own commit
  narrowing: holds over this repository's committed record as repaired by 255543c1, not over the corpus as it stood when the intent's own commit landed
  evidence: .abcd/development/brief/glossary/core/construal.md:11 — "the mattpocock/skills attribution comment moved below the frontmatter block in 255543c1"
  evidence: internal/core/reading/project.go:1134 — "a frontmatter block displaced from line 0 by %d line(s)"
- cond-2609021003134464 — survived: every parsed row admits markdown and nothing else, every other admitted kind declares ScanUnscanned and is disclosed by the mark, and the parseable set was not widened
  evidence: internal/core/reading/include_test.go:429 — "func TestParsedRowsAdmitOnlyMarkdown(t *testing.T) {"
  evidence: internal/core/reading/include.go:450 — "Scan: ScanUnscanned,"
- cond-2609021003130127 — survived: the committed detection and widening entries name source and test, the entailment entry names neither, and the test asserts every such item in each entry's manifest carries the unscanned mark
  evidence: .abcd/config/reading-presets.json:28 — ""kinds": ["brief-section", "glossary-term", "discipline", "spec", "doc", "config", "source", "test"],"
  evidence: .abcd/config/reading-presets.json:60 — ""kinds": ["brief-section", "glossary-term", "discipline", "spec", "intent-projection"],"
  evidence: internal/core/reading/scope_test.go:1058 — "func TestOnlyTheTreePositionsNameSourceOrTest(t *testing.T) {"
- cond-2609021003139989 — survived: no reading ran in the delivered range: the readings family holds only its charter, and every criterion above was judged against the assembler's output and its manifest
  evidence: .abcd/development/readings/README.md:120 — "the readings family carries the rendered include table and no reading record"
## Grounds

- pursued: we expect refusing markdown the floor cannot resolve, and marking every item the floor did not parse as unscanned in the manifest, to close the container-shape class that nine rounds of added patterns did not, because every leak shared one shape, a scan that declined with nothing downstream told; a leak in a document the manifest marks as parsed, or an unparsed item arriving without the mark, would show it wrong
