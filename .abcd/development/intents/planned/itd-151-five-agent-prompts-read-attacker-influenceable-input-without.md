---
id: itd-151
slug: five-agent-prompts-read-attacker-influenceable-input-without
spec_id: spc-44
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-278
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

_Empty. Populated by intent-auditor when intent moves to shipped/._
