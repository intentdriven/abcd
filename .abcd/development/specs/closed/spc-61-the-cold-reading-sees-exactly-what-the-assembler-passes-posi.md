---
id: spc-61
slug: the-cold-reading-sees-exactly-what-the-assembler-passes-posi
intent: itd-183
---

# The cold-reading input assembler: positive inclusion at field granularity, a pathless bundle, and a hashed per-run manifest

## Bundle

itd-183, [itd-184](../../intents/shipped/itd-184-four-cold-reading-definitions-one-blindness-core-each-positi.md)
and [itd-185](../../intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md)
are one design under one bundle kind, and the ceremony cannot give them one
spec: a spec's `intent:` is a single id, captured as iss-2608300108376943, so
the three are written as one cross-linked design record.

| Spec | Component it owns |
| --- | --- |
| spc-61 (this record) | The input assembler, the include table, the pathless bundle, the manifest, and the bundle's shared decisions |
| [spc-62](spc-62-four-cold-reading-definitions-one-blindness-core-each-positi.md) | The four reading definitions under `agents/` and the blindness-core byte-identity test |
| [spc-63](spc-63-one-ingest-verb-validates-every-cold-reading-output-includin.md) | The output contract, the supply-regime gate, and the ingest sub-verb |

**Shared decisions are stated once, here**, under "The package, the verb tree
and the artefact layout". spc-62 and spc-63 link to that section rather than
restating it.

## Summary

spc-61 delivers itd-183's assembler: blindness stops being a promise the
reader makes and becomes a property of the input it is handed. A new
cobra-free package, `internal/core/reading`, walks the repository under a
**positive include table at field granularity**, projects the admitted fields
into a **bundle carrying no repository path**, and emits a **hashed manifest**
naming every passed item by path and field, so a third party can re-run the
assembly and diff the result. What the include table does not name is absent,
including a record family invented after the table was written, and including
the instrument's own output.

The policy is settled and not this spec's to reopen: brief invariants
[14 and 15](../../brief/02-constraints/03-invariants.md),
[adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md),
and the 2026-08-28 rulings (3), (5), (8), (9), (12) and (18). This spec
settles the mechanics those rulings left open: the package and verb names, the
invocation grammar, the run-identifier form, the manifest and bundle schemas,
and how the record scan is reused.

## Scope

**In.** The `internal/core/reading` package (assembly, projection, manifest);
the include table as a single source with a rendering test; the `abcd reading`
verb tree in `internal/surface/cli` and its plugin markdown counterpart; the
run-identifier mint; the `.abcd/development/readings/<run-id>/` family and its
charter.

**Out.** The definitions (spc-62); the output contract and ingest validation
(spc-63); the read-block and amnesia evals
([itd-186](../../intents/shipped/itd-186-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md),
[itd-187](../../intents/shipped/itd-187-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md));
the reading-record and disposition schemas
([itd-180](../../intents/shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md),
spc-58); admission
([itd-189](../../intents/shipped/itd-189-what-the-widening-reading-proposes-is-admitted-or-declined-o.md)).

## Approach

### The package, the verb tree and the artefact layout (shared bundle decisions)

- **Package: `internal/core/reading`.** Cobra-free and stdout-free like every
  sibling under `internal/core/`, per
  [adr-23](../../decisions/adrs/0023-transport-agnostic-core.md). It holds the
  include table, the projection, the manifest, the definition reader, and
  (spc-63) the output contract: one package, because the read-block and the
  contract are two halves of one instrument and a split would let them drift.
- **Verb tree: `abcd reading`.** Bare `abcd reading` is a read-only status
  render (the include table's version, the definitions found, any orphaned
  stage). `abcd reading assemble --position <position> --target <commit>`
  produces a run; `abcd reading ingest --output-json <path>` is spc-63's.
  Every sub-verb takes `--json`, and the plugin surface `/abcd:reading` is a
  thin front door onto the same core calls.
- **Run identifier: `rdg-<yymmddHHMMSS><rrrr>`**, minted by
  `recordid.Minter.Mint("rdg")` under
  [adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md):
  the family tag satisfies the mint's `^[a-z]+$` bound, and the mint reads no
  maximum, so two checkouts cannot converge on one run id.
- **Artefact layout.** `assemble` parks the run in the local tier at
  `.abcd/.work.local/scratch/reading-runs/<run-id>/`, holding `bundle.json`
  and `manifest.json`; `ingest` promotes `manifest.json` and writes `run.json`
  to the durable tier at `.abcd/development/readings/<run-id>/` (ruling (9)).
  `--dry-run` renders and writes nothing, on the render-without-writing idiom
  `disembark plan` already carries.

### The invocation interface: position and target, nothing else

`assemble` takes exactly two operands and no free-text argument anywhere
(ruling (5)):

- `--position` accepts one of four closed tokens: `widening`, `entailment`,
  `comparative`, `detection`. An unknown token is refused by name.
- `--target` accepts `HEAD` or a hexadecimal commit sha of 7 to 40 digits, and
  nothing else. Branch names and tags are refused as mutable and prose-shaped,
  and the manifest's re-runnability rests on a reference that cannot move.

Assembly reads the working tree and refuses unless `HEAD` resolves to the
target sha **and** no included path is dirty: a dirty tree cannot be described
by a commit reference, so the manifest would promise a re-run it could not
deliver.

### The include table: one source, rendered and tested

`include.go` holds the table as Go data on the single-source-with-test idiom
of [`internal/core/lifeboat/mapping.go`](../../../../internal/core/lifeboat/mapping.go),
which is rendered into the brief's `00-meta.md` under a test asserting the two
agree. Here the render target is the new family's charter,
`.abcd/development/readings/README.md`. A row is
`{position, source, fields, admitting rule}`, and every framing row cites
adr-55 as the rule admitting it, per that ADR's own consequence.

The table is per-position over a shared floor (ruling (8)). The floor admits
the brief's `01-product/` and `02-constraints/` chapters, the framing
section's construal statement included and `03-evidence/` deliberately
excluded as deliberation; `brief/glossary/`; `intents/disciplines/`; `specs/`;
and the shipped tree. Shipped intents are admitted **by field**: press
release, acceptance criteria, scope conditions, mechanism claim and `spec_id`,
and nothing else, so the Audit Notes a shipped intent also carries cannot
travel.

**The widening and entailment asymmetry is stated in the table, not
remembered**: the `widening` rows exclude `intents/drafts/` and
`intents/planned/`, because a reading must not see the candidate set it is
asked to widen; the `entailment` rows include both, because articulation
precedes selection. itd-183 and itd-184 disagreed about the `comparative`
position's object; **ruling (8) settles it as the widening reading's
pre-admission output** for the same cycle, read against the selection-criteria
discipline
[itd-191](../../intents/disciplines/itd-191-the-selection-criteria-are-a-declared-recorded-discipline-a.md),
with a prior run's stored output staying read-blocked.

### Default deny, and the two assembler rules

The stance is `internal/core/launch`'s: a structural deny no include can
promote. `DenyNamespaces` there denies `.abcd` on every path component,
case-insensitively, and the same shape is used here. Two rules bind the table
itself (ruling (18)):

1. **No include may name a directory that contains a record family.** "The
   shipped tree" is scoped to source, tests, documentation and configuration,
   with `.abcd/` denied wholesale and `agents/`, `evals/` and
   `internal/core/reading/` denied besides; record paths a reading
   legitimately needs are named individually. A family added later, the
   readings family itself included, is excluded by construction.
2. **A reading's object excludes the material whose state that reading exists
   to change.** The drafts asymmetry and the Audit-Notes exclusion are its two
   instances, which is what makes the table derivable rather than remembered.

Each exclusion carries the signal that detects it: `origin` and the
production-mode key by frontmatter key, Audit Notes by heading, directories by
never appearing in the positive walk, and dispositions and grounds by living
in denied paths. The manifest asserts the exclusions so a reader can check
them rather than trust them.

### The record scan, reused

Record enumeration comes from `internal/core/lint`'s exported
[`LoadRecordGraph`](../../../../internal/core/lint/graph.go), the one parser of
the record's shape in this binary (`internal/core/site` is the existing
consumer, and a second parser is forbidden). It supplies each record's id,
store, lifecycle bucket and path, which is what the table selects on, and no
body sections, so **field projection is a heading-scoped extractor in this
package** reading the file the graph named: a projection over an enumerated
record, not a second reading of the record's shape.

### The bundle carries no repository path

Invariant 15 requires the assembled input to be the reading's entire working
set with no repository path in its context. The Go binary cannot stop a host
from also handing the agent repository tools; what it can do, and does, is
make the bundle itself pathless. Each bundle item is `{item_key, kind, text}`,
where `item_key` is `itm-<4-digit ordinal>` in lexicographic walk order and
`kind` is a closed vocabulary of material classes (`brief-section`,
`glossary-term`, `intent-projection`, `discipline`, `spec`, `source`, `doc`,
`config`) naming a class and never a location. The manifest, not the bundle,
maps `item_key` to path and field: the auditor resolves an item to its source,
and the reading cannot.

The remaining half of the isolation, that the dispatching host grants the
subagent no repository access, is stated in each definition (spc-62) and in
the plugin surface section, disclosed as a host obligation and never claimed
as an enforcement this binary performs.

### The manifest

`manifest.json` is JSON with `DisallowUnknownFields` on the read side, and
carries no timestamp of any kind:

```
_type, schema_version, run_id, position, target_commit, assembler_version,
items[]      = {item_key, path, field, sha256}
exclusions[] = {rule, signal, detail}
```

Its own content hash over canonical bytes is the reference spc-63 checks. It
enumerates the passed, cold items only, hashed, never their content, so
committing it needs no redaction. Both halves of its warmth are recorded: the
content is cold, which is why committing it is safe; as evidence it is warm,
revealing run timing and target selection, which is why the readings family
sits inside the read-block and no later reading receives it.

### Determinism and the assembler version

The walk is lexicographic over cleaned POSIX paths, hashing is `sha256`, JSON
is emitted with sorted keys, and no timestamp, host name, absolute path or
map-iteration order reaches either artefact: the same state assembled twice is
byte-identical, which is the property itd-187's eval falsifies independently.
`reading.AssemblerVersion` is a semver constant carried into the manifest and,
per ruling (12), into run metadata; a golden hash of the rendered table pins
it, so changing the table without bumping the version fails the gate.

## Acceptance criteria mapping

| itd-183 criterion | How spc-61 satisfies it | Test |
| --- | --- | --- |
| Given a repository state, when the assembler runs, then its output contains no field on the exclusion list | Positive inclusion at field granularity; the projection emits only the fields a row names, and denied namespaces are unreachable by any include | `TestExcludedFieldsNeverReachTheBundle` (fixture repo carrying `origin`, production mode, Audit Notes, dispositions and grounds) |
| Given a new record type added under `.abcd/development/`, when the assembler runs, then that type is absent without any change to the assembler | Assembler rule 1: no include names a directory containing a record family, and `.abcd` is a denied namespace on every path component | `TestNewRecordFamilyIsAbsentWithoutTableChange` (fixture adds `.abcd/development/inventions/`) |
| Given the invocation interface, when it is inspected, then it accepts a position and a target state and nothing else | Two operands, both closed-shape: four position tokens, `HEAD` or a hex sha | `TestAssembleRefusesFreeTextOperands`, `TestPositionTokenIsClosed`, `TestTargetRefusesBranchAndTag` |
| Given an assembler run, when the manifest is emitted, then every passed item appears with its path, its field where projection occurred, and a hash, and a reader can determine a named excluded field was not passed | Manifest `items[]` carries path, field and sha256 per item; `exclusions[]` names each excluded field with its detection signal | `TestManifestCoversEveryBundleItem`, `TestManifestAssertsNamedExclusions` |
| Given a reading invocation, when its context is constructed, then it contains the assembled input and no repository path | Bundle items are keyed `itm-NNNN` with a class label; the path mapping lives only in the manifest | `TestBundleCarriesNoRepositoryPath` (scans the serialised bundle for any path separator or known repo path fragment) |

## Tests

Every case below is written to fail against the empty package before the
change and to pass after. All live in `internal/core/reading/`, with the CLI
wiring pinned in `internal/surface/cli/`.

- `include_test.go`: `TestReadingsCharterCarriesTheRenderedIncludeTable`,
  `TestEveryFramingRowCitesAdr55`, `TestNoIncludeNamesARecordFamilyDirectory`
  (rule 1 as an executable property of the table),
  `TestWideningExcludesDraftsAndPlannedEntailmentIncludesThem`,
  `TestAssemblerVersionCoversTheIncludeTable`.
- `assemble_test.go`: `TestExcludedFieldsNeverReachTheBundle`,
  `TestNewRecordFamilyIsAbsentWithoutTableChange`,
  `TestBundleCarriesNoRepositoryPath`,
  `TestShippedIntentProjectsFiveFieldsOnly`,
  `TestAssembleRefusesDirtyIncludedPath`, and
  `TestWalkIsLexicographicAndByteStable` (two runs over one fixture state
  produce identical bundle bytes).
- `manifest_test.go`: `TestManifestCoversEveryBundleItem`,
  `TestManifestAssertsNamedExclusions`, `TestManifestCarriesNoTimestamp`,
  `TestManifestHashIsStableAcrossRuns`; `runid_test.go`:
  `TestRunIDIsAdr45Native` (injected clock and entropy).
- `internal/surface/cli/reading_surface_test.go`:
  `TestAssembleRefusesFreeTextOperands`, `TestPositionTokenIsClosed`,
  `TestTargetRefusesBranchAndTag`, `TestAssembleDryRunWritesNothing`,
  `TestReadingVerbReachesBothPlanes`.

## Out of scope

- **Running a reading.** The instrument ships unrun for the whole cycle: no
  reading is commissioned, no bundle is dispatched, and no reading record is
  produced by this delivery.
- Preventing timing and target-selection leakage. The operator still chooses
  when to run and what to point at; the manifest and the run record make that
  visible after the fact rather than preventing it. Disclosed residue.
- Prose-borne warmth inside an admitted chapter. It has no structural signal;
  the chapter-level include bound and the glossary discipline carry it, and it
  is disclosed as residue rather than claimed as caught.
- The comparative fallback interface (a shape-validated record-id list at
  invocation) stays recorded and unbuilt unless the four definitions show the
  evaluative position needs it.
