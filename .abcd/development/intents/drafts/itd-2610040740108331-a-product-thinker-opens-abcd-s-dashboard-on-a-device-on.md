---
id: itd-2610040740108331
slug: a-product-thinker-opens-abcd-s-dashboard-on-a-device-on
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610032150577708]
related_adrs: [adr-2610032150581128]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# The dashboard on the home network, without Tailscale

## Press Release

> A product thinker opens abcd's dashboard on a device on their home network without Tailscale, after scanning a one-time code shown at the computer; the device stays let in for seven days, and the dashboard switches itself off when nobody looks.

_Filed by the facilitator on 2026-10-04 from the dashboard interview (itd-2610032150577708 decisions 4, 5 and 9): the product thinker chose Tailscale first, and the home network later._

## Why This Matters

_Facilitator-written._

Not every device the product thinker uses is on Tailscale, and a guest device in the household may never be. Reaching the dashboard on the home network alone is what the request first named. It is plain HTTP on the Wi-Fi, so it carries the security review's local-network requirements in adr-2610032150581128: an explicit address, a one-time code, a capped device lifetime, an idle switch-off, and a start line saying the page is only as private as the Wi-Fi.

Typed links: builds on itd-2610032150577708.

## Decisions

1. 2026-10-03, the product thinker (on itd-2610032150577708, decision 4): a new device is let in by scanning a one-time code shown at the computer.
2. 2026-10-04, the product thinker (decision 5 there): a device stays let in for seven days, revocable from the computer.
3. 2026-10-04, the product thinker (decision 9 there): Tailscale first; this draft is later. When the dashboard switches itself off when idle on the home network was asked and not answered (the answer reverted the reach instead); it is open here.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- How long the dashboard runs on the home network when nobody looks (1 hour, 8 hours, never).
- Whether notes and "Still right" can be written over the home network at all (the security review says no over plain HTTP).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
