---
id: itd-2610040740108331
slug: a-product-thinker-opens-abcd-s-dashboard
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

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [dashboard-sites SOTA note](../../research/notes/2026-10-23-dashboard-sites-sota.md).

- **Plain HTTP on a LAN address is not a secure context** (MDN): only HTTPS, loopback, `localhost` and `*.localhost` qualify. Restricted to secure contexts: Web Authentication (passkeys), Service Workers, the Async Clipboard, Web Crypto, Notifications, Web Share and Storage APIs. A passkey pairing on the home network is impossible by construction; the one-time code (decision 1) is the right class of mechanism.
- **The local-server field exposes a LAN page only with authentication and HTTPS:** Syncthing ("set a username and a strong password … and check the option to use HTTPS"; binding your computer's LAN address "will NOT restrict access to only devices on your local network!"); code-server (web views need a secure context: loopback, a real certificate, or "a trusted self-signed certificate" via mkcert); Jupyter Server (a token in the URL that sets a cookie; `allow_remote_access` off by default returns 403 when the Host header is not local, its DNS-rebinding guard; `shutdown_no_activity_timeout` is a setting, default `0`).
- **Writes over plain HTTP are weaker than the cross-origin layer suggests:** Go's protection allows any request with neither `Sec-Fetch-Site` nor `Origin`, so a non-browser client on the Wi-Fi holding the cookie can write (net/http). The ADR's "nothing is changed through the page over plain HTTP" is the conservative reading.
- **Chrome's Local Network Access prompt gates public-to-local requests;** a page opened at the computer's own LAN address is unaffected, and a public page probing it is prompted (Chrome developers blog).
- **A `.local` name needs mDNS,** which the Go standard library does not provide: the explicit address the ADR requires is an IP and port, or the computer's hostname where the router resolves it (inference, not sourced).

**Path 3 — deliberate divergence from the field's "HTTPS on the LAN" advice, chosen given the parent's ruling against a publicly logged certificate (decision 23, which a locally minted authority does not touch) and the cost of installing a trust root on each device (this pass's inference, to be put to the product thinker, not a ruling of this draft): reads only over the home network, with a locally minted certificate authority named as the later route to writes.** The idle switch-off becomes a setting beside the status-line settings with a default the product thinker picks; the research supplies a precedent for the setting, not a measured number. The independent fit-challenge runs at plan time.

## Open Questions

- How long the dashboard runs on the home network when nobody looks (1 hour, 8 hours, never).
- Whether notes and "Still right" can be written over the home network at all (the security review says no over plain HTTP).
- (SOTA 2026-10-23) Whether a home-network device is worth letting in for seven days when it can never write; and whether the press release should say "read" rather than "mark and note" for this route.
- (SOTA 2026-10-23) Whether a later writes route by a local certificate authority installed on each device (the mkcert pattern) is acceptable at all, given it asks the person to trust a root on their phone.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
