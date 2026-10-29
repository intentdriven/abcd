---
id: spc-2610230609005313
slug: live-session-setup-records-the-relay
intent: itd-2610091016262335
origin: researcher-authored
production_mode: hand-written
---
# Live-session setup on the cloud in the two steps the intent promises

## Summary

_Draft, filed 2026-10-23 by the remainder close of spc-2610141709430146, on
abcd-ae's ruling as coordinator (dcn-2610230609300445).
Not READY: its design is unsettled below, and it is planned through the usual
walk._

spc-2610141709430146 delivered live brief sessions. Its intent-fidelity
audit (Opus, with a Sonnet second; receipt rcp-a260a0a30612) found one
acceptance criterion not met, and this spec carries it. The criterion, in the
product thinker's words:

> (setup) **Given** Cloud, **when** the technical facilitator sets it up,
> **then** it takes two one-click steps: creating the team's GitHub App from a
> pre-filled manifest, and deploying the relay with Render's Deploy button,
> whose own form asks for the secrets. abcd writes the files and records the
> relay's address itself.

What shipped takes about six steps:
1. create the app;
2. install it;
3. generate a client secret and paste it into Render's form;
4. deploy;
5. re-run `abcd session setup --relay-origin <address>` by hand;
6. add the relay's callback URL on the app's settings page.

abcd writes `relays.json` only from an address the person types. This spec closes that gap, so that the setup is the two
one-click steps and abcd learns the relay's address itself.

The same audit found ac-6 (a guest from another tailnet joining a local
session through Tailscale) inconclusive: never exercised. Its live proof is
iss-2610161444081367, outside this spec.

## Open design questions (to settle at planning)

- **How abcd learns the relay's address.** Options include reading it back
  from Render after the deploy, the relay announcing itself to the repository,
  and a page on the relay that writes it back through the person's session.
  Each has a different trust story, so it is a decision with its own ADR
  check against adr-2610091016270711.
- **Whether installing the app and listing the callback URL can fold into the
  manifest step.** The manifest flow can carry a callback URL only once the
  relay's address is known, which depends on the first question.
- **Whether the client secret can reach Render without being copied by
  hand**, never through abcd's own output (the step 6 review's rule that no
  secret is printed or stored stands).

## Footprint

- packages: internal/surface/session, internal/core/session, internal/surface/cli, commands/session.md, docs/how-to/run-a-brief-session-relay.md, .abcd/development/brief/04-surfaces/37-session.md
- tests: internal/surface/session, internal/core/session, internal/surface/cli

## Steps

_To be listed at planning, once the open design questions are settled._
