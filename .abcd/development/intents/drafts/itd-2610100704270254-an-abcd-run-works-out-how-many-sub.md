---
id: itd-2610100704270254
slug: an-abcd-run-works-out-how-many-sub
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201925079472]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# An abcd run sets how many sub-agents it keeps alive by the models they run on

## Press Release

> An abcd run works out how many sub-agents it may keep alive from the models they run on. The repository's own config sets the budget, eight of the middle model class by default as abcd ships it, and each class above or below it counts double or half: four of the top class, sixteen of the class below, thirty-two of the smallest, and classes mix by share, so four of the top class and eight of the middle use the whole budget. The class names are vendor-neutral, with each vendor's models mapped onto them; Anthropic's are Fable, Opus, Sonnet and Haiku.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [autonomous-loops SOTA note](../../research/notes/2026-10-23-autonomous-loops-sota.md).

- **The vendor limits by model class, and the API's buckets are not shared
  weights.** API rate limits are set "for each model class" and "Each row in
  the following tables has its own RPM, ITPM, and OTPM limits": the classes
  are separate token buckets, so running a top-class agent consumes none of
  another class's capacity, and no cross-class weight follows from the
  bucket sizes. Those sizes also vary by tier: the top class's input and
  output buckets are a quarter of the others' at Start, about 0.3 at Build
  and 0.4 at Scale, with the three lower classes identical at every tier,
  and a bucket measures throughput, not price or sub-agent weight (Claude
  API rate limits). On a subscription the classes share one window and the
  vendor gives only an ordering, "Opus costs several times more per turn than
  Sonnet, and Sonnet more than Haiku", with no published ratio (Claude
  support).
- **The host has a class vocabulary and a ceiling.** A sub-agent's `model` is
  `sonnet`, `opus`, `haiku`, `fable`, a full id or `inherit`, resolved through
  the call, the frontmatter, `CLAUDE_CODE_SUBAGENT_MODEL`, then the main
  conversation's model; 20 running sub-agents is the default cap
  (`CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS`); every sub-agent draws on the same
  usage limits, and cloud sessions share them too (Claude Code docs).
- **The served model is reported after the fact.** The result's `modelUsage` is
  keyed by the model that ran and includes sub-agents (Agent SDK cost
  tracking), which is how an agent left at `host-decides` gets its class.
- **Other vendors cut their classes differently.** A second vendor's models
  page lists a flagship tier above Pro, Flash and Flash-Lite (Gemini models
  page, opened on 2026-10-23); the class count is the vendor's, which is why
  the class table is a seam.
- **Weighted slot pools** (Airflow pools, the make jobserver) and sizing from
  rate-limit headers are in the 2026-10-17 note.

**Path 2**: a weighted slot pool, budget in middle-class units and a weight per
class, mixed by share as ruled, with the double-per-class ladder as the bundled
default in a table the repository may override, since no vendor publishes a
shared-pool ratio (the API's per-class buckets are separate and tier-dependent,
and the subscription ordering is "several times" with no number); an agent at
`host-decides` counts at the top class until the host reports what ran, then is
re-counted from `modelUsage`; the pool sits beside the pacing ceiling with the
lower winning, and the host's concurrent-sub-agent cap is a third ceiling. The
glossary word is "slots", keeping "budget" for the pacing lane's quota check.
The ruling "double per class" is left intact; the evidence says the ratio is
contestable, and the table is how that is carried. No Go dependency. The
independent fit-challenge runs at plan time.

## Open Questions

- How does a run know an agent's class? The routing table names a tier per agent (itd-2609201916056194); an agent left at `host-decides` has no class until the host says which model it ran, so the draft must say what it counts as.
- Does the budget replace the pacing intent's `pace.sub_agents` ceiling (itd-2609201925079472, two by default), or sit beside it with the lower of the two winning?
- Does the machine's own config (`~/.abcd.noindex/config.json`), which the pace settings already read between the repository's and the bundled layer, apply here too, or only the repository's config over abcd's default?
- What is this called? The rebuilt pacing lane adds a budget check that compares an estimate of agent runs with a runner's reported quota (abcd-a5, 2026-10-10), so calling this a sub-agent "budget" would give one word two meanings; the glossary entry is settled at planning.
- (SOTA 2026-10-23) Whether the class weights are a table the repository may override (with the ruled double-per-class ladder as the bundled default), given that no vendor publishes a shared-pool ratio: the API's classes are separate buckets whose sizes vary by tier (the top class a quarter, 0.3 or 0.4 of the others), and the subscription ordering is "several times" with no number.
- (SOTA 2026-10-23) Whether an agent at `host-decides` is counted at the top class until the host's result names the model that ran, and re-counted then.

## Decisions

- 2026-10-10: the product thinker filed this as one intent ("File as proposed"), after the facilitator pointed out that their rough rule, "times 2 minus 1", gives 15 of the class below eight but 4.5, not 3, of the class above. Asked whose sub-agents the budget covers, they chose "Each abcd run": only the agents a build or a drain starts count, not a session's hand-started agents or another session's. Asked how the classes trade, they chose "Double per class": each class up counts double and each class down half, four, eight, sixteen and thirty-two from the top, mixed by share. They added that the setting is the repository's, with abcd's bundled default, and is never kept in an agent's memory or anywhere like it.
- 2026-10-22: the product thinker reversed the answer "Each abcd run" above: the budget is one pool shared across sessions, and one busy session can slow the others (dcn-2610222025116937, a typed reversal recorded on the coordination intent itd-2610170634394993, which also reverses dcn-2610180720379757). The class weights and the repository's setting stand.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
