---
id: itd-2609292107351737
slug: every-development-session-records-what
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
related_issues: [iss-2608301744251874, iss-2608301856299268, iss-2608220150157508]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A development session's token cost is measured, and set against the size class its spec predicted

Typed links: `related_issues` [iss-2608301744251874](../../../work/issues/open/iss-2608301744251874-development-token-metering-is-parked-as.md) (token metering), [iss-2608301856299268](../../../work/issues/open/iss-2608301856299268-a-size-classification-for-specs-and.md) (size classes) and [iss-2608220150157508](../../../work/issues/open/iss-2608220150157508-local-token-usage-accounting-in-the.md) (token accounting in the history store), the three records this draft plans as one.

## Press Release

> Every development session records what it actually cost in tokens, and every spec says in advance which size class it expects to fall in, so a prediction can be checked against what the work became. When a transcript is captured into the history store, abcd keeps the per-message token counts and the model id the transcript itself carries, locally and as metadata only; a report sums them per session, per spec and per repository; and a spec declares a size class before it is built that the report sets beside the measured cost. "We used to guess a flat cost per task and never learned whether the guess was right," said Maya, an autonomous-development practitioner. "Now the size we predicted sits next to what the work actually took, and a wrong class shows up in the next report."

## Why This Matters

Nothing in the record measures what a development act cost. The superseded
itd-29 estimated token cost forward from a flat per-task constant, and the
draft itd-98 names token cost only inside one experiment's report, so an
estimate can be wrong indefinitely without anything noticing. The one tally the
2026-08-30 cost calibration made had to be assembled by hand from one session's
notifications, and it was lost with that session. The parked notes
(`.abcd/development/research/notes/2026-08-30-development-token-metering-gap.md`
and `2026-08-30-spec-size-classification-gap.md`) name the pair: a size class
predicts, metering measures, and neither is worth much alone. The product
thinker ruled on 2026-09-29 to plan metering and size classes as one intent
(ruling J3) and that session token accounting in the history store joins it
(ruling J4).

## What's In Scope

- **Measurement at capture**: when the history store captures a transcript, it
  keeps the per-message usage the transcript carries (input, output, cache read,
  cache write) and the model id, aggregated with a timestamp in the stored
  transcript's metadata. Metadata only, never bodies (brief invariant 15's
  existing carve-out for metadata).
- **A report** that sums the counts per session, per spec and per repository,
  local-only: never committed, never rendered on any site.
- **Size classes** a spec declares before it is built, checked against the
  measured cost in the same report. The notes' finding binds the scheme: depth
  does not track volume, and the roadmap forbids time estimates, so a class is
  not denominated in days.
- **Prices on explicit ask only**: a user-level price table; a report meeting a
  model with no recorded price offers an explicit refresh and never fetches on
  its own (adr-38). Money figures are labelled as API-equivalent estimates.

## What's Out of Scope

- Budget enforcement: refusing to start a run on a budget is itd-29's
  superseded ground and is not revived here.
- Anything that leaves the machine.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the run from the three records and the ruling, unconfirmed: the
planning interview walks each one.

- **Given** a transcript whose messages carry usage fields, **when** the history
  store captures it, **then** the stored metadata carries the summed input,
  output, cache-read and cache-write counts and the model id, and no message
  body is added to the metadata.
- **Given** a transcript that carries no usage fields, **when** it is captured,
  **then** its metadata records the counts as unmeasured rather than zero.
- **Given** captured sessions for one repository, **when** the report runs,
  **then** it sums the counts per session, per spec and per repository, and
  writes nothing.
- **Given** a spec that declares a size class, **when** the report covers the
  sessions that built it, **then** it shows the declared class beside the
  measured cost.
- **Given** a report meeting a model with no recorded price, **when** it runs,
  **then** it shows token counts without a money figure, names the refresh verb,
  and performs no network fetch.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [models, prompts and measurement SOTA note](../../research/notes/2026-10-23-models-prompts-measurement-sota.md);
the gap itself is in the [token-metering](../../research/notes/2026-08-30-development-token-metering-gap.md)
and [size-class](../../research/notes/2026-08-30-spec-size-classification-gap.md) notes.

- **The transcript already carries the counts.** This repository's stream
  fixture (`internal/core/runner/testdata/claude-stream-2.1.302.jsonl`) shows
  per-message `input_tokens`, cache creation split by lifetime, cache read,
  output, thinking tokens and `model`, with a result record carrying per-model
  `costUSD` at `costBasis: "list"` and sub-agent statistics.
- **The host's own dollar figure is local, at list price, and an estimate**:
  `/usage` computes it "from token counts at list price", says it is "an
  estimate", and that for subscribers "the session cost figure isn't relevant
  for billing" (Claude Code cost docs). "API-equivalent estimate" is the
  host's own framing.
- **Two conventions disagree on what "input" is.** OpenTelemetry's
  `gen_ai.usage.input_tokens` "SHOULD include ... cached tokens"; the host's
  exported `type="input"` "excludes tokens read from or written to the prompt
  cache" (GenAI attribute registry; Claude Code monitoring docs). Keep the
  four raw counts and derive totals in the report, naming the convention.
- **Community meters price from one community table.** ccusage reads the
  host's local data, tracks cache tokens separately, prices from LiteLLM's
  `model_prices_and_context_window.json` (a "community maintained list" at a
  stable raw URL) and runs offline on cached prices, at the cost that "you
  will need to upgrade to get updated pricing" (ccusage; LiteLLM docs).
- **A price is four rates and modifiers**: base input, 5-minute cache write
  (1.25×), 1-hour write (2×), cache read (0.1×, 0.025× or 0.05× on the newest
  models), output; batch 50 % off; data residency 1.1× (vendor pricing page).
  The user-level table needs that shape per model, and two cases a flat
  four-rate row cannot hold: the long-context tier (one model pays higher
  prices on a prompt over 100,000 tokens) and the fast-mode premium (the
  stream records `speed`); the table either carries them or the report marks
  such a session as priced at base rates.
- **Relative size estimates are weak and overconfident.** Story points
  correlated strongly with development time in 7 % of 37 projects and
  estimators misestimated in 78 % of 37,440 stories (Tawosi, Moussa and
  Sarro, ESEM 2022); relative estimates were less accurate, slower to produce
  and not developer-independent than hour estimates (Jørgensen and Escott,
  2022). A class is a distribution checked by hit rate, not a point.
- **Task difficulty is predictable from task features, cost is not** (agent
  psychometrics: IRT plus issue, repository, solution and test features
  predicts task-level success; no tokens or dollars; Ge et al., 2026). The
  corpus's own variable, review rounds, is the candidate denomination.

**Path 2**: native metering at capture (four raw counts, model id, sub-agent
split, aggregated into metadata), a report that derives totals and names its
convention, and a user-level price table in LiteLLM's per-model field shape
so a refresh can import that community file on explicit ask and a row can be
hand-edited; size classes denominated in expected review rounds, declared
before the build and reported as a hit rate per class beside measured tokens
and rounds; attribution by capture stamp (the stream's session id; an agent
name exists only as `agent.name` in the host's telemetry export, which the
history store does not read) with the branch as fallback; itd-98 consumes this rather than counting
its own. No dependency; the LiteLLM file is a format, not a library. The
independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Whether the report shows cached tokens inside the input
  total (the OpenTelemetry convention) or beside it (the host's), given that
  the two disagree and both readers exist.
- (SOTA 2026-10-23) Whether the price table's refresh imports LiteLLM's
  community file wholesale, on explicit ask, or only the rows for models the
  store has seen.
- **What a size class measures**: the notes say depth does not track volume; is
  the class about the number of ways the work can be quietly wrong, the review
  rounds it takes, or tokens alone?
- **How a session is attributed to a spec**: the branch, the run log's lane
  record, or a stamp at capture.
- **Where the price table lives** in the user-level home, and its refresh verb.
- **Whether itd-98's comparison report consumes this measurement** rather than
  measuring on its own.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
