---
id: spc-2609021004075744
slug: the-invocation-is-a-position-and-a-target-state-and-the-comm
intent: itd-2609021003095168
origin: researcher-authored
production_mode: dictated-and-formatted
---

# The invocation is a position and a target state: the scope operand and the override stamp leave, and the committed preset for the position is applied by the assembler

## Summary

This spec delivers the intent that returns `abcd reading assemble` to the two
operands the design specifies. `--scope` is removed from the verb, the
assembler applies the committed preset entry for the invoked position, the
manifest records that entry and its hash in place of the scope source and the
override stamp, itd-184's operand pin is re-pinned to the two operands, and
every surface that describes the invocation says two operands. adr-2609021016286571
supersedes adr-58; adr-2609021016272867 derives the comparative position's widening run
from the record, so no position needs a third operand.

## Scope

In: the CLI surface of `reading assemble`; the preset loader and its file
shape; the assembly's scope resolution; the manifest and bundle fields that
carried the scope; the operand pin; the plugin page, the brief's reading
chapter, the four definitions' Object sections and the generated command
reference; the tests that pinned the three-operand shape; the fourth
condition of the four definitions' shared blindness core.

Out: the presets' content and windows (the preset-windows intent); the
comparative position's derived run (the comparative-channel spec); the
include table (itd-194).

## Approach

### The verb

`internal/surface/cli/reading.go` drops the `--scope` flag and the refusal
text that named it; the positional-operand refusal names `--position` and
`--target` only. `TestAssembleRefusesFreeTextOperands` keeps refusing a
positional; a new `TestAssembleRefusesAScopeOperand` asserts that `--scope`
is an unknown flag, exit 2, nothing written.

### Preset application

`internal/core/reading/scope.go` keeps `LoadPresets`, the file's shape
(`schema_version`, `presets`, per-position `kinds`, `records`, `paths`, and
the `window` the preset-windows spec adds) and every load-time refusal
(uncommitted edit, symlink, path outside the tree, denied segment). What
changes is resolution: `ResolveScope(token)` becomes `PresetFor(position)`,
which returns the one committed entry for the position and refuses when the
file holds no entry for it. The `cold` and `warm` names retire as tokens; the
committed file is reshaped to one entry per position (the preset-windows spec
owns the entry contents and their measured figures, and lands after this
spec in the landing order below). `extends` is removed with the second name.

### The manifest and the bundle

`Manifest.Scope`, `Manifest.ScopeHash` and `Manifest.ScopeOverridden`
(`manifest.go`) are replaced by `Manifest.Preset` (the entry applied, as
selectors) and `Manifest.PresetHash`; `BundleScope` keeps carrying the kinds
and records the reading was handed and its location-narrowing count, renamed
`BundlePreset`, so the reading still knows what it was given (itd-199's ac-5
reasoning stands). `SchemaVersion` moves by one because the field set
changes. `TestBundleScopeCarriesNoRepositoryPath` is kept under the new name.

### The operand pin

`readingOperands` in `internal/surface/cli/regime_surface_test.go` pins the
assemble verb's operand set, and `TestNoOperatorSurfaceSetsARegime` walks the
command tree against it, failing closed on any addition, as itd-184 designed
it. The pin is updated to `position`, `target`, and the test gains a case
that adds a third operand and watches the pin fail.

### Surfaces

`commands/reading.md` (the invocation section, the examples, the manifest
field list), `.abcd/development/brief/04-surfaces/23-reading.md`,
`docs/reference/cli/commands.md` (regenerated), and the `## Object` section of
each of the four definitions under `agents/` state two operands and no scope;
the definitions' precedence sentence ("where the definition and the bundle
disagree, the bundle governs") stays, with "bundle" now describing the
preset-derived content. Brief invariant 15 already carries the two-operand
enumeration (amended with adr-2609021016286571).

### The blindness core's fourth condition

The four definitions share one blindness core, and its fourth condition says
items come back unordered and unweighted where the readings companion's
section 2 says items are returned in the order they arise in the object. On
correction (7) of the 2026-09-02 ruling the condition takes the companion's
sentence: In each of `agents/cold-reading-widening.md`,
`agents/cold-reading-entailment.md`, `agents/cold-reading-comparative.md` and
`agents/cold-reading-detection.md` the sentence "Items come back unordered
and unweighted" becomes "Items are returned in the order they arise in the
object", byte-identical across the four, with the rest of the condition (no
severity, no confidence score, no most-important-first, no top-N, and the
reason) unchanged. The four `prompt_version` values move PATCH together, and
`agents/CHANGELOG.md` gains one entry naming the four. This is a change to
the definitions and not to the bundle, so it moves neither `SchemaVersion`
nor `AssemblerVersionCore`; the definition content hash in the run metadata
moves by construction. This resolves
[iss-2609021153261145](../../../work/issues/resolved/iss-2609021153261145-the-blindness-core-s-fourth-condition-says-items-come-back-u.md).

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

This spec lands first in Phase A, because the preset-windows spec measures
and declares per-position entries in the one-entry-per-position shape this
spec introduces, and the comparative spec's derived run assumes no operand.
Versions: `SchemaVersion` +1 (field set), `AssemblerVersionCore` MINOR
(bundle shape), preset `schema_version` unchanged here (the preset-windows
spec moves it), the four definitions' `prompt_version` PATCH (the fourth
condition above).

## How the Acceptance Criteria are satisfied

- **ac-1 (a third operand refuses).** `--scope` is gone from the flag set,
  so cobra refuses it as unknown; a positional refuses as today. Tests:
  `TestAssembleRefusesAScopeOperand`, `TestAssembleRefusesFreeTextOperands`.
- **ac-2 (the preset for the position is applied and recorded).**
  `PresetFor(position)` supplies the selectors the include table's admission
  is intersected with; the manifest carries `preset` and `preset_hash`. Test:
  `TestAssemblyAppliesTheCommittedPresetForThePosition` asserts the bundle
  item set equals the intersection and the manifest names the entry.
- **ac-3 (uncommitted preset refuses).** Unchanged from itd-199:
  `TestUncommittedPresetRefuses`.
- **ac-4 (no override stamp, reproducible).** The manifest struct has no
  `scope_overridden` and no `scope.source`; `TestManifestCarriesNoOverride`
  decodes a manifest and asserts both keys absent, and
  `TestRunIsReproducibleFromCommitAndPreset` assembles twice at one commit
  and compares bundles byte for byte (the amnesia eval's property, exercised
  here on the preset path).
- **ac-5 (the pin fails closed).** `TestNoOperatorSurfaceSetsARegime` over
  `readingOperands` with the added-operand case.
- **ac-6 (every surface says two operands).** `TestReadingSurfacesNameTwoOperands`
  greps the plugin page, the brief chapter, the four definitions and the
  generated reference for `--scope` and for the phrase the design uses, and
  fails on a hit or a miss.
- **ac-7 (the fourth condition takes the companion's sentence).** The one
  sentence in the four definitions, the four PATCH bumps and the changelog
  entry. `TestBlindnessCoreIsByteIdenticalAcrossDefinitions` (existing) holds
  the four cores identical; `TestTheFourthConditionTakesTheCompanionsSentence`
  in `definitions_test.go` asserts the core's fourth condition carries the
  sentence "items are returned in the order they arise in the object" and
  not "unordered and unweighted", and that each definition's
  `prompt_version` has moved.

## Tests

`internal/surface/cli`: `TestAssembleRefusesAScopeOperand`,
`TestAssembleRefusesFreeTextOperands`, `TestNoOperatorSurfaceSetsARegime`
(over `readingOperands`).
`internal/core/reading`: `TestAssemblyAppliesTheCommittedPresetForThePosition`,
`TestPresetForRefusesAMissingPosition`, `TestManifestCarriesNoOverride`,
`TestRunIsReproducibleFromCommitAndPreset`, `TestUncommittedPresetRefuses`
(kept), `TestBundlePresetCarriesNoRepositoryPath` (renamed).
`internal/core/lint` or the docs lane: `TestReadingSurfacesNameTwoOperands`.
`internal/core/reading`: `TestBlindnessCoreIsByteIdenticalAcrossDefinitions`
(existing), `TestTheFourthConditionTakesTheCompanionsSentence`.
`evals/`: the read-block and amnesia lanes run unchanged over the new
invocation; the oracle's pinned counts do not move.

## Out of scope

The preset entries' contents, windows and measured figures; the mechanism
proportion line; the derived widening run at the comparative position; the
include table's narrowing and the unscanned mark.
