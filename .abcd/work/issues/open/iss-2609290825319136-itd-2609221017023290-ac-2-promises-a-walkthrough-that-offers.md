---
schema_version: 1
id: "iss-2609290825319136"
slug: "itd-2609221017023290-ac-2-promises-a-walkthrough-that-offers"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-2609221017023290"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/ahoy_credential.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lane drainFresh of autonomous run A, 2026-09-29): Does the record say the CLI takes the credential's home as a --home flag on a second invocation, or should abcd ahoy credential ask on the terminal as the spec and press release say? No recorded decision drops the CLI's ask; the plugin page asks."
---

itd-2609221017023290 ac-2 promises a walkthrough that offers the three homes with the keychain recommended in the prose above the choice; its press release says abcd asks the person once per service where the secret should live, and spc-2609221017544877 scope 2 says the CLI asks on the terminal and the plugin page asks through the host's question tool. Delivered, the CLI's abcd ahoy credential <name> explains and lists the homes with a setup command for each and writes nothing, and the choice is a --home flag on a second invocation (internal/surface/cli/ahoy_credential.go newAhoyCredentialCommand); the CLI never asks. Only the plugin page asks, through the host's question tool (commands/ahoy.md, the credential section). A non-interactive CLI is consistent with every other abcd verb and with the value arriving on stdin only, so this may be the right shape, but the spec and the press release say otherwise and no recorded decision says the CLI's ask was dropped. Either the record says the CLI takes the home as a flag, or the walkthrough asks. Fidelity audit verdict: ac-2 MET_WITH_CONCERNS.
