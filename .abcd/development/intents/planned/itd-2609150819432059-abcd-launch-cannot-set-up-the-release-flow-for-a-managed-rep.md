---
id: itd-2609150819432059
slug: abcd-launch-cannot-set-up-the-release-flow-for-a-managed-rep
spec_id: spc-2609202019026366
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
promoted_from: iss-2609061432214212
origin: extracted-from-record
production_mode: hand-written
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

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the release gate is abcd's central promise and it runs only in abcd's own repository today; we expect a declared artefact kind to carry the gate into every managed repository because the gate's inputs are already kind-independent; shown wrong if the two managed repositories still cut by hand after this ships, or if a kind needs a gate input no declaration can supply
