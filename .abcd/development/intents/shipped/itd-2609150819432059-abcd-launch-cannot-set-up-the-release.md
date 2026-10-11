---
id: itd-2609150819432059
slug: abcd-launch-cannot-set-up-the-release
spec_id: spc-2609202019026366
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
related_issues: [iss-2609061432214212]
origin: extracted-from-record
production_mode: hand-written
impact: additive
---

# A managed repository that is not a plugin gets the same release gate

## Press Release

> **A repository abcd manages declares what it ships, and the release flow follows the declaration.** A managed Go application, a macOS app with its own tag-driven workflow, or any repository that is not a plugin can today only say what it is not by failing, and its operator cuts every release by hand with the release gate never running. With this change the repository declares its artefact kind once. `launch scaffold` lays what that kind needs and refuses to guess the rest. `launch --dry-run` and `launch ship` run against the declared kind, so the changelog-driven gate, the deferral read and the derived version reach every managed repository, not only the one that ships a plugin. A repository with its own release workflow keeps it.

## Why This Matters

The launch verbs assume a plugin at three places: The payload include config, the lockstep table of plugin manifests, and the source-bundle scan. Two managed repositories hit that wall at v0.9.0, and one of them cut three releases in a single day entirely by hand: A dated changelog heading on a release branch, deferral keys written into open major records by script, a pull request, a tag from the repository's own workflow. The release-cut gate that refuses on an open major captured since the anchor tag never ran there; the session enforced it by reading the ledger. The cost is measured, and the remedy is the capability, not a patch. Graduated from `iss-2609061432214212`.

## Mechanism

We expect the gate to reach every artefact kind once the kind is declared, because the gate's inputs (the ledger, the anchor tag, the version location) are already kind-independent and only the manifest reads bind it to a plugin; it is shown wrong if some kind needs a gate input that no declaration can supply.

## Scope Conditions

- A repository abcd manages (a marker block fired), on a forge with a merge queue or branch protection, whose releases are tagged by a workflow abcd can scaffold or call. <!-- cond: cond-2609202019020629 -->
- A single artefact per repository; a repository that ships several artefacts of different kinds is outside this claim. <!-- cond: cond-2609202019027680 -->

## What's In Scope

- Artefact kinds in the first cut: A plugin (the shipped shape), a Go binary, and an application with its own build and publish steps. Any other kind is refused by name.
- The declaration, the kind-shaped scaffold, and the launch verbs reading the declaration.

## What's Out of Scope

- The write path for `deferred_after` and `deferral_reason`, which stays `iss-2609181223260994`; this intent delivers the read half of that seam.
- A hand-written changelog or version for any kind: The changelog stays derived (adr-37) and the version derived (adr-31).
- What a hand-merged workflow does with the gate it calls: The binary proves the gate, the derivation and the refusals; the runbook asks a human to hold the rest.

## Acceptance Criteria

- **Given** a managed repository with no `.abcd/config/artefact.json`, **when** `launch --dry-run` runs, **then** it refuses naming that file as the declaration's home and listing the kinds it accepts, never with a missing-file error; and `ahoy` reports the absence as a gap `ahoy install` can close.
- **Given** an artefact file declaring a non-plugin kind and a lockstep list, **when** `launch --dry-run` runs, **then** the lockstep check reads the primary from `version-location.json` (adr-19) and every listed file, refuses a listed path it cannot read, reads no plugin manifest, and requires no payload include config.
- **Given** a declared non-plugin kind and no `CHANGELOG.md`, **when** `launch scaffold` runs, **then** afterwards a changelog holding only the empty `[Unreleased]` anchor exists, the gate workflow exists as a file of its own with a named empty build job, and the report names every file written.
- **Given** a repository with its own release workflow, **when** `launch scaffold` runs, **then** that workflow is byte-for-byte what it was, the gate workflow is written beside it, and the report names the existing file as left alone and states what to add to it to call the gate.
- **Given** a scaffolded gate workflow later edited by hand, **when** `launch scaffold` runs again, **then** it refuses naming the file until `--confirm`, as it does for abcd's own scaffold today.
- **Given** a declared kind whose release cut carries an open major captured since the anchor tag with no deferral, **when** `launch ship` runs, **then** it refuses naming the record, exactly as it does for a plugin.
- **Given** a declared kind whose only post-anchor major carries `deferred_after` naming the anchor tag and a `deferral_reason`, **when** `launch ship` runs, **then** the guard passes and the receipt names the deferred record.
- **Given** a declared non-plugin kind, **when** `launch --dry-run` runs, **then** the identity scan runs over the tree the tag would archive minus the record namespace, and the report says which tree was scanned.
- **Given** a kind the binary does not know, **when** any launch verb runs, **then** it refuses naming the kind and the accepted set, and writes nothing.

## Decisions

Settled in the planning interview with the product thinker on 2026-09-20; each replaces the open question it answers.

1. **Declaration home:** A new `.abcd/config/artefact.json` holds the artefact kind and, with it, the lockstep list and later the kind's opt-ins. Not the repo config, not the version-location file.
2. **Lockstep secondaries:** The artefact file names the files held in lockstep with the primary; the check reads what was declared and refuses a declared path it cannot read.
3. **Adoption home:** An `ahoy install` gap. A managed repository with no declaration is a gap ahoy detects, prompts for and writes; `launch scaffold` then lays the kind's files.
4. **Seeded file class:** Drift-checked, as the scaffolded workflows are today; a later hand edit refuses the next scaffold run until `--confirm`.
5. **An existing release workflow:** Left byte-for-byte; abcd's gate is scaffolded as a separate workflow the repository's own calls before its build step, and the report says what to add.
6. **Build and asset shape:** Gate plumbing only. The scaffold lays the verify, derive, changelog, receipt and tag steps and a named empty build job the repository fills or calls; `iss-2608270559310755` stays its own record and is not absorbed.
7. **Changelog seeding with tags and no changelog:** An empty `[Unreleased]` anchor only; history before adoption is not represented.
8. **Payload for a built artefact:** The preview scans the source tree the tag would archive, minus the record namespace, exactly as a plugin payload is scanned; an empty include set is not a refusal for a non-plugin kind.
9. **The release-rendered site** (`itd-2609061543533170`): A separate intent built on this one; the artefact file reserves a site opt-in this intent does not act on.

## Typed Links

- **refines `itd-93`** (a scaffolded release gate that works on the first try): Widens it from the one artefact kind it assumes to a declared one.
- **refines `adr-37`, `adr-31`** (changelog-driven releases, derived versioning): Every kind keeps both; nothing here adds a hand-written path.
- **built on by `itd-2609061543533170`** (the release-rendered site): Reads the artefact file this intent introduces.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-97519c4308ad -->
Fidelity review — receipt rcp-97519c4308ad (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:0ba7df9fbbcf7cab4dc9553e264ddd0f97116f904f3f870ce848e360105749ed
Input attestations: commit:b52746fb3 (spc-2609202019026366 close, itd-2609150819432059 ships); tree audited at 52c2236a55830421c2af5fa58f5a196c4eb7fdd5@-;

Acceptance rollup: MET 9 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: LoadArtefact turns an absent file into a PreflightError wrapping ErrNoArtefact whose message names .abcd/config/artefact.json and the three kinds; DryRun surfaces it (TestDryRunWithNoArtefactDeclarationRefusesNamingItsHome); ahoy raises artefact.missing as Required and Resolvable inside the managed-repo branch, and install writes it (TestDetectRaisesArtefactMissingUntilTheKindIsDeclared, TestInstallPromptsForTheKindAndWritesIt); all pass at BASE
  evidence: internal/core/launch/artefact.go:117 — "return Artefact{}, &PreflightError{msg: noArtefactMessage(), err: ErrNoArtefact}"
  evidence: internal/core/launch/artefact.go:97 — "this repository declares no artefact kind: " + ArtefactRelPath + " is where it is declared"
  evidence: internal/core/launch/dryrun_kind_test.go:15 — "func TestDryRunWithNoArtefactDeclarationRefusesNamingItsHome"
  evidence: internal/core/ahoy/artefact.go:58 — "Required: true, Resolvable: true,"
  evidence: internal/core/ahoy/detect.go:124 — "gaps = append(gaps, detectArtefact(abs)...)"
  evidence: internal/core/ahoy/artefact_test.go:46 — "func TestDetectRaisesArtefactMissingUntilTheKindIsDeclared"
- ac-2 — MET: CheckDeclaredLockstep reads the primary through version-location.json and every declared file, refuses an unreadable declared path by name, and reads no plugin manifest; the dry-run test for a binary previews with no payload include config and no plugin-only refusal
  evidence: internal/core/launch/lockstep.go:317 — "No plugin manifest is read."
  evidence: internal/core/launch/lockstep.go:358 — "declared lockstep file "+f.Path+" not readable"
  evidence: internal/core/launch/lockstep_declared_test.go:23 — "func TestDeclaredLockstepDevPassesWithEveryKeyAbsentAndReadsNoPluginManifest"
  evidence: internal/core/launch/lockstep_declared_test.go:47 — "func TestDeclaredLockstepRefusesADeclaredPathItCannotRead"
  evidence: internal/core/launch/dryrun_kind_test.go:57 — "func TestDryRunForABinaryScansTheArchivedTreeAndChecksTheDeclaredLockstep"
- ac-3 — MET: For a non-plugin kind the scaffold writes CHANGELOG.md as the empty [Unreleased] anchor only, abcd-release-gate.yml as its own file with a build job that is `mkdir -p dist` and no guessed build, and the report names every written file; TestScaffoldForABinaryLaysTheChangelogAndTheGateWorkflow passes
  evidence: internal/core/launch/scaffold/kind.go:28 — "ChangelogAnchor = "# Changelog\n\n## [Unreleased]\n""
  evidence: internal/core/launch/scaffold/kind.go:23 — "GateWorkflowPath = ".github/workflows/" + GateWorkflowName"
  evidence: internal/core/launch/scaffold/kind_test.go:105 — "func TestScaffoldForABinaryLaysTheChangelogAndTheGateWorkflow"
  evidence: internal/core/launch/scaffold/kind_test.go:118 — "run: mkdir -p dist"
- ac-4 — MET: An existing release.yml/release.yaml is found by Lstat and never opened, the gate is written beside it, the report marks it kept with 'left alone' and carries the workflow_call stanza; proven at the core (TestScaffoldLeavesAnExistingReleaseWorkflowByteForByte) and at the front door (TestLaunchScaffoldForABinaryWithItsOwnReleaseWorkflowPrintsTheStanza)
  evidence: internal/core/launch/scaffold/kind.go:34 — "var ownReleaseWorkflows = []string{".github/workflows/release.yml", ".github/workflows/release.yaml"}"
  evidence: internal/core/launch/scaffold/kind.go:40 — "const gateCallStanza = \`jobs: abcd-release-gate:"
  evidence: internal/core/launch/scaffold/kind_test.go:169 — "func TestScaffoldLeavesAnExistingReleaseWorkflowByteForByte"
  evidence: internal/surface/cli/launch_kind_test.go:137 — ""[kept] .github/workflows/release.yml", "[written] .github/workflows/abcd-release-gate.yml""
- ac-5 — MET: A hand-edited gate workflow makes the next scaffold return ErrScaffoldBlocked with the file marked refused and the edit kept; Confirm restores the machinery — the same drift class as the plugin scaffold's
  evidence: internal/core/launch/scaffold/kind_test.go:205 — "func TestScaffoldRefusesAHandEditedGateWorkflowUntilConfirm"
  evidence: internal/core/launch/scaffold/kind_test.go:214 — "if !errors.Is(err, ErrScaffoldBlocked) {"
  evidence: internal/core/launch/scaffold/writefiles_test.go:14 — "func TestWriteFilesKeepsASeedAndRefusesDriftedMachinery"
- ac-6 — MET: A declared binary's `launch ship` with an open major captured since the anchor tag and no deferral exits 1 naming the record (TestLaunchShipForADeclaredBinaryRefusesAnOpenMajor); the GuardFindings path is unchanged for every kind
  evidence: internal/surface/cli/launch_kind_test.go:76 — "func TestLaunchShipForADeclaredBinaryRefusesAnOpenMajor"
  evidence: internal/surface/cli/launch_kind_test.go:87 — "if !strings.Contains(string(out), "iss-90") {"
- ac-7 — MET: The same cut with deferred_after naming the anchor tag and a deferral_reason exits 0 and the report names 'deferred: iss-90' with its reason (TestLaunchShipForADeclaredBinaryPassesAndNamesADeferral)
  evidence: internal/surface/cli/launch_kind_test.go:94 — "func TestLaunchShipForADeclaredBinaryPassesAndNamesADeferral"
  evidence: internal/surface/cli/launch_kind_test.go:106 — ""deferred: iss-90""
- ac-8 — MET: For a non-plugin kind the bundle is git archive's view of HEAD minus the record namespace, the dry-run report's ScannedTree carries ArchiveTreeDescription and the CLI prints 'scanned tree: the tree the release tag would archive'; a secret under .abcd/ is not scanned as shipping
  evidence: internal/core/launch/bundle.go:1053 — "const ArchiveTreeDescription = "the tree the release tag would archive (git archive's view of HEAD, export-ignore honoured), minus the record namespace""
  evidence: internal/core/launch/dryrun.go:112 — "report.Bundle, report.ScannedTree = bundle, tree"
  evidence: internal/core/launch/dryrun_kind_test.go:63 — "report.ScannedTree != ArchiveTreeDescription"
  evidence: internal/surface/cli/launch_nopayload_test.go:57 — "func TestLaunchDryRunForADeclaredBinarySaysWhichTreeItScanned"
- ac-9 — MET: ParseArtefact refuses an unknown kind naming it and the accepted set, and TestEveryLaunchVerbRefusesAnUnknownKind drives dry-run, ship, scaffold, receipts and archive, asserting the refusal text and that neither the repository nor --out gained a file
  evidence: internal/core/launch/artefact.go:152 — "names the kind %q, which abcd does not know: the accepted kinds are %s"
  evidence: internal/surface/cli/launch_kind_test.go:36 — "func TestEveryLaunchVerbRefusesAnUnknownKind"
  evidence: internal/surface/cli/launch_kind_test.go:64 — "if status := r.Git("status", "--porcelain", "--ignored"); status != "" {"

Gap audit:
- honoured:
  - one reader of the declaration shared by every launch verb and ahoy, and ahoy proves what it writes through it before writing
    evidence: internal/core/launch/artefact.go:9 — "Every launch verb and ahoy go through it, so a kind one of them accepts is a kind all of them accept."
    evidence: internal/core/ahoy/artefact.go:79 — "if _, err := launch.ParseArtefact(data); err != nil {"
  - the shipped plugin shape adopts silently: a plugin manifest declares kind plugin without a prompt
    evidence: internal/core/ahoy/artefact.go:51 — "it carries a plugin manifest, so install declares kind plugin without asking"
    evidence: internal/core/ahoy/artefact_test.go:87 — "func TestInstallAdoptsKindPluginSilentlyForAPluginRepository"
  - the changelog and the version stay derived for every kind; the site opt-in is read and validated but not acted on
    evidence: internal/core/launch/artefact.go:81 — "Site is the release-rendered site opt-in (decision 9). It is read and validated here and acted on by itd-2609061543533170, not by this intent."
  - a plugin declaring a lockstep list is refused: the plugin keeps the pinned manifest table
    evidence: internal/core/launch/artefact.go:169 — "declares a lockstep list for kind plugin"
  - a repository whose own workflow releases receives no second release chain
    evidence: internal/core/launch/scaffold/kind_test.go:191 — "a repository whose own workflow releases must not receive a second release chain"
- diverged: (none)
- missing:
  - the acceptance rehearsal the spec names — a managed repository's next cut run through launch --dry-run and launch ship with the gate enforced by the binary — is not evidenced in this tree; the Grounds' falsifier (the two managed repositories still cutting by hand) remains unobserved either way
    evidence: .abcd/development/specs/closed/spc-2609202019026366-abcd-launch-cannot-set-up-the-release.md:63 — "The Gropius repository's own release is the acceptance rehearsal"
    evidence: .abcd/development/intents/shipped/itd-2609150819432059-abcd-launch-cannot-set-up-the-release.md:85 — "shown wrong if the two managed repositories still cut by hand after this ships"

Scope-condition dispositions:
- cond-2609202019020629 — survived: the artefact gap is raised only inside ahoy's managed-repository branch (marker block fired), and the delivery supplies both halves of the tagging assumption — a gate workflow abcd scaffolds and a call stanza for a workflow the repository already has; nothing delivered contradicts the forge assumption
  evidence: internal/core/ahoy/detect.go:124 — "gaps = append(gaps, detectArtefact(abs)...)"
  evidence: internal/core/launch/scaffold/kind.go:40 — "const gateCallStanza = \`jobs: abcd-release-gate:"
- cond-2609202019027680 — survived: the declaration admits exactly one kind as a single JSON string and refuses anything else, so a repository can only ever declare one artefact
  evidence: internal/core/launch/artefact.go:147 — "if err := json.Unmarshal(raw["kind"], &kind); err != nil || kind == "" {"
  evidence: internal/core/launch/artefact_test.go:112 — "func TestParseArtefactRefusesRepeatedAndCaseFoldedKeys"
<!-- abcd-review-end receipt=rcp-97519c4308ad -->

## Grounds

- pursued: the release gate is abcd's central promise and it runs only in abcd's own repository today; we expect a declared artefact kind to carry the gate into every managed repository because the gate's inputs are already kind-independent; shown wrong if the two managed repositories still cut by hand after this ships, or if a kind needs a gate input no declaration can supply
