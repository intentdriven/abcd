---
id: spc-2609202019026366
slug: abcd-launch-cannot-set-up-the-release-flow-for-a-managed-rep
intent: itd-2609150819432059
origin: researcher-authored
production_mode: hand-written
---
# abcd-launch-cannot-set-up-the-release-flow-for-a-managed-rep

## Summary

The design record for itd-2609150819432059, written from the nine decisions the
product thinker settled on 2026-09-20 (the intent's `## Decisions` section).
Every choice below traces to one of them.

## Scope

1. **The artefact declaration** (decision 1, 2, 9): a new `.abcd/config/artefact.json`
   with `kind` (one of `plugin`, `binary`, `application`), `lockstep` (a list
   of repo-relative files held in lockstep with the `version-location.json`
   primary), and `site` (a boolean this spec reads but does not act on). The
   file is validated by one reader in `internal/core/launch` that every launch
   verb and `ahoy` share; an unknown kind or an unreadable lockstep path is a
   named refusal.
2. **The ahoy gap** (decision 3): a managed repository (marker block fired)
   whose artefact file is absent raises a required, resolvable
   `artefact.missing` gap; `ahoy install` prompts for the kind and writes the
   file. A plugin repository (a `.claude-plugin/plugin.json` present) takes
   `kind: plugin` without a prompt so the shipped shape adopts silently.
3. **Lockstep by declaration** (decision 2): `CheckLockstep` reads the primary
   as today and the declared list in place of the pinned plugin-manifest table
   when the kind is not `plugin`; the plugin table stays for `kind: plugin`.
4. **The kind-shaped scaffold** (decisions 4, 5, 6, 7): `launch scaffold`
   lays, for a non-plugin kind, `CHANGELOG.md` with only the empty
   `[Unreleased]` anchor when absent, and `.github/workflows/abcd-release-gate.yml`:
   the verify, derive, changelog-ingest, receipt and tag steps abcd's own
   `release.yml` template carries, plus one named empty job `build` with a
   comment naming the output directory the publish step reads. An existing
   release workflow is never opened; the scaffold report names it as left
   alone and prints the `workflow_call` stanza to add. All scaffolded files
   are drift-checked exactly as the plugin scaffold's are (`--confirm`).
5. **The preview and the cut for a non-plugin kind** (decision 8): with no
   payload include config the bundle is the tree the release tag would
   archive (`git archive`'s view of HEAD) with the record namespace denied by
   the same `DenyNamespaces`; the identity scan runs over it; the report
   states which tree was scanned. The changelog derivation, the version
   derivation, `GuardFindings` and the deferral read are unchanged and run
   for every kind.

## Out of scope

The deferral write path (`iss-2609181223260994`); the build and publish
steps themselves (the repository's, in the empty job or its own workflow);
the site (`itd-2609061543533170` builds on the `site` flag); a repository
that ships several artefacts.

## Approach

Test-first, one lane per numbered piece above, in that order; pieces 1 and 2
land first because everything else reads the file. The scaffold template
for the gate workflow is derived from the plugin template by subtraction,
held to it by the existing scaffold parity test so a fix to the gate reaches
both. The Gropius repository's own release is the acceptance rehearsal: its
next cut runs through `launch --dry-run` and `launch ship` with the gate
enforced by the binary rather than by a session reading the ledger.

## How the criteria are satisfied

Criteria 1 and 9 by the shared reader and the ahoy gap (pieces 1, 2); 2 by
piece 3; 3, 4 and 5 by piece 4; 6 and 7 by the unchanged guard now reachable
(piece 5); 8 by piece 5's archived-tree bundle.

