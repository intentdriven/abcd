---
id: itd-2610091016262335
slug: the-product-thinker-and-the-technical
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2610052000411162, itd-2610040740122709, itd-2610040740135705, itd-2610091010348892]
related_adrs: [adr-2610091016270711]
---

# The product thinker and facilitator make a project's design decisions on abcdesign.app

## Press Release

> The product thinker and the technical facilitator of an abcd-managed project open it on abcdesign.app and make their design decisions there: the product thinker shapes the brief and the intents, and the facilitator answers the technical decisions. Each can change only what their access to the project's repository allows; everyone else sees the same pages read-only, drawn from the repository.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- How a person proves they may change a project, which change each role may make, and how a change lands in the repository: decided in adr-2610091016270711.
- What the technical facilitator's decisions are on the page: the open questions abcd puts to them (setup, criteria walks, a drain's hand-backs), and how they reach abcd through the same checks a terminal answer passes.
- How this page relates to abcdev.app, which shows a project's landing page and record (itd-2610091010348892): one project page with a read view and a design view, or two sites that link to each other.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Decisions

- 2026-10-09: the product thinker ruled that this intent stands beside itd-2610052000411162 (anyone opens a public brief and submits a proposal) rather than rewriting or superseding it. Answer verbatim: "New intent beside it".
- 2026-10-09: the product thinker ruled that the local dashboard's planned routes for acting (itd-2610040740122709, itd-2610040740135705) are kept beside this hosted one, so a team may decide locally or on abcdesign.app. Answer verbatim: "Keep both routes".
- 2026-10-09: the product thinker ruled on reach and load, for both the local route and abcdesign.app. Answer verbatim: "both (local and abcdesign.app) must allow for public & private repos (via github); building read-only must be rate-limited; building with github login must come from their allowance (e.g. a single cache for anyone logged in via github -- not sure if possible). We don't want public access (e.g. bots) overload the github allowance to serve)." Read as: public and private repositories are both supported, a private one only through the person's own GitHub sign-in; read-only builds without a sign-in are rate-limited; a signed-in build spends that person's own GitHub allowance, never the site's; and anonymous traffic, bots included, can never exhaust the allowance the site serves from.
