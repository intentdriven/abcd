---
schema_version: 1
id: "iss-87"
slug: "adoption-templates-outside-record"
severity: "major"
category: "drift"
source: "agent-finding"
found_during: "2026-07-13 B1 dogfood: prepare-this-repo audit of Manuscripts"
found_at: "commands/abcd/prepare-this-repo.md"
promoted_to: itd-162
resolution: "every adopt-phase asset resolves from the record or the embedded binary; the prepare-commit-msg template is scaffolded by ahoy install --attribution and an onboarding self-containment test refuses a machine-local path"
impact: fix
resolved_by:
  intent: "itd-162"
  spec: "spc-54"
  commit: "0bc31780ea3604cb5212a01082e0301f466d1b9f"
---

prepare-this-repo adopt phase is not self-contained in the abcd record: Phase 3-5 reference templates at a machine-local path outside both the abcd repo and the target (pre-commit-config.yaml, prepare-commit-msg, AGENTS.md, DECISIONS.md, NEXT.md). A fresh clone of abcd-cli onboarding a repo would not have them, so the adoption step silently degrades against loud-staging. Detector: an onboarding self-containment check -- every asset the adopt phase applies resolves from within the abcd record or the binary, never an external machine-local path. Acceptance: the Phase 3/4/5 template references in prepare-this-repo.md.