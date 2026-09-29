---
id: itd-1
slug: acceptance-gates
kind: discipline
kind_notes: "Cross-cutting acceptance-criteria gate; applied via the record lint (internal/core/lint) at /abcd:intent plan time and verified by intent-auditor (single-document role) on every shipped intent."
suggested_kind: null
spec_id: null
reclassification_history:
  - { date: 2026-05-07, from: standalone, to: discipline, reason: "Reclassified per the three-intent-kinds change: itd-1 has no user moment of its own (it's a rule that applies to every other epic), so press-release shape was structurally wrong. Discipline shape (## Rule + ## Why) fits." }
severity: critical
---

# Every Intent Has Acceptance Criteria; Every Audit Verifies Them

## Rule

Every standalone or bundle-member intent in `drafts/` and `planned/` carries a `## Acceptance Criteria` section with at least one well-formed Given-When-Then bullet. Every discipline intent in `disciplines/` carries the same. `/abcd:intent plan` refuses to promote an intent without it (hard block via the record lint, `internal/core/lint`). Every shipped intent's `## Audit Notes` section contains per-criterion verdicts (`MET` / `MET_WITH_CONCERNS` / `NOT_MET` / `INCONCLUSIVE`) emitted by the `intent-auditor` agent's single-document role.

## Why

abcd's intent format captures direction (press release or `## Rule`) and scope (in/out lists). Without acceptance criteria it cannot capture **the verifiable bar for "shipped"**. The `intent-auditor` agent is otherwise reduced to interpreting prose, which means drift detection depends on the reviewer's judgement rather than on a pre-committed standard.

Prior art ([PAUL][paul]) treats acceptance criteria as a hard gate: defined before tasks, verified by an Execute/Qualify loop, with multi-state escalation outcomes (`DONE` / `DONE_WITH_CONCERNS` / `NEEDS_CONTEXT` / `BLOCKED`). The full PAUL framework is more than abcd currently needs; the **acceptance-criteria pattern** ([Given-When-Then][bdd-given-when-then]) is the load-bearing piece.

This is a small schema bump with a large quality return. Every intent gets a measurable definition of done; every audit becomes a check, not an interpretation. The discipline is project-agnostic: any project using the abcd intent framework inherits this rule — application projects (idelphiDev, etc.) write their own Given-When-Then criteria for their own user moments; framework projects (abcd itself) write criteria for their own discipline intents.

## What's In Scope

- **`## Acceptance Criteria` section** required in every intent template (standalone, bundle-member, *and* discipline). At least one Given-When-Then bullet. The section header is fixed (parser depends on it).
- **Hard-block validation in `/abcd:intent plan`** — intent cannot transition `drafts/` → `planned/` (or `drafts/` → `disciplines/`) without at least one well-formed acceptance criterion. Lint code: `IL002` (delivered by spc-8 (predecessor store); see `05-internals/06-lint.md`).
- **`intent-auditor` single-document role** — when auditing a shipped intent, the agent emits a per-criterion verdict block into the intent's own `## Audit Notes` section. The writer maintains a single delimited `### itd-1 review <ts>` block (machine-fenced so a repeat review *replaces* it in place — the section never accumulates stale blocks; git history is the prior-review trail):
  ```
  ## Audit Notes

  ### itd-1 review <ts>

  Acceptance:
  - [MET]                 Given <X>, when <Y>, then <Z> — verified by …
  - [MET_WITH_CONCERNS]   Given <A>, when <B>, then <C> — partially observed; concern: …
  - [NOT_MET]             Given <D>, when <E>, then <F> — divergence: …
  - [INCONCLUSIVE]        Given <G>, when <H>, then <J> — could not verify: …

  Overall: MET / MET_WITH_CONCERNS / NOT_MET / INCONCLUSIVE
  ```
  This `## Audit Notes` write is the **verdict of record**; each run also writes a per-run forensic copy at `.abcd/logbook/audit/review-<ts>/report.{json,md}`. **spc-12 (predecessor store) ships the manual reviewer** (`/abcd:intent audit <itd-N>`) plus the `## Audit Notes` / `review-<ts>` writers; **automatic invocation on the `planned → shipped` transition is deferred to the lifecycle-owning spec** (`spc-6`, predecessor store). Until that lands, a shipped intent's `## Audit Notes` is populated only when `/abcd:intent audit` is run by hand.
- **Escalation states** — four states, lifted from PAUL. Binary pass/fail loses information; four states preserve nuance without exploding.
- **Verdict family disjointness** (cross-referenced from [`05-internals/01-agents.md § Verdict-tag protocol`](../../brief/05-internals/01-agents.md#verdict-tag-protocol)). The four criterion verdicts above (`MET` / `MET_WITH_CONCERNS` / `NOT_MET` / `INCONCLUSIVE`) score *promise vs reality on a shipped intent* — they belong to `intent-auditor`'s Role 1 output. They are **deliberately disjoint from review verdicts** (`SHIP` / `NEEDS_WORK` / `MAJOR_RETHINK`) which score *changes/runs* (oracle reviews of plans, implementations, completions; consumed by the native receipt schema validator). The two enums never mix — review verdicts emit on a *change*, criterion verdicts emit on a *promise*. This disjointness was reinforced 2026-05-08 when idea-4's pre-review draft conflated the two families ("NOT_MET on an agent run" — wrong; criterion verdicts apply to intents not agents). Closing-the-loop signals on agents (per Frontier Awareness, idea-4) MUST use canary/golden-test/operator-tagged failure signals, NOT spec-level criterion verdicts.
- **Intent template update** — the intent template the binary scaffolds (`internal/core`) includes the `## Acceptance Criteria` section with one example criterion. The discipline template (separate file) includes the same section.
- **Inheritance into every other spec** — every native spec plan-reviewed under abcd inherits the discipline's gate: the spec must reference the parent intent's acceptance criteria as the verification bar, and `intent-auditor` checks delivered reality against them on shipping.

## What's Out of Scope

- **Full PAUL Execute/Qualify loop** — the per-task verification machinery (coherence checks, diagnostic routing) is out of scope. This discipline takes only the acceptance-criteria pattern.
- **Automated criterion-checking** — the reviewer reads, judges, and emits verdicts. No code-level test generation from Given-When-Then. (Could become an `itd-` candidate later.)
- **Cross-intent acceptance dependencies** — each intent's criteria are independent. No "this intent inherits acceptance from itd-N" beyond the discipline's own application.
- **Quantitative thresholds as a separate system** — criteria can include numbers ("token overhead < 200") but there's no separate metric-tracking system. The reviewer reports observed vs. target.
- **Retroactive backfill of historic intents that have already shipped** — once an intent ships, the criteria are frozen alongside the press release. Pre-discipline historic intents (none yet shipped at time of this draft) won't be retroactively rewritten.

## Acceptance Criteria

> _Yes, this discipline eats its own dog food. The criteria below describe how the discipline itself is checked — by the record lint (`internal/core/lint`) at promotion time and by `intent-auditor`'s single-document role on every shipped intent._

- **Given** a draft intent without an `## Acceptance Criteria` section, **when** the user runs `/abcd:intent plan itd-N`, **then** the command refuses to promote the intent and lists the missing section as the reason.
- **Given** a draft intent with a malformed acceptance section (e.g. no Given-When-Then bullets, or a header but empty body), **when** `/abcd:intent plan` runs, **then** the lint emits a specific error pointing at the malformed line.
- **Given** a discipline-kind intent in `drafts/` without a `## Acceptance Criteria` section, **when** the user attempts to promote it via `/abcd:intent plan --kind discipline`, **then** the same hard-block applies — disciplines are not exempt from their own rule.
- **Given** a shipped intent with three acceptance criteria, **when** `intent-auditor` runs, **then** the resulting Audit Notes contain exactly three per-criterion verdict lines plus an overall status. *(Audit note 2026-05-07: this criterion is verifiable only after the reviewer's implementation lands in a separate spec; until then it is a paper-only gate. The reviewer itself will retroactively run against this discipline when implemented.)*
- **Given** a reviewer verdict of `NOT_MET` on any criterion, **when** the reviewer writes its report, **then** the report includes a "Divergence" sub-section explaining what was delivered vs. what was promised — not just a label.
- **Given** the brief is updated, **when** a contributor reads `04-surfaces/05-intent.md` (intent system), **then** they find the acceptance-pattern requirement and one worked example with all four verdict types.
- **Given** a native spec that traces back to a parent intent, **when** the spec's plan-review runs, **then** the review verifies the spec's acceptance section references (or is structurally compatible with) the parent intent's `## Acceptance Criteria` bullets — drift between the two surfaces is flagged.
- **Given** any native spec that traces back to a brief phase or directly to this discipline, **when** the spec's plan-review runs, **then** the review verifies the spec carries its own `## Acceptance Criteria` block — the discipline applies to plumbing specs too, not just intent-derived ones.

## Open Questions

- **Lint strictness on existing drafts** — pre-discipline drafts (already grandfathered above) may have malformed AC sections rather than missing ones. Should `/abcd:intent plan` warn-and-prompt or hard-block in that case? (Recommend: hard-block. The whole point is to force the discipline. Backfill happens at the user's own pace via `/abcd:intent refine`.)
- **Verdict-rollup logic** — lives in [`brief/04-surfaces/05-intent.md § 7`](../../brief/04-surfaces/05-intent.md#7-the-intent-auditor-agent-three-roles-three-verbs) (SSOT). Don't duplicate here.
- **How many criteria are too many?** — 1 minimum is enforced; 5+ may indicate the intent is too big and should split. Document as a soft guideline, not a lint.
- **Criterion language** — strict Given-When-Then only, or allow plain bullet criteria as a fallback? (Recommend: strict. Plain bullets are what scope sections already provide; criteria need to be observable, not aspirational.)
- **Discipline-specific AC shape** — disciplines describe rules, not deliverables. Should their `## Acceptance Criteria` describe how the *rule* is verified (e.g., "Given an intent without AC, when plan runs, then it fails") rather than what the deliverable is? (Recommend: yes, explicit in the discipline template; this discipline's own AC above is a worked example.)

## Audit Notes

_Empty. Populated by intent-auditor's single-document role when this discipline is first audited. Note: disciplines never "ship" in the standalone-intent sense — they are audited continuously via the rule-applies-to-every-spec semantics rather than via a planned→shipped transition. The reviewer's findings here record any spec that violated this discipline (e.g., merged without an AC section)._

## References

[paul]: https://github.com/ChristopherKahler/paul "PAUL — Plan-Apply-Unify Loop, project orchestration framework for Claude Code (Kahler)"
[bdd-given-when-then]: https://martinfowler.com/bliki/GivenWhenThen.html "Given-When-Then (Fowler) — BDD acceptance-criteria pattern"

- See [`research/related-work.md`](../../research/related-work.md) for the full PAUL / BDD comparison that informed this discipline.
- The discipline is one of two introduced alongside [itd-5](itd-5-prompt-quality-additions.md) (prompt-quality additions). Both lived as `kind: standalone` intents and were reclassified to `kind: discipline` when the three-kinds model was introduced.
