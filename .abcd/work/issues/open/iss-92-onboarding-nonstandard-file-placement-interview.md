---
schema_version: 1
id: "iss-92"
slug: "onboarding-nonstandard-file-placement-interview"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "2026-07-13 B1 dogfood: prepare-this-repo audit of Manuscripts"
found_at: "commands/abcd/prepare-this-repo.md"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M35: research, then plan, next cycle: SOTA research on how other tools onboard existing projects, then the placement interview's design; iss-91 rides along)."
---

Maintainer hunch (record only; do not implement, and map against SOTA during design per prefer-sota). Onboarding should follow a strict playbook that identifies every non-standard file in a target repo (e.g. Manuscripts WORKLOG.md, DECISIONS.local.md, SLICE_START), matches each against abcd canon, and proposes where it belongs -- presented interview-style so the maintainer picks from a recommended default they can accept or override (verifier-selects-gates-decide). This generalises the narrower worklocal-nonstandard-members-no-migration finding into the adopt-phase UX. The interview mechanism is a hypothesis, not a decision: the SOTA for convention-onboarding/scaffolding UX must be researched and adversary-filtered for fit before adopting. Detector/acceptance (to firm at design): an adopt run that emits a per-non-standard-file placement proposal with a recommended default the user accepts or overrides.