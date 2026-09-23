---
id: itd-162
shipped_in: v0.6.8
slug: adoption-templates-outside-record
spec_id: spc-54
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-87]
impact: fix
---

# prepare-this-repo adopt phase is not self-contained in the abcd record: Phase 3-5 reference templates at a machine-local path outside both the abcd repo and the target (pre-commit-config.yaml, prepare-commit-msg, AGENTS.md, DECISIONS.md, NEXT.md). A fresh clone of abcd-cli onboarding a repo would not have them, so the adoption step silently degrades against loud-staging. Detector: an onboarding self-containment check -- every asset the adopt phase applies resolves from within the abcd record or the binary, never an external machine-local path. Acceptance: the Phase 3/4/5 template references in prepare-this-repo.md.

## Press Release

> _Seeded by promotion from iss-87. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-87`: prepare-this-repo adopt phase is not self-contained in the abcd record: Phase 3-5 reference templates at a machine-local path outside both the abcd repo and the target (pre-commit-config.yaml, prepare-commit-msg, AGENTS.md, DECISIONS.md, NEXT.md). A fresh clone of abcd-cli onboarding a repo would not have them, so the adoption step silently degrades against loud-staging. Detector: an onboarding self-containment check -- every asset the adopt phase applies resolves from within the abcd record or the binary, never an external machine-local path. Acceptance: the Phase 3/4/5 template references in prepare-this-repo.md.. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a machine without the machine-local `~/ABCDevelopment/.agents/templates/` directory, **when** prepare-this-repo's adopt phase runs, **then** it applies the templates from the committed record or the embedded binary and the adoption completes rather than silently degrading.
- **Given** the adopt phase's assets (the pre-commit config, the prepare-commit-msg hook, AGENTS.md, DECISIONS.md and NEXT.md templates), **when** they are resolved, **then** every one resolves from within the abcd record or the binary and never from an external machine-local path.
- **Given** the onboarding self-containment check, **when** it scans prepare-this-repo, **then** no reference to the `~` machine-local templates path remains.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-40d49180fe07 -->
Fidelity review — receipt rcp-40d49180fe07 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:ed2b2065cd66537debaeb9aa8a2e34c1a61a83ef21a8766532e121c6358a4a10
Input attestations: diff:328a6755^1..328a6755 (PR #555) -- internal/core/ahoy and commands/prepare-this-repo.md, judged against the tree at 0ab2ad02@sha256:32017707383f66265268a397c56bc0bdc8e4005486a4ebfc6c2bd0c1e731138c;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: Phase 3 steps 5 and 6 scaffold every hook through the binary (ahoy install, ahoy install --attribution) whose templates are go:embed'ed, and TestAttributionHookScaffoldsFromTheBinary installs under an empty hermetic HOME and asserts the hook arrives; concern: the asset the old step offered was a secrets + absolute-path pre-commit framework config, and the delivery substitutes abcd's private name guard for it rather than embedding that gate, so the adoption completes with a different gate than the one promised (iss-2609231016278107)
  evidence: commands/prepare-this-repo.md:176 — ""${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install"
  evidence: internal/core/ahoy/attribution_hook.go:39 — "//go:embed defaults/prepare-commit-msg"
  evidence: internal/core/ahoy/banlist_scaffold.go:65 — "//go:embed defaults/pre-commit"
  evidence: internal/core/ahoy/attribution_hook_test.go:25 — "func TestAttributionHookScaffoldsFromTheBinary"
- ac-2 — MET_WITH_CONCERNS: the pre-commit, pre-merge-commit and prepare-commit-msg hooks resolve from the binary's embedded defaults, AGENTS.md/DECISIONS.md/NEXT.md are described inline in the record with no template file, and the self-containment scan finds no machine-local path; concern: the 'pre-commit config' the criterion names as an asset no longer exists as one — it was replaced, not relocated (iss-2609231016278107)
  evidence: internal/core/ahoy/attribution_hook.go:39 — "//go:embed defaults/prepare-commit-msg"
  evidence: internal/core/ahoy/banlist_scaffold.go:68 — "//go:embed defaults/pre-merge-commit"
  evidence: commands/prepare-this-repo.md:110 — "1. **Three tiers.** Create `.abcd/development/`"
  evidence: internal/core/ahoy/onboarding_test.go:44 — "func TestOnboardingIsSelfContained"
- ac-3 — MET: TestOnboardingIsSelfContained scans commands/prepare-this-repo.md for ~/, $HOME/ and .agents/templates and fails on any hit; TestMachineLocalRefsCatchesAReintroduction proves the scan fires on the three reintroduced shapes and stays quiet on the binary-resolved form; both pass at 0ab2ad02
  evidence: internal/core/ahoy/onboarding_test.go:26 — "func machineLocalRefs(data []byte) []int"
  evidence: internal/core/ahoy/onboarding_test.go:44 — "func TestOnboardingIsSelfContained"
  evidence: internal/core/ahoy/onboarding_test.go:60 — "func TestMachineLocalRefsCatchesAReintroduction"

Gap audit:
- honoured:
  - no adopt-phase asset resolves from a machine-local path; the prepare-commit-msg hook is net-new and embedded
    evidence: internal/core/ahoy/attribution_hook.go:39 — "//go:embed defaults/prepare-commit-msg"
    evidence: internal/core/ahoy/onboarding_test.go:44 — "func TestOnboardingIsSelfContained"
  - adoption on a machine with an empty HOME completes rather than silently degrading
    evidence: internal/core/ahoy/attribution_hook_test.go:25 — "func TestAttributionHookScaffoldsFromTheBinary"
- diverged:
  - the pre-commit config (secrets + absolute-path gate) resolves from the record or the binary — delivered as a substitution: step 5 scaffolds the private name guard instead, and the embedded hook carries no secrets or absolute-path gate
    evidence: commands/prepare-this-repo.md:171 — "5. **Commit gates.** Scaffold them from the binary"
    evidence: internal/core/ahoy/defaults/pre-commit:159 — "# --- itd-74 private name guard"
- missing: (none)