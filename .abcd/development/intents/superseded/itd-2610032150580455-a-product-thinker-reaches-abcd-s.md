---
id: itd-2610032150580455
slug: a-product-thinker-reaches-abcd-s
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history:
  - { date: 2026-10-04, from: standalone, to: superseded, reason: "the product thinker chose Tailscale first (2026-10-04); reach through Tailscale is the first version's way in, folded into the dashboard intent" }
builds_on: [itd-2610032150577708]
related_adrs: [adr-2610032150581128]
severity: minor
origin: researcher-authored
production_mode: hand-written
kind_at_supersession: standalone
superseded_by: itd-2610032150577708
---

# The dashboard from anywhere, through Tailscale

> **Superseded by itd-2610032150577708** on 2026-10-04: the product thinker chose Tailscale first (2026-10-04); reach through Tailscale is the first version's way in, folded into the dashboard intent

## Press Release

> A product thinker reaches abcd's dashboard from anywhere, away from their home network, through their own private Tailscale network, seeing the same page they see at home, open only to the people they let in.

_Proposed by the facilitator on filing (2026-10-03): the product thinker's request named "Tailscale network later, from anywhere in the future", filed as its own draft at the routing they confirmed. At the dashboard's interview (itd-2610032150577708 decision 3) the product thinker chose both ways in from the start, so this draft is planned with it._

## Why This Matters

_Facilitator-written._

Away from home the product thinker cannot reach a dashboard served on the home network. Their own private Tailscale network can carry it without opening it to the internet, under the same rule for who may open it (adr-2610032150581128).

Typed links: builds on itd-2610032150577708 (the dashboard on the home network).

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_From the state-of-the-art pass and the record review of 2026-10-03 (facilitator-written)._

- Who keeps the dashboard running while the person is away, and for how long.
- Whether being on the person's own private Tailscale network counts as being let in, or a device still pairs.
- An HTTPS name on Tailscale is published in public certificate logs, so the computer's and the private network's names become public.
- The sign-in identity Tailscale adds can be trusted only on a dashboard listening to this computer alone, never while it also listens on the local network.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
