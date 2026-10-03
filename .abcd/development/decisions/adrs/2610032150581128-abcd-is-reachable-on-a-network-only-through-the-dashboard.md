---
id: adr-2610032150581128
slug: abcd-is-reachable-on-a-network-only-through-the-dashboard
status: proposed
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610032150577708, itd-2610032150580455]
related_rfcs: []
related_adrs: []
---

# ADR-2610032150581128: abcd is reachable on a network only through the dashboard, only while the person runs it, and only for people the person lets in

## Context

The product thinker asked on 2026-10-03 for a web dashboard of an abcd-managed project, reachable from a phone, an iPad and a computer on the home or office network, and later from anywhere through Tailscale (itd-2610032150577708, itd-2610032150580455). Until now abcd never listens on a network: it makes outbound calls only when asked (adr-38, brief invariant 7) and keeps its files in folders the person handed it. A page anyone on the same network can open is a new trust boundary: a guest on the home Wi-Fi, a device that resolves a hostile page's name to the person's computer, or a link copied into the wrong chat can all reach it. The product thinker confirmed on 2026-10-03 that this rule is its own record, apart from the feature.

## Decision

Proposed by the facilitator; the details are put to the product thinker at the dashboard's planning interview, informed by the state-of-the-art pass:

- abcd listens on a network only while the person runs the dashboard on purpose, never as a background service that starts by itself, and stops when they stop it.
- Only the people the person lets in can open it: access is granted by something the person hands over (for example a one-time pairing link or code), never by being on the same network alone.
- What the page may change, if anything, is decided at the interview; until it is, the page shows and changes nothing.
- Nothing the page serves leaves the person's own devices: no third-party scripts, fonts or trackers.

## Alternatives Considered

- Never listen, and publish a static site instead (the retired itd-139's route): no new boundary, but nothing live and nothing private to the person's own devices.
- Listen on the network with no access control, as many local development servers do: simplest, and the commonest cause of exposed local dashboards.
- Listen only on this computer and reach it from other devices through a tunnel: safe, but a phone on the home network cannot open it without extra software, which the request rules out until Tailscale.
- The proposed rule above: reachable on the network, on purpose, and only for people let in.

## Consequences

- abcd gains its first listening surface, so a security review covers every change to it, and the spec states its bind address, its access check and its limits.
- Brief invariant 7's network trichotomy gains an inbound clause when this is accepted; the brief's invariants are edited in the change that accepts it.
- Tailscale (itd-2610032150580455) is a second way in under the same rule: being on the person's private network is not, alone, being let in, unless the interview rules it so.
