---
id: itd-151
shipped_in: v0.6.8
slug: five-agent-prompts-read-attacker-influenceable-input-without
spec_id: spc-44
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-278
impact: additive
---

# Five agent prompts read attacker-influenceable input without the itd-5 contract (ruthless-reviewer, security-reviewer, docs-currency-reviewer, intent-auditor, sota-researcher) and agents/ sits outside both lint roots, so no detector exists for the class; the PQ linter (agents/README.md) is the missing detector

## Press Release

> _Seeded by promotion from iss-278. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-278`: Five agent prompts read attacker-influenceable input without the itd-5 contract (ruthless-reviewer, security-reviewer, docs-currency-reviewer, intent-auditor, sota-researcher) and agents/ sits outside both lint roots, so no detector exists for the class; the PQ linter (agents/README.md) is the missing detector. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** the record-lint configuration, **when** it runs, **then** it walks the `agents/` tree, which previously sat outside every lint root.
- **Given** an agent prompt under `agents/` that reads attacker-influenceable input but lacks its itd-5 trust-contract frontmatter, **when** record-lint evaluates it, **then** the gate fails and names the missing frontmatter.
- **Given** an untrusted-input agent that declares the itd-5 frontmatter but ships no injection-canary fixture, **when** the detector runs, **then** the gate fails and names the missing canary fixture.
- **Given** an agent added or changed in a diff without a matching per-agent changelog entry, **when** record-lint runs over that diff, **then** the gate fails and names the missing changelog entry.
- **Given** an agent that carries its itd-5 frontmatter, an injection-canary fixture, and a per-agent changelog entry, **when** record-lint runs, **then** the gate passes with no finding raised against that agent.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-31373784df44 -->
Fidelity review — receipt rcp-31373784df44 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:56d0cf51f967f0385fb3e1e57fb356870004de978337dd7ab3fbee83eadd1d49
Input attestations: diff:328a6755^1..328a6755 (PR #555), judged against the tree at bad1c73e@-;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the agent_contract rule is enabled in the committed record-lint config and runs once outside the per-root loop, enumerating every agents/*.md itself
  evidence: internal/core/lint/lint.go:394 — "if acCfg, ok := cfg.Rules[ruleAgentContract]; ok && acCfg.Enabled {"
  evidence: internal/core/lint/agentcontract.go:104 — "entries, err := os.ReadDir(dirAbs)"
  evidence: .abcd/record-lint.json:288 — ""agent_contract": {"
  evidence: internal/core/lint/agentcontract_test.go:47 — "func TestAgentContractWalksAgentsTree"
- ac-2 — MET: an untrusted-input prompt missing capability_scope, task_classes or designed_for yields a finding naming the missing field, at blocker severity
  evidence: internal/core/lint/agentcontract.go:195 — "agent prompt reads untrusted input but declares no 'capability_scope'"
  evidence: internal/core/lint/agentcontract.go:203 — "is missing 'capability_scope.designed_for'"
  evidence: internal/core/lint/agentcontract_test.go:87 — "func TestAgentContractMissingTrustFields"
- ac-3 — MET: the rule Lstats agents/< name>/fixtures/injection-canary.json for every untrusted-input agent and names the expected path when absent, non-regular or empty
  evidence: internal/core/lint/agentcontract.go:213 — "canaryRel := filepath.Join(dir, p.name, "fixtures", agentCanaryFixture)"
  evidence: internal/core/lint/agentcontract.go:217 — "agent prompt reads untrusted input but ships no "+agentCanaryFixture+"
  evidence: internal/core/lint/agentcontract_test.go:161 — "func TestAgentContractMissingCanary"
- ac-4 — MET_WITH_CONCERNS: over an armed range a changed prompt without a prompt_version bump is refused, and a bumped prompt without a CHANGELOG entry for that version is refused; concern: the entry check is keyed on prompt_version rather than on diff membership, so an edit that neither bumps nor is linted under an armed range (only CI arms -agent-diff; `make record-lint` does not) raises no finding
  evidence: internal/core/lint/agentcontract.go:347 — "changed in this diff without a 'prompt_version' bump, so no new"
  evidence: internal/core/lint/agentcontract.go:232 — "The TREE-shaped part runs always: every prompt's current prompt_version must"
  evidence: internal/core/lint/agentcontract_test.go:177 — "func TestAgentContractChangelogEntryRequiredOverDiff"
  evidence: internal/core/lint/agentcontract_test.go:230 — "func TestAgentContractUnbumpedChangeOverDiff"
  evidence: .github/workflows/ci.yml:294 — "`-agent-diff` additionally arms agent_contract's unbumped-edit check"
- ac-5 — MET: a complete fixture lints clean in the unit test, and the real tree — sixteen prompts, each with frontmatter and a canary fixture — passes `go run ./cmd/record-lint` at BASE with exit 0
  evidence: internal/core/lint/agentcontract_test.go:66 — "func TestAgentContractCompleteAgentPasses"
  evidence: agents/intent-auditor.md:8 — "prompt_version: 0.3.1"
  evidence: agents/intent-auditor.md:9 — "reads_untrusted_input: true"
  evidence: agents/security-reviewer/fixtures/injection-canary.json:1 — "injection-canary.json present for every prompt under agents/"

Gap audit:
- honoured:
  - agents/ is under a lint gate via a dedicated rule, not by adding it to cfg.Roots
    evidence: internal/core/lint/lint.go:391 — "agent_contract walks the agent-prompt tree (agents/ — outside cfg.Roots"
  - the five named prompts now carry the itd-5 frontmatter and a canary fixture
    evidence: agents/intent-auditor.md:10 — "capability_scope:"
    evidence: agents/sota-researcher/fixtures/injection-canary.json:1 — "fixture present"
  - the diff range is caller-supplied and validated before it reaches git
    evidence: internal/core/lint/agentcontract.go:318 — "if !agentDiffRangeRe.MatchString(cfg.DiffRange) {"
    evidence: internal/core/lint/agentcontract_test.go:257 — "func TestAgentContractRefusesHostileDiffRange"
- diverged:
  - the per-agent changelog check was promised as diff-driven ('added or changed in a diff'); delivered as a tree-shaped entry-per-prompt_version check plus a diff-armed unbumped-edit check, so the diff half fires only where CI arms a range
    evidence: internal/core/lint/agentcontract.go:238 — "The DIFF-shaped part runs only when a range is armed"
    evidence: cmd/record-lint/main.go:27 — "agentDiff := flag.String("agent-diff""
- missing: (none)