# Jev and model routing: the research pass behind adr-2609221009491186 and itd-2609221009495079

Recorded 2026-09-22 from an independent research pass (web only, primary
sources where they exist), run before the product thinker's rulings of the
same night. It is the SOTA record for the API adapter (itd-2609081951381895),
the decision adapter (itd-2609221009495079), the model tier's escalation
rule (itd-2609170822093401) and the retirement of itd-17.

## What Jev is

Jev is TypeSafe AI's "System One" model (early access 2026-09-15; on
OpenRouter as `typesafe/jev-1.13`, alias `typesafe/jev-latest`, at $0.042 per
million input tokens, 32K context, text only). It is neither a text model nor
a router: it takes a state and a typed question and returns a value with a
calibrated probability through three primitives, choose one of N, score
against a rubric, yes or no, in under a second. It cannot generate text and it
cannot abstain; its founder conceded publicly that it "can still emit a
completely wrong valid value". Weights are closed, hosting is single-vendor
and waitlisted. Independent evidence is thin: one forty-case routing test (all
correct, middle-tier confidence 0.57 to 0.67); the vendor's speed and cost
figures are measured by the vendor on its own workflows. The name resolves to
this one thing; the product thinker's "from TypeScript" was a near-homophone
of TypeSafe.

## Routing, 2024 to 2026, in one table

| System | Decides | Signal | Evidence | Failure mode |
| --- | --- | --- | --- | --- |
| RouteLLM (Berkeley, 2024) | strong or weak model, before inference | preference data, learned classifiers | paper: over 2x cost cut at held quality | one pair of models |
| Arch-Router (2025) | which user-defined policy a query matches | a 1.5B classifier | vendor-authored paper | intent, not difficulty |
| OpenRouter Auto (2026) | model per request in five cost tiers | task classifier ranked by market spend | vendor-run | popularity is not correctness; not reproducible |
| Provider routing, LiteLLM | which provider or deployment | price, throughput, uptime | docs | not a quality decision |
| Not Diamond, Martian, Unify | model per request | learned on eval data | vendor-quoted; RouterArena ranks Not Diamond twelfth for over-selecting expensive models | opacity |
| Degenerate convergence (2026) | analysis | | routers collapse to the priciest model as budget rises | objective mismatch |
| Rerouting LLM routers (2025) | attack | | query-independent gadgets force upgrades | an attackable control plane |
| Cascades (TMLR 2026; RouteNLP) | escalate after a cheap attempt | confidence or a judge | 40 to 85 per cent cost cut at 96 to 100 per cent quality on structured tasks | needs a calibrated signal |

Vendor guidance converges: classify then dispatch, "build the right system,
not the most sophisticated" (Anthropic); the host's own mechanism is static
per role (Claude Code's `model:` frontmatter and the plan-with-opus,
execute-with-sonnet split); prototype on the most capable model and downgrade
where evaluations hold (OpenAI).

## The mapping the rulings rest on

- **Model per role**: not routing; a configuration table. The role name is
  the classifier. Learned routers lose on opacity and on the measured drift
  toward expensive models. Kept as the tier.
- **Escalation inside a lane**: a cascade whose signal abcd already owns, the
  failed fix round; a rule, recorded as a fact. No predicted difficulty beats
  ground truth.
- **The closed-option judgements**: the one place a decision model fits; an
  adapter behind the host's own interface, measured in shadow as a lab before
  it decides anything. Constraints: 32K (retrieve first), no abstention (never
  for a judgement whose right answer is often unknown), single vendor.
- **What to build next**: nowhere in the literature a routing problem; a
  scoring problem; the pick stays computed. A rubric score may become one
  component later.
- **Corpus consistency**: batch classification; lint first, typed verdicts on
  the residue, the strong model on what is flagged.

## Not adopted

A learned per-request router for lane models; OpenRouter Auto as a role's
model; provider routing as a quality lever; Jev as a default judge.

## Where no evidence was found

An independent, methodology-disclosed benchmark of Jev's calibration; third-
party measurements of the commercial routers beyond RouterArena; any framing
of backlog selection as routing; a study of cost-escalation cascades inside
multi-round coding-agent fix loops.
