---
id: itd-2609081951381895
slug: abcd-ships-an-openai-compatible-api-oracle-adapter-the-first
spec_id: spc-2609221011153746
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609170822093401]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609201916056194, itd-6, itd-2609221009495079]
related_adrs: [adr-2609221009491186]
---

# abcd ships an OpenAI-compatible API adapter, and a provider serves only the models it lists

## Press Release

> **An OpenAI-compatible API adapter reaches a configured provider for the roles and judgements pointed at it, and serves only the models that provider's list allows.**
>
> "I wanted one cheap decision model through OpenRouter, and I wanted to be certain nothing else of mine would ever go through it," said a product thinker configuring the first aggregator. "The adapter takes a base URL and a key name, the provider block lists the models it may serve, a bundled denylist keeps the frontier vendors out whatever I list, and the run record shows the model that actually answered."

## Why This Matters

adr-25 names an API oracle backend and nothing implemented it. The first need arrived with Jev (TypeSafe AI, September 2026), a decision model reachable through OpenRouter, which speaks the OpenAI-compatible protocol; the same adapter serves a local OpenAI-compatible server with no key. The product thinker's condition, ruled on 2026-09-22 (adr-2609221009491186), is that an aggregator serves only listed models under a vendor denylist, so a frontier model the person pays for through the host is never billed or routed through a third party unasked.

## Mechanism

We expect a default-deny list per provider to make the aggregator serve only what the person meant, because a route that must be listed is a route someone wrote down; shown wrong if a frontier model is ever billed through the adapter.

## Scope Conditions

- Holds for providers that speak the OpenAI-compatible chat protocol; a provider with its own protocol is its own adapter. <!-- cond: cond-2609221011152421 -->
- Holds where the credential can be named in the machine's configuration or the environment; a provider that needs an interactive login is out of reach. <!-- cond: cond-2609221011151740 -->

## What's In Scope

- **Configuration**: `oracle.api.<provider>` with `base_url`, `key` (a name, resolved from the machine's configuration or the environment), and `models` (the allowlist); roles and judgement types are pointed at `<provider>/<model>`.
- **The refusal**: a role or judgement configured for a model not on its provider's list is refused when the configuration is read, before any call, naming the list; a listed model matching the bundled vendor denylist (`anthropic/*` at minimum) is refused the same way, and no allowlist entry overrides the denylist.
- **The call**: the same prompt, inputs and output contract the host sub-agent gets, over the protocol; the transcript captured into the same store.
- **The record**: provider, model asked for and model reported, per call, in the run record.
- **Unconfigured**: nothing changes and no network call is attempted (adr-25's default).
- **The one-time setup** (itd-63's explain-then-install mode): when `ahoy` takes a repository over, or is run bare, and no provider is configured, it explains what an aggregator is, what abcd would use it for (decision models and cheap judgements), what works without it (everything, on the host), and offers the walkthrough; on yes it asks where the key should live, the person's choice of three: a setup accessible separately from abcd (an existing OpenRouter or opencode configuration or a named environment variable; abcd stores only the name), abcd-only (the user-level `~/.abcd/` store, owner-only permissions), or the platform keychain (macOS Keychain; the secret service on Linux), with the keychain named as the recommendation in the prose above the choice, never as a marked option; then it writes the provider block with the first allowlist and verifies with one call. The key never enters the harness's settings or the repository.
- **Security review** before it ships: it sends prompts to a network endpoint under a credential.

## What's Out of Scope

- Provider routing (`:nitro`, `:floor`) and any per-request learned router; the adapter asks for the model it is given.
- A model's quality judgement (the tier's, itd-2609170822093401).
- Anything generative through a decision model (itd-2609221009495079).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609221009491186 records the vocabulary rulings it rests on):

1. Default-deny per provider with a vendor denylist above it (adr-2609221009491186).
2. OpenRouter is configuration of this adapter, not code.
3. The first models listed are decision models; frontier models stay on the host.
4. **The setup is a one-time walkthrough at `ahoy`, and the key's home is the person's choice of three** (ruled 2026-09-22): a setup outside abcd, abcd-only on the machine, or the platform keychain, recommended in prose. Basic by default, the adapter as the optional upgrade.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** `oracle.api.openrouter` configured with a base URL, a key name and a model list, **when** a role or judgement pointed at a listed model runs, **then** the call goes over the OpenAI-compatible protocol with the host sub-agent's prompt, inputs and output contract; unconfigured, nothing changes and no call is attempted.
- **Given** a role or judgement configured for a model not on its provider's list, **when** the configuration is read, **then** it is refused before any call, naming the list.
- **Given** a listed model whose prefix matches the vendor denylist, **when** the configuration is read, **then** it is refused the same way, and an allowlist entry does not override it.
- **Given** the key, **when** the adapter reads it, **then** it comes from the machine's configuration or the environment by name, and no credential is written into the repository.
- **Given** a call through the adapter, **when** the run record is read, **then** it names the provider, the model asked for and the model the provider reported.
- **Given** a repository abcd takes over with no provider configured, **when** `ahoy` runs, **then** it explains the aggregator, its use and what works without it, and offers the walkthrough; declining leaves the host as the only route and says so.
- **Given** the walkthrough accepted, **when** it asks where the key lives, **then** it offers the three homes (outside abcd; abcd-only, owner-only; the platform keychain) with the keychain recommended in the prose above the choice, writes the provider block with the first allowlist, verifies with one call, and writes nothing into the harness's settings or the repository.
- **Given** the lane, **when** it ships, **then** a security review of the network path and the credential handling is on its record.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: Jev is reachable only this way and the person's subscription must stay where Opus runs; we expect the listed route to serve only what was meant; shown wrong if a frontier model is ever billed through the adapter
