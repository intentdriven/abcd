---
id: itd-15
slug: self-dogfooded-sota-audit
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
builds_on: [itd-14]
---

# abcd Audits Its Own Prompts on Every Self-Disembark

## Press Release

> **abcd audits its own agent prompts whenever you disembark abcd-cli.** Pass C automatically runs a SOTA-prompt-audit pass over every `agents/*.md` file in the repo, comparing each against its research baseline in `.abcd/development/research/prompting/agents/<name>.md`. Findings ship as part of the lifeboat, ready to inform the next round of prompt iteration. abcd uses its own infrastructure to maintain its own quality.
>
> "Eat your own dog food," said Kira, open-source maintainer. "Disembarking abcd-cli tells me which of my prompts have drifted from their research basis. The SOTA audit becomes part of the lifeboat itself, not a separate process to remember."

## Why This Matters

abcd's prompt-quality layer D (periodic SOTA audit) is a manual once-per-release ceremony. Easy to forget; easy to skip. The whole point of abcd is to *automate* lessons-extraction; making abcd's own quality assurance manual is hypocritical.

This intent closes the loop. Once abcd is mature enough to reliably self-disembark, the SOTA audit becomes a Pass C step that runs automatically. abcd literally uses its own oracle infrastructure to audit its own agents.

## What's In Scope

- Pass C agent: `prompt-sota-self-auditor` (only runs on disembarks of abcd-cli itself, detected via repo metadata)
- Reads `agents/*.md` + `.abcd/development/research/prompting/agents/<name>.md` pairs
- Generates findings per agent: alignment, drift, recommendations
- Findings included in the disembark report and as a separate lifeboat artefact
- Optional: self-applies low-risk recommendations (typo fixes, footer updates) automatically

## What's Out of Scope

- Generic "audit any project's prompts" — this is specifically abcd self-audit
- Auto-applying high-risk prompt rewrites (always requires human review)
- Cross-version comparison of prompt drift — handled by general SOTA audit

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** abcd-cli itself runs `/abcd:disembark pack <repo> <path>`, **when** Pass C executes, **then** `prompt-sota-self-auditor` activates, reads every pair `(agents/<name>.md, .abcd/development/research/prompting/agents/<name>.md)`, and produces per-agent findings in `audit/prompt-sota-<ts>.{json,md}` covering alignment, drift, and recommendations.
- **Given** any other repo (not abcd-cli) runs `/abcd:disembark pack <repo> <path>`, **when** Pass C executes, **then** `prompt-sota-self-auditor` does NOT activate — the agent is gated by `config.json["meta"].project_name == "abcd"` (or equivalent self-detection) and is silent on third-party repos.
- **Given** the audit produces findings, **when** the lifeboat is written, **then** the findings are present both in the disembark report ("Self-audit summary") AND as a separate lifeboat artefact (`audit/prompt-sota-<ts>.json`) so they survive embark to a downstream copy.
- **Given** the audit's drift exceeds a configured threshold for any agent, **when** the disembark concludes, **then** the disembark report flags the lifeboat as "ship-with-audit-warning" but does NOT refuse to ship — auditing is informational, not gating.
- **Given** an agent's research file is missing (no `.abcd/development/research/prompting/agents/<name>.md` exists), **when** the auditor runs, **then** the agent is reported as "no research baseline" and a separate finding is emitted suggesting the user create the file rather than treating absence as drift.
- **Given** the audit's low-risk recommendations include typo fixes and footer updates, **when** the user runs `/abcd:disembark pack <repo> <path> --apply-self-audit`, **then** those low-risk changes are applied to the working tree as a separate commit BEFORE the lifeboat is packed AND high-risk recommendations remain in the report for human review.

## Open Questions

- How does abcd detect "this is a self-disembark" — repo name match? `.abcd/config.json["meta"].project_name`?
- Should the audit gate the lifeboat (refuse to ship if drift exceeds threshold) or just report?
- What's the recursion depth — does the prompt-sota-self-auditor itself get audited? Probably not (turtles stop somewhere).

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
