---
schema_version: 1
id: "iss-92"
slug: "onboarding-nonstandard-file-placement-interview"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "2026-07-13 B1 dogfood: prepare-this-repo audit of Manuscripts"
found_at: "commands/abcd/prepare-this-repo.md"
remedy: "Waits on ruling M35 (the planning interview it names): extend prepare-this-repo so Phase 2 detects every member of the adopted tiers that abcd canon does not name and classifies it as canon, equivalent (plays a canon role under another name) or foreign; Phase 3 proposes one disposition per file in a single batch the person accepts whole or edits line by line, defaulting to keep in place untouched for foreign and keep plus a pointer from the canon file for equivalent, with fold and move offered only as content-preserving overrides; the answers are recorded in the adopted repository so a re-run does not re-ask; nothing is written until the batch is accepted. Acceptance: the three iss-91 members come out keep in place. Grounds: research at .abcd/development/research/notes/2026-09-30-onboarding-existing-projects-sota.md (Renovate, Copier, nx init, Spec Kit, the harness init, Biome and Terraform all detect, then propose a default the person overrides, then write, and none moves a foreign file on its own authority)."
deferred_after: v0.11.1
deferral_reason: "research at .abcd/development/research/notes/2026-09-30-onboarding-existing-projects-sota.md; ruling owed: M35's second half, the planning interview on what the placement interview asks about each non-standard file (the note proposes detect, classify as canon, equivalent or foreign, and one batch of per-file dispositions defaulting to keep in place)"
---

Maintainer hunch (record only; do not implement, and map against SOTA during design per prefer-sota). Onboarding should follow a strict playbook that identifies every non-standard file in a target repo (e.g. Manuscripts WORKLOG.md, DECISIONS.local.md, SLICE_START), matches each against abcd canon, and proposes where it belongs -- presented interview-style so the maintainer picks from a recommended default they can accept or override (verifier-selects-gates-decide). This generalises the narrower worklocal-nonstandard-members-no-migration finding into the adopt-phase UX. The interview mechanism is a hypothesis, not a decision: the SOTA for convention-onboarding/scaffolding UX must be researched and adversary-filtered for fit before adopting. Detector/acceptance (to firm at design): an adopt run that emits a per-non-standard-file placement proposal with a recommended default the user accepts or overrides.

## Deferral 2026-09-30

Deferred past v0.11.1: research at .abcd/development/research/notes/2026-09-30-onboarding-existing-projects-sota.md; ruling owed: M35's second half, the planning interview on what the placement interview asks about each non-standard file (the note proposes detect, classify as canon, equivalent or foreign, and one batch of per-file dispositions defaulting to keep in place)
