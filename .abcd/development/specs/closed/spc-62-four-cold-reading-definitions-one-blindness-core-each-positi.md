---
id: spc-62
slug: four-cold-reading-definitions-one-blindness-core-each-positi
intent: itd-184
---

# Four cold-reading definitions: one verbatim blindness core, four objects, four licences no definition may borrow

## Bundle

itd-184,
[itd-183](../../intents/shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md)
and
[itd-185](../../intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md)
are one design under one bundle kind, and the ceremony cannot give them one
spec: a spec's `intent:` is a single id, captured as iss-2608300108376943.

| Spec | Component it owns |
| --- | --- |
| [spc-61](../closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md) | The input assembler, the include table, the pathless bundle, the manifest, and the bundle's shared decisions |
| spc-62 (this record) | The four reading definitions under `agents/` and the blindness-core byte-identity test |
| [spc-63](spc-63-one-ingest-verb-validates-every-cold-reading-output-includin.md) | The output contract, the supply-regime gate, and the ingest sub-verb |

The package name, the verb tree, the run-identifier form and the artefact
layout are the bundle's shared decisions, stated once in
[spc-61 § The package, the verb tree and the artefact layout](../closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md#the-package-the-verb-tree-and-the-artefact-layout-shared-bundle-decisions).
This spec uses them and does not restate them.

## Summary

spc-62 delivers itd-184's four definitions: four prompt files under `agents/`,
one per supply regime, each holding its object, its question, the blindness
core verbatim, its regime value, and its item shape, with a Go test holding the
core byte-identical across all four. One definition with four objects cannot
hold, because the prohibition against proposing is constitutive of the
detection pass and would void the widening pass entirely; four definitions as
instances within one detector context is ruling (13)'s form, and this spec
implements it as stated.

The definitions are the assertion half of the blindness. The enforcement half
is spc-61's assembler and its evals, and the licence check on what a reading
produced is spc-63's regime gate. A definition therefore states the posture
and never claims to be the wall.

## Scope

**In.** Four `agents/*.md` definitions; their injection-canary fixtures and
`agents/CHANGELOG.md` entries; the one-line extension of the `task_classes`
closed enum; the definition locator in `internal/core/reading`; the Go
byte-identity test on the shared core, and the tests that hold each definition to
its five parts and its regime value.

**Out.** Enforcing the blindness, which is the assembler's job checked by
[itd-186](../../intents/shipped/itd-186-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md)'s
and
[itd-187](../../intents/shipped/itd-187-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md)'s
evals; validating what a reading produced against its regime, which is
spc-63's gate; the reading-record schema, which is
[itd-180](../../intents/shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md)
and spc-58's.

## Approach

### Four files, named by position

`agents/` holds ten definitions today (`docs-currency-reviewer`,
`graveyard-interpreter`, `intent-auditor`, `lifeboat-reviewer`,
`press-release-composer`, `principle-distiller`,
`release-changelog-composer`, `ruthless-reviewer`, `security-reviewer`,
`sota-researcher`). Four more join them, named by position so the ingest verb
can resolve a run's position to its definition by construction:
`cold-reading-widening.md`, `cold-reading-entailment.md`,
`cold-reading-comparative.md`, `cold-reading-detection.md`. The shape
reference is [`agents/intent-auditor.md`](../../../../agents/intent-auditor.md),
the repository's existing verdict-emitting definition: an untrusted-input
notice, a scope block, an inputs block, a rubric, and exactly one fenced JSON
block as output.

### The five things, and the file's envelope

Each definition holds five things and nothing else: its object, its question,
the blindness core verbatim, its regime value, and its item shape.

| Definition | Regime | Object |
| --- | --- | --- |
| Widening | `generative` | Brief current text including the construal statement, glossary, disciplines, specs, and the shipped tree where one exists. Excludes `intents/drafts/` and `intents/planned/` |
| Entailment | `explicative` | The claim record, drafts and planned intents included, plus the constraint sources: disciplines, glossary, specs, brief current text |
| Comparative | `evaluative` | The candidate set, which is the widening reading's pre-admission output, against the declared selection criteria |
| Detection | `registrative` | The shipped tree against the claim record |

Questions, one per definition: widening asks what configurations the
construal admits that are not present in what has been committed to;
entailment asks what this design commits to, by being the kind of thing it is,
that its articulation does not state; comparative asks, for each candidate and
each declared criterion, how options of this shape ordinarily behave;
detection asks itd-86's question, the shipped tree against the claim record.

Item shapes match the bodies spc-63 validates: widening is configuration and
what admits it, with no third body field, so neither a preference nor a
comparison against what was built has anywhere to go; entailment is claim
surfaced, claim type and what implies it, surfaced and never dispositioned;
comparative is one item per candidate-criterion pair, being candidate id,
criterion and characterisation; detection is tension, constraint in play, and
why it is a tension. The pattern named sits in the record's envelope at every
position, never in a body (ruling (18)).

**The five things are the definition's substance; the itd-5 contract is the
file's envelope, not a sixth thing.** `record-lint`'s `agent_contract` rule
requires `prompt_version`, `reads_untrusted_input` and
`capability_scope.{task_classes,designed_for}` of every prompt in the tree,
requires an injection-canary fixture of any prompt declaring untrusted input,
and requires a per-agent `CHANGELOG.md` entry keyed on the version. Each
definition therefore ships at `prompt_version: 0.1.0` inside the `0.x`
calibration band, with `reads_untrusted_input: true` (a reading reads
repository text it did not write), a canary at
`agents/cold-reading-<position>/fixtures/injection-canary.json`, and a
CHANGELOG entry.

Two further frontmatter keys carry the machine-readable half of the five
things: `position:` and `regime:`. spc-63 resolves a run's position to the
definition file and reads `regime:` there, which is what makes the regime the
definition's property rather than the payload's.

`capability_scope.task_classes` draws from a closed enum documented in
[`02-constraints/04-naming.md`](../../brief/02-constraints/04-naming.md), which
is PR-to-extend and holds no token for this work. The four definitions take a
new token, `cold_reading`, added in the same change: reusing
`cross_document_audit` would name these prompts as audits, and an audit judges
against a standard, which is precisely the licence a widening reading does not
hold.

### The blindness core, verbatim and delimited

The core is carried in all four definitions, byte-identical, its seven
conditions in one order: no project context, no ledger access, no memory
across runs, no ranking or prioritisation, no selection, explanation or
commitment, named provenance on every item produced, and no passed input is
authoritative.

The seventh was adopted into the core on 2026-08-28 (ruling (8)): no document
passed to a reading is designated the fixed side of any comparison, so a
discipline, a glossary term or a declared criterion is as open to being named
in an item as anything else. Unlike the other six it is held by the core's
wording rather than by construction, because the assembler cannot enforce it,
and each definition discloses that in the sentence that carries it.

The core is delimited by `<!-- blindness-core:begin -->` and
`<!-- blindness-core:end -->` markers, so the byte-identity test compares an
exact span rather than a heuristic slice, and an edit to one copy fails the
gate rather than drifting.

### The object asymmetry, and the comparative reading's object

The widening and entailment asymmetry over draft and planned intents is
deliberate: the widening reading must not see the candidate set it is asked to
widen, and the entailment reading properly reads drafts, since articulation
precedes selection. Each definition states its own side of the asymmetry, and
the asymmetry is enforced in spc-61's include table, which is where a reading's
object is actually bounded.

itd-183 and itd-184 disagreed about the comparative reading's object. **Ruling
(8) settles it: the object is the widening reading's pre-admission output**,
read within one cycle before admission, against the selection-criteria
discipline
[itd-191](../../intents/disciplines/itd-191-the-selection-criteria-are-a-declared-recorded-discipline-a.md).
Criteria are never supplied at invocation, and a prior run's stored output
stays read-blocked. Where a widening reading returns fewer than two
configurations the comparative reading is not exercised at all, and the
outcome is recorded as such (ruling (18)).

### What the definitions assert but do not enforce

Two obligations are stated in every definition and are honestly disclosed as
assertions rather than mechanisms: that the dispatching host grants the
subagent no repository access, so the assembled bundle is the reading's whole
working set (brief invariant 15); and the seventh core condition above. The
widening prohibitions are a third case of a softer kind: a recommendation
among configurations, or a characterisation of one as better than another,
raises a review flag rather than an ingest refusal, because the generative
licence is the widest and comparison belongs to the comparative position.

## Acceptance criteria mapping

The criteria were split on 2026-08-31, before this spec was built, so that each
one names a single observable a gate can hold. The numbering below is the
positional authority ac-1..ac-3.

| itd-184 criterion | How spc-62 satisfies it | Test |
| --- | --- | --- |
| ac-1 — the blindness core is byte-identical across the four and carries its seven conditions in the fixed order | One delimited core span, carried verbatim in each file, the delimiters making the span exact rather than heuristic | `TestBlindnessCoreIsByteIdenticalAcrossDefinitions`, `TestBlindnessCoreCarriesSevenConditions` |
| ac-2 — every definition states a regime, the four distinct and resolvable by position alone | `position:` and `regime:` are frontmatter keys of the definition file, which is what makes the regime the definition's property rather than the payload's | `TestEveryDefinitionStatesItsRegime`, `TestRegimeValuesAreTheFourAndDistinct` |
| ac-3 — no registered flag and no registered configuration key sets or overrides a regime | The command tree is walked through `commandSurface`, the repository's one canonical cobra walk, and every flag and shorthand inspected. The configuration keys are enumerated from the git index — every tracked `.json` under `.abcd/` — with the two largest schema types also walked by reflection. Two written lists survive and are stated in the guard's header rather than implied: the reading verb's pinned operand set, which fails CLOSED and is a tripwire rather than an enumeration, and the one exclusion from the file walk, which fails OPEN | `TestNoOperatorSurfaceSetsARegime`, in `internal/surface/cli/regime_surface_test.go` |

The residue itd-184 discloses against ac-3 is a channel that was never
registered. This spec adds no mechanism for one, and the walk sees every channel
that is. Two narrower edges sit inside that residue, both stated in the guard's
own header rather than left to be discovered.

- A key declared by a schema type outside the two walked by reflection, and
  written into no tracked file, is unregistered in both senses until a file
  carries it — at which point the index walk reaches it.
- A knob written into one of the machine-written baselines
  (`*-baseline.json`) is skipped. That exclusion exists because the citations
  baseline uses cited URLs as its map keys, so a paper whose URL says `regimes`
  would turn the guard red while naming no knob; nothing reads a baseline as
  configuration, and the exclusion is where that would change if anything ever
  did.

The file set is read from the git index rather than globbed from chosen
directories, and iss-2608311100496798 is why: two directories was a written list
and it had already fallen behind by four files — `personas.json`, the release
surface snapshot, the release-gate manifest and the ruleset mirror all sit
outside them, so a regime key in any of them passed.


## Tests

Each case is written to fail before the definitions land and pass after. The
Go tests live in `internal/core/reading/definitions_test.go`, which is the right
home because the definition LOCATOR lives in that package and is spc-62's own
delivery: `LoadDefinition(repoRoot, position)` resolves a position to its
definition file by construction, reads the `position:` and `regime:` keys out of
its frontmatter, and returns the file's sha256, which is the instrument identity
spc-63 stamps into a run. The root is a parameter, so a caller — the ingest verb,
or a test over a temporary tree — decides which repository is read. `Describe`,
behind the bare `abcd reading` verb, renders the definitions the locator resolves
rather than a directory listing, which is what keeps the locator wired.

- `TestBlindnessCoreIsByteIdenticalAcrossDefinitions`: extracts the delimited
  span from all four files and compares bytes; a one-character edit to one copy
  fails.
- `TestBlindnessCoreCarriesSevenConditions`: the span holds the seven
  conditions in the fixed order, so a silent deletion is caught as well as a
  divergence.
- `TestEveryDefinitionStatesItsRegime` and
  `TestRegimeValuesAreTheFourAndDistinct`: four files, four positions, four
  distinct regime values, resolvable by position alone.
- `TestDefinitionHoldsItsFiveParts`: each file carries object, question, core,
  regime and item shape, and the item shape's field names match the body
  spc-63 validates for that regime, so the definition and the contract cannot
  drift.
- `TestWideningDefinitionExcludesDraftsAndPlanned` and
  `TestEntailmentDefinitionIncludesThem`: the stated object matches the
  assembler's include table for that position.
- `TestComparativeObjectIsTheWideningPreAdmissionOutput`: the settled reading
  of ruling (8), pinned so the two intents' disagreement cannot be
  reintroduced.
- `TestLoadDefinitionResolvesUnderAnArbitraryRoot`,
  `TestLoadDefinitionRefusesAMalformedDefinition` and
  `TestLoadDefinitionsSkipsAnAbsentDefinition`: the locator's own contract over a
  temporary root — a resolved position, regime and hash; a refusal by name for a
  definition silent about either key or stating a regime outside the closed set;
  and absence treated as a state while a present-but-broken definition is a
  fault. `TestDescribeReportsTheDefinitionsTheLocatorResolves` holds it wired, by
  requiring the bare verb to refuse what the locator refuses.
- `TestNoOperatorSurfaceSetsARegime`, in
  `internal/surface/cli/regime_surface_test.go`: walks the command tree and the
  configuration schema and fails if any registered flag or key can set or
  override a regime. Its own file, so the ingest verb's surface tests and this
  guard never contend for one.
- `record-lint` over the tree supplies the contract cases already: the four
  new prompts must carry the itd-5 frontmatter, a non-empty regular canary
  fixture each, and a CHANGELOG entry at `0.1.0`. A confirming case,
  `TestColdReadingDefinitionsSatisfyTheAgentContract`, runs the
  `agent_contract` rule over the tree from the test so the failure is local.

## Out of scope

- **Running a reading.** The instrument ships unrun for the whole cycle: the
  definitions are written, linted and tested, and none is dispatched.
- Calibration. The definitions sit in the `0.x` band, which declares them
  shipped and wired but honestly unmeasured; a `1.0.0` lock has to be earned
  against a corpus and none has been run.
- The selection criteria themselves, which are a recorded discipline
  (itd-191), never supplied at invocation and never authored here.
- Whether `held` is available at the widening position, deferred with a
  revisit point at the first widening run's dispositions.
