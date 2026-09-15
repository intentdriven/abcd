---
id: itd-2609021003095168
slug: the-invocation-is-a-position-and-a-target-state-and-the-comm
spec_id: spc-2609021004075744
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-183, itd-184, itd-199]
severity: major
impact: fix
origin: researcher-authored
production_mode: dictated-and-formatted
---

# The invocation is a position and a target state, and the committed preset for the position supplies what the reading is handed

Typed links: `builds_on` [itd-183](../shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md) (the assembler), [itd-184](../shipped/itd-184-four-cold-reading-definitions-one-blindness-core-each-positi.md) (the operand pin), [itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md) (the presets); `refines` [itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md) (its scope operand and override stamp are withdrawn; its presets stay).

## Press Release

> **A reading is commissioned with two things, as the design says: where it stands and what it reads against.** `abcd reading assemble --position <position> --target <commit>` is the whole invocation. What the reading is handed comes from the committed preset for that position, applied by the assembler with no flag; changing it is a commit to the preset file, reviewed and inside the dirty gate. The manifest records the preset entry applied and its hash, so a run is reproducible from the commit it names. There is no scope flag, no override, and no override stamp.

> "I want to run the readings exactly as the design specifies them, and the design says position and target state," said an AI/agent researcher who runs the four positions against their own record. "If I want the detection reading to see more, I change the preset and commit it. Nothing about a run should depend on what I typed at the prompt."

## Why This Matters

The design fixes the invocation at a position and a target state; the reading's object and question come from the definition, and the operator's remaining residue is when to run and what state to point at. itd-199 added a scope operand and an override stamp because the instrument could not be pointed at anything, and adr-58 recorded that departure. The maintainer has ruled that Iteration 2 runs exactly as the design specifies, and adr-2609021016286571 supersedes adr-58 accordingly.

What the scope operand did is not lost. The committed presets already map each position to what it reads, and a preset is a record fact the definition and the repository supply rather than something typed. Applying the position's preset with no operand keeps the calibration lever and removes the channel the design closes.

## What's In Scope

- **The assembler's invocation is `--position` and `--target`**, and a third operand of any kind is refused by name; itd-184's operand pin is updated to the two and fails closed on any addition, as it was designed to.
- **The committed preset for the position is applied by the assembler**, with no operand naming it. The preset file keeps its shape from itd-199 (kinds, records and paths per position, windows per the preset-windows intent); the `cold` and `warm` names retire as invocation tokens, and one entry per position stands. A repository that wants a wider reading commits a wider entry.
- **The manifest records the preset entry applied and its hash**; the override stamp leaves the manifest and the bundle, because nothing can be overridden.
- **No repository path is accepted at the invocation**, as before; the preset file remains the only place a path may be named.
- **The plugin surface page, the brief's reading chapter, the definitions' Object sections and the generated command reference** say two operands and no scope.

## What's Out of Scope

- The presets' content and windows, which the preset-windows intent carries.
- The comparative position's candidate run, which adr-2609021016272867 derives from the record; no operand there either.
- Any change to the blindness core beyond the fourth-condition sentence below, or to the supply regimes.

## Mechanism

We expect applying the committed preset per position with no operand to keep the instrument pointable without the channel the design closes, because the preset is the record's statement of what a position reads and the manifest names which commit's statement a run used. It fails if a run needs material no committed preset names, in which case the remedy is a commit to the preset, and the failure is visible as a dry run whose size report shows the gap.

## Scope Conditions

- The preset file is committed and inside the dirty gate; a run against an uncommitted preset edit refuses, as today. <!-- cond: cond-2609021004076748 -->
- One preset entry per position. A second named preset is not a concept this intent keeps; a repository with two calibrations commits one and records the other in its history. <!-- cond: cond-2609021004074586 -->
- The impact is `fix`: the verb's third operand shipped in v0.7.0 with no user able to hand its output to a reader, and the design the verb exists to serve never admitted the operand. <!-- cond: cond-2609021004073841 -->

## Acceptance Criteria

- **Given** an invocation carrying a `--scope` operand or any operand beyond position and target, **when** it runs, **then** the verb refuses and names the two operands the design admits.
- **Given** an invocation with a position and a target, **when** the assembly runs, **then** the bundle carries what the committed preset for that position names and nothing else, and the manifest records the preset entry applied and its hash.
- **Given** an uncommitted edit to the preset file, **when** an assembly runs, **then** the verb refuses, as it does today.
- **Given** the manifest of any run, **when** it is read, **then** it carries no override stamp and no scope source, and a reader can reproduce the run from the commit and the preset entry it names.
- **Given** itd-184's operand pin, **when** a third operand is added to the assemble verb, **then** the pin fails closed and names it.
- **Given** the plugin page, the brief's reading chapter, the four definitions and the generated command reference, **when** they are read, **then** each states two operands and none mentions a scope operand.
- **Given** the four definitions, **when** their shared blindness core is read, **then** its fourth condition says items are returned in the order they arise in the object, byte-identical across the four, each definition's `prompt_version` has moved PATCH and the agents changelog names the change (correction (7) of the 2026-09-02 ruling; iss-2609021153261145, resolved by this intent).

## Prior Art

- The design framework v4 section 8.2 and ruling M8; the companion v4 section 4.1; adr-2609021016286571, which supersedes adr-58; adr-2609021016272867.
- [itd-199](../shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md) (the presets, kept; the operand, withdrawn), [itd-184](../shipped/itd-184-four-cold-reading-definitions-one-blindness-core-each-positi.md) (the operand pin).

## Open Questions

None.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-2d81906de2e8 -->
Fidelity review — receipt rcp-2d81906de2e8 (verifier intent-auditor claude-opus-5[1m]).

Provenance: intent-auditor@claude-opus-5[1m] · rubric_hash sha256:0e09f874c820362be6f80ad9eacacb7c37138ad450a3c46a9eed429b2b7239f1 · prompt_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5
Input attestations: diff:0026e8f5^..0026e8f5@sha256:3ce581c227b74dac589333d119bf0f137da2907885a8939fd0d41fee4c78dede;

Acceptance rollup: MET 7 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Run live at 0026e8f5, `--scope cold` exits 2 with a message naming --position and --target, and a positional argument exits 2 naming the same two; both refusals are in the delivered source and both are held by tests that pass.
  evidence: internal/surface/cli/reading.go:86 — "reading assemble: this verb takes no positional argument; a reading's object and question come from its definition, and the invocation carries --position and --target, and nothing else"
  evidence: internal/surface/cli/reading.go:268 — ". The invocation is --position and --target, and nothing else: a reading's object and question come from its definition, and what it is handed comes from the committed preset entry for the position, in"
  evidence: internal/surface/cli/reading_surface_test.go:186 — "func TestAssembleRefusesAScopeOperand(t *testing.T) {"
  evidence: internal/surface/cli/reading_surface_test.go:163 — "func TestAssembleRefusesFreeTextOperands(t *testing.T) {"
- ac-2 — MET: A live `assemble --position detection --target HEAD --out` wrote a bundle whose preset block is exactly the committed detection entry's three kinds and 107 items drawn from those kinds alone, and a manifest carrying `preset` (the applied selectors) and `preset_hash`; PresetFor applies the entry for the invoked position with no operand and refuses a position the file does not name rather than defaulting to everything.
  evidence: internal/core/reading/scope.go:447 — "func PresetFor(pf PresetFile, position Position) (AppliedPreset, error) {"
  evidence: internal/core/reading/manifest.go:160 — "Preset AppliedPreset `json:"preset"`"
  evidence: .abcd/config/reading-presets.json:14 — ""detection": { "kinds": ["brief-section", "discipline", "spec"]"
  evidence: internal/core/reading/scope_test.go:255 — "func TestBundleCarriesThePresetAndManifestCarriesItsHash(t *testing.T) {"
  evidence: internal/core/reading/scope_test.go:857 — "func TestAssemblyAppliesTheCommittedPresetForThePosition(t *testing.T) {"
  evidence: internal/core/reading/scope_test.go:45 — "func TestThePresetNarrowsNeverWidens(t *testing.T) {"
- ac-3 — MET: The dirty gate still covers the preset file: an assembly against an uncommitted preset edit refuses and the refusal names the preset path, and untracked and symlinked preset files refuse too; all three tests pass at 0026e8f5.
  evidence: internal/core/reading/scope_test.go:327 — "func TestUncommittedPresetRefuses(t *testing.T) {"
  evidence: internal/core/reading/scope_test.go:337 — "if !strings.Contains(err.Error(), PresetConfigPath) {"
  evidence: internal/core/reading/scope_test.go:585 — "func TestAnUntrackedPresetRefuses(t *testing.T) {"
- ac-4 — MET: The manifest written by a live run carries the keys _type, schema_version, run_id, position, target_commit, assembler_version, preset, preset_hash, items, exclusions and no override or scope-source key at any level; two assemblies of one commit produce byte-identical bundles and manifests differing only in the run id.
  evidence: internal/core/reading/manifest.go:156 — "There is no override stamp. It counted departures from the committed presets, which was worth counting while an operand could depart from them; nothing can now"
  evidence: internal/core/reading/scope_test.go:953 — "for _, gone := range []string{"scope", "scope_hash", "scope_overridden"} {"
  evidence: internal/core/reading/scope_test.go:977 — "func TestRunIsReproducibleFromCommitAndPreset(t *testing.T) {"
  evidence: internal/core/reading/scope_test.go:387 — "func TestThePresetHashIsOrderIndependent(t *testing.T) {"
- ac-5 — MET: itd-184's readingOperands pin is re-pinned to dry-run, out, position, target, and the guard adds a --framing flag to a cobra tree built for the purpose and asserts both that the pin refuses and that the refusal names the added operand; the mutation is on a constructed tree, not the worktree, and the test passes.
  evidence: internal/surface/cli/regime_surface_test.go:90 — ""abcd reading assemble": {"dry-run", "out", "position", "target"},"
  evidence: internal/surface/cli/regime_surface_test.go:329 — "t.Run("a third operand fails the pin closed", func(t *testing.T) {"
  evidence: internal/surface/cli/regime_surface_test.go:348 — "if !strings.Contains(strings.Join(msgs, "\n"), "framing") {"
- ac-6 — MET: A grep over the seven named surfaces at 0026e8f5 returns no --scope, and each names both operands; the guard checks all seven for --scope, checks that every shipped assemble invocation names --position and --target, and checks the verb's own Use line at the source the reference is generated from. The guard sits in internal/surface/cli rather than internal/core/lint, which is a placement choice no criterion constrains and which costs no coverage: it runs under `go test ./...` in preflight and in CI's always-on Linux unit lane.
  evidence: internal/surface/cli/reading_surface_test.go:795 — "func TestReadingSurfacesNameTwoOperands(t *testing.T) {"
  evidence: internal/surface/cli/reading_surface_test.go:812 — "if strings.Contains(text, "--scope") {"
  evidence: commands/reading.md:69 — "**The invocation is two operands and nothing else.**"
  evidence: docs/reference/cli/commands.md:817 — "**Usage:** `abcd reading assemble --position <position> --target <HEAD|sha> [flags]`"
  evidence: .abcd/development/brief/04-surfaces/23-reading.md:39 — "| `--position` | one of `widening`, `entailment`, `comparative`, `detection`"
- ac-7 — MET: The fourth condition now opens with the companion v4 section 2 sentence "Items are returned in the order they arise in the object", replacing "Items come back unordered and unweighted", byte-identical across the four definitions and held so by a test; all four prompt_version values moved PATCH (widening 0.2.0 to 0.2.1, the other three 0.1.0 to 0.1.1) and agents/CHANGELOG.md carries one dated entry naming iss-2609021153261145 and all four definitions.
  evidence: agents/cold-reading-detection.md:77 — "4. **No ranking or prioritisation.** Items are returned in the order they arise in the object"
  evidence: agents/cold-reading-widening.md:8 — "prompt_version: 0.2.1"
  evidence: agents/CHANGELOG.md:3 — "## 2026-09-02 (iss-2609021153261145 — the fourth condition takes the companion's sentence)"
  evidence: internal/core/reading/definitions_test.go:685 — "func TestTheFourthConditionTakesTheCompanionsSentence(t *testing.T) {"
  evidence: internal/core/reading/definitions_test.go:147 — "func TestBlindnessCoreIsByteIdenticalAcrossDefinitions(t *testing.T) {"

Gap audit:
- honoured:
  - The assembler's invocation is --position and --target, and a third operand of any kind is refused by name
    evidence: internal/surface/cli/reading.go:266 — "func readingAssembleFlagError(_ *cobra.Command, err error) error {"
    evidence: internal/surface/cli/reading.go:86 — "this verb takes no positional argument"
  - The committed preset for the position is applied by the assembler with no operand naming it, and a position the file does not name refuses rather than defaulting to everything
    evidence: internal/core/reading/scope.go:452 — "names no entry for the %s position, so a run there would assemble nothing; an entry that selects nothing is a refusal rather than an empty bundle"
  - The manifest records the preset entry applied and its hash; the override stamp leaves the manifest and the bundle
    evidence: internal/core/reading/manifest.go:161 — "PresetHash string `json:"preset_hash"`"
    evidence: internal/core/reading/manifest.go:27 — "const SchemaVersion = 5"
  - cold, warm and extends retire together; the loader refuses a file holding a second preset because nothing at the invocation could choose between them
    evidence: internal/core/reading/scope.go:466 — "holds %d presets; one entry per position means one preset"
    evidence: internal/core/reading/scope.go:178 — "`extends` retired with the second preset name."
  - No repository path is accepted at the invocation and none enters the bundle; a location narrowing reaches the reading as a count and never as a path
    evidence: internal/core/reading/manifest.go:89 — "LocationNarrowings counts the location-based narrowings applied. It is a count and never a list, because the list would be the paths."
  - itd-184's operand pin is updated to the two operands and was watched failing closed on an added third
    evidence: internal/surface/cli/regime_surface_test.go:337 — "cmd.Flags().StringVar(&added, "framing", "", "a third operand nobody decided on")"
  - The plugin page, the brief's reading chapter, the four definitions and the generated command reference each state two operands and none mentions a scope operand
    evidence: internal/surface/cli/reading_surface_test.go:797 — ""commands/reading.md", "docs/reference/cli/commands.md", ".abcd/development/brief/04-surfaces/23-reading.md", "agents/cold-reading-widening.md""
  - The blindness core's fourth condition takes the companion's sentence byte-identically across four definitions, with four PATCH bumps and one changelog entry
    evidence: agents/CHANGELOG.md:14 — "### cold-reading-widening 0.2.1"
    evidence: agents/cold-reading-comparative.md:88 — "Items are returned in the order they arise"
- diverged:
  - The surviving preset entry is keyed `default`, a preset NAME the record does not decide. adr-2609021016286571 and the spec retire cold and warm and name no replacement key, and the Iteration 2 divergence register — which by its own terms lists every point the materials chose where the documents are open — carries no entry for it. The delta is inert at the invocation, since solePreset refuses any file holding other than one preset and nothing selects the key, and the intent's own In Scope clause that the preset file keeps its shape from itd-199 admits the naming level; but the committed record now carries a name chosen by the implementation rather than by a decision record.
    evidence: .abcd/config/reading-presets.json:4 — ""default": {"
    evidence: internal/core/reading/scope.go:465 — "if len(pf.Presets) != 1 {"
    evidence: .abcd/development/research/notes/2026-09-02-iteration-2-divergence-register.md:13 — "A point not listed here is one the documents settle and the materials follow."
- missing: (none)

Scope-condition dispositions:
- cond-2609021004076748 — survived: The preset file is still read from the committed tree inside the dirty gate: an assembly against an uncommitted preset edit refuses naming the preset path, and an untracked or symlinked preset file refuses too.
  evidence: internal/core/reading/scope_test.go:327 — "func TestUncommittedPresetRefuses(t *testing.T) {"
  evidence: internal/core/reading/scope_test.go:330 — "// deliberately NOT committed"
  evidence: internal/core/reading/scope_test.go:585 — "func TestAnUntrackedPresetRefuses(t *testing.T) {"
- cond-2609021004074586 — survived: The committed file holds one entry per position for the three assembling positions and a second named preset is structurally refused by the loader, with the refusal naming both offending presets; the retired warm calibration is left in git history rather than in the file, which is the remedy the condition itself names.
  evidence: .abcd/config/reading-presets.json:5 — ""positions": { "widening": ..., "entailment": ..., "detection": ... }"
  evidence: internal/core/reading/scope_test.go:88 — "func TestASecondNamedPresetIsRefused(t *testing.T) {"
  evidence: internal/core/reading/scope.go:273 — "names %d presets (%s); the invocation names none, so"
- cond-2609021004073841 — survived: The intent ships stamped impact: fix, and the premise holds — the CHANGELOG's 0.7.0 section is where `abcd reading assemble --scope` and the override stamp were announced, and framework v4 section 8.2 with ruling M8 fixes the invocation at a position and a target state, so the design the verb serves never admitted the operand.
  evidence: .abcd/development/intents/shipped/itd-2609021003095168-the-invocation-is-a-position-and-a-target-state-and-the-comm.md:10 — "impact: fix"
  evidence: CHANGELOG.md:80 — "**A reading is commissioned about something.** `abcd reading assemble --scope` takes a record id"
  evidence: CHANGELOG.md:74 — "## [0.7.0] - 2026-09-01"
## Grounds

- pursued: we expect a two-operand invocation with the committed preset applied per position to run Iteration 2 exactly as the design specifies while keeping the instrument pointable; a run that needs material no committed preset can name, with no way to commit it, would show it wrong
