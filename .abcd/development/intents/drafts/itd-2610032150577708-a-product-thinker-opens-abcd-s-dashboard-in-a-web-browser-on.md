---
id: itd-2610032150577708
slug: a-product-thinker-opens-abcd-s-dashboard-in-a-web-browser-on
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610031214560142]
related_adrs: [adr-2610032150581128]
severity: minor
origin: researcher-authored
production_mode: hand-written
supersedes: [itd-139]
---

# The product thinker's dashboard, in a browser on the home network

## Press Release

> A product thinker opens abcd's dashboard in a web browser on their phone, iPad or computer, anywhere on their own local network, and sees in plain words what is being built now, what comes next, and what is waiting on them, without opening a Terminal or knowing any command.

_Proposed by the facilitator on filing (2026-10-03) from the product thinker's request, quoted verbatim: "A web-based dashboard for an abcd-managed project for the product thinker. SOTA on what is required and how to structure it. Must be accessible via phone, ipad, and PC/Mac via a web browser in the local network. Tailscale network later, from anywhere in the future."; to be confirmed at the planning interview._

## Why This Matters

_Facilitator-written._

The board planned today shows the product thinker where things stand, but only in a Terminal or a Claude Code session, which they may not have open, and never on a phone. A page they can open from any device at home shows the same picture where they already are. Being reachable on a network is new for abcd, so the rule for who may open it, and what it may change, is its own record (adr-2610032150581128, proposed).

Typed links: builds on itd-2610031214560142 (the board: the same picture, from the same read of the project; the board's Terminal rendering is not reused); supersedes itd-139 (the readable static site for a team), as the product thinker ruled on 2026-10-03; reach from anywhere is itd-2610032150580455. A state-of-the-art pass on what the dashboard shows and how to serve it safely was started on 2026-10-03 at the product thinker's request and feeds the reviews.

## Decisions

1. 2026-10-03, the product thinker, confirming the routing (itd-84, hand-run): split. This draft is the dashboard on the home network; who may open it and what it may change is a standing rule (adr-2610032150581128, proposed); reach from anywhere through Tailscale is a later draft (itd-2610032150580455).
2. 2026-10-03, the product thinker, asked how the dashboard relates to itd-139 (a readable static site of the record for a team; keep both; the dashboard replaces it; decide later): the dashboard replaces it. The product thinker accepted the cost shown: no published site for a team.
3. 2026-10-03, the product thinker, asked where the dashboard is reachable in its first version (home network first; Tailscale only; both from the start; decide later), with the cost of each shown (a home-network page is only as private as the Wi-Fi; Tailscale needs an app on each device; both is the biggest first version): both from the start. Reach through Tailscale (itd-2610032150580455) is therefore part of the first version, planned and built with this draft rather than later.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_Facilitator-written, for the reviews and the planning interview; the record review of 2026-10-03 added the last two._

- What it shows, in what order, on a phone first (the research's answer, then the product thinker's choice).
- Whether it only shows, or can also do something, such as answering a question waiting on the product thinker.
- How a person is let in (a one-time link or code, a passcode), and how a device stays let in.
- How it is started and stopped, and how the person finds its address on their phone.
- Superseding itd-139 leaves the discipline itd-140 without its demonstration: its rule 3 names itd-139 as the second managed project proving the record site is not abcd-specific. Either that claim is given up, or a new carrier is named.
- Someone starts the dashboard at the computer: the press release's "without opening a Terminal" holds for the person who opens the page, not for the one who starts it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
