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

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [models, prompts and measurement SOTA note](../../research/notes/2026-10-23-models-prompts-measurement-sota.md).
Self-detection and recursion depth are internal plumbing and needed no outside
practice; the audit's method did.

- **Reflective prompt rewriting works when it has a metric and traces to
  read.** GEPA beats GRPO by 6 % on average and up to 20 % with up to 35×
  fewer rollouts, and MIPROv2 by over 10 %, by reading execution traces and
  proposing textual edits (Agrawal et al., ICLR 2026). Its precondition is the
  point: without a metric and a trainset, an audit of prose against prose is
  an opinion.
- **Prompt advice is model-generation-specific.** The vendor's current
  prompting reference is organised as model-specific guidance first, then
  shared techniques, then migration notes between generations (Claude
  prompting best practices). "Drift from the baseline" therefore has two
  sources, the prompt and the model the baseline assumed, and a finding must
  say which.
- **Judges over-flag and detailed rubrics make them harsher**
  ([judge-calibration survey](../../research/notes/2026-07-12-judge-calibration-sota.md)),
  so an auditor that returns findings for every agent carries no information;
  a clean verdict per agent must be an admissible result.
- **"Low-risk" edits are behaviour changes** until fixtures say otherwise:
  formatting alone moved accuracy by up to 76 points (Sclar et al., ICLR
  2024). The `--apply-self-audit` path goes through itd-14's bump-and-golden
  run, never around it.
- **The host already ships a prompt audit.** `/doctor prompt-audit` checks
  CLAUDE.md, AGENTS.md and the rules, skills, commands, subagents and output
  styles under `.claude/` and `~/.claude/` for "instructions written for
  older models", references to files or commands that do not exist and files
  that contradict each other, returns findings with proposed edits, and
  changes nothing until asked (Claude Code memory docs). That is the nearest
  shipped analogue of the model-generation drift named above. It audits a
  path only when given one (abcd's `agents/*.md` are outside its default
  scope), keys no finding to a baseline section and gates nothing on
  fixtures; the auditor here adds the baseline keying, the generation stamp
  and the clean-verdict rule, and whether it defers to the host's audit for
  instruction files is a plan-time question.

**Path 2**: a native, host-delegated auditor whose findings are keyed to a
baseline section and carry the model generation the baseline assumed, with a
seam: each finding is recorded as a candidate-plus-feedback pair (prompt
section, textual feedback, citation, generation) in the shape a GEPA-style
optimiser consumes, so a later optimiser reads the audit's record rather than
replacing it. No dependency. The independent fit-challenge runs at plan time.

## Open Questions

- How does abcd detect "this is a self-disembark" — repo name match? `.abcd/config.json["meta"].project_name`?
- (SOTA 2026-10-23) Whether the auditor must itself carry an itd-81 corpus (known-good prompts it should leave alone) before its findings count, or whether the first disembark's findings are the corpus-building run.
- Should the audit gate the lifeboat (refuse to ship if drift exceeds threshold) or just report?
- What's the recursion depth — does the prompt-sota-self-auditor itself get audited? Probably not (turtles stop somewhere).

## Audit Notes

- 2026-10-02 — Declared mutual pair with itd-14 (the technical facilitator's ruling CY2 of 2026-10-02, verbatim: "SAME, DECLARED PAIRS — each direction names the piece it needs"). This record's builds_on itd-15 needs the self-dogfooded SOTA audit's per-agent findings, which this record's sixth criterion writes into the affected prompts' changelogs as `audit:` entries. itd-15's builds_on itd-14 needs the structured per-prompt changelog that replaces the `## Last SOTA Audit:` footer, which the audit's low-risk self-applied footer updates write into. Both edges stand. The record has no field record-lint's edge_cycle rule reads as a declared pair, so the linter still reports the cycle at warn until it does (iss-2610020836255174). Each piece is read from the two records' own scope, criteria and decisions; the ruling names none. Recorded by lane rd2 of autonomous run A.
