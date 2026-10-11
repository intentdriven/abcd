---
id: itd-2609081951381895
slug: abcd-ships-an-openai-compatible-api
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
related_adrs: [adr-2609221009491186, adr-2609300107513982]
---

# abcd ships an OpenAI-compatible API adapter, and a provider serves only the models it lists

## Press Release

> **An OpenAI-compatible API adapter reaches a configured provider for the roles and judgements pointed at it, and serves only the models that provider's list allows.**
>
> "I wanted one cheap decision model through OpenRouter, and I wanted to be certain nothing else of mine would ever go through it," said a product thinker configuring the first aggregator. "The adapter takes a base URL and a key name, the provider block lists the models it may serve and nothing I left off it goes through, and the run record shows the model that actually answered."

## Why This Matters

adr-25 names an API oracle backend and nothing implemented it. The first need arrived with Jev (TypeSafe AI, September 2026), a decision model reachable through OpenRouter, which speaks the OpenAI-compatible protocol; the same adapter serves a local OpenAI-compatible server with no key. The product thinker's condition, ruled on 2026-09-22 (adr-2609221009491186), is that an aggregator serves only listed models, so a frontier model the person pays for through the host is never billed or routed through a third party unasked; the vendor denylist that ruling put above the list is retired by ruling H9 of 2026-09-29 (adr-2609300107513982), and the list alone decides.

## Mechanism

We expect a default-deny list per provider to make the aggregator serve only what the person meant, because a route that must be listed is a route someone wrote down; shown wrong if a frontier model is ever billed through the adapter.

## Scope Conditions

- Holds for providers that speak the OpenAI-compatible chat protocol; a provider with its own protocol is its own adapter. <!-- cond: cond-2609221011152421 -->
- Holds where the credential can be named in the machine's configuration or the environment; a provider that needs an interactive login is out of reach. <!-- cond: cond-2609221011151740 -->

## What's In Scope

- **Configuration**: `oracle.api.<provider>` with `base_url`, `key` (a name, resolved from the machine's configuration or the environment), and `models` (the allowlist); roles and judgement types are pointed at `<provider>/<model>`.
- **The refusal**: a role or judgement configured for a model not on its provider's list is refused when the configuration is read, before any call, naming the list; the list alone decides, so abcd bundles no vendor denylist, and a listed model an `oracle.denylist` entry the configuration writes matches is refused the same way, naming the entry.
- **The call**: the same prompt, inputs and output contract the host sub-agent gets, over the protocol; the transcript captured into the same store.
- **The record**: provider, model asked for and model reported, per call, in the run record.
- **Unconfigured**: nothing changes and no network call is attempted (adr-25's default).
- **The one-time setup** through the credential store (itd-2609221017023290, adr-2609221017021499), in itd-63's explain-then-install mode: when `ahoy` takes a repository over, or is run bare, and no provider is configured, it explains what an aggregator is, what abcd would use it for (decision models and cheap judgements), what works without it (everything, on the host), and offers the walkthrough; on yes it asks where the key should live, the person's choice of three: a setup accessible separately from abcd (an existing OpenRouter or opencode configuration or a named environment variable; abcd stores only the name), abcd-only (the user-level `~/.abcd/` store, owner-only permissions), or the platform keychain (macOS Keychain; the secret service on Linux), with the keychain named as the recommendation in the prose above the choice, never as a marked option; then it writes the provider block with the first allowlist and verifies with one call. The key never enters the harness's settings or the repository.
- **Security review** before it ships: it sends prompts to a network endpoint under a credential.

## What's Out of Scope

- Provider routing (`:nitro`, `:floor`) and any per-request learned router; the adapter asks for the model it is given.
- A model's quality judgement (the tier's, itd-2609170822093401).
- Anything generative through a decision model (itd-2609221009495079).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609221009491186 records the vocabulary rulings it rests on):

1. Default-deny per provider with a vendor denylist above it (adr-2609221009491186). **Amendment, 2026-09-30:** ruling H9 of 2026-09-29 retires the bundled vendor denylist; the provider's allowlist alone decides, and `oracle.denylist` stays as an optional extension the configuration writes (adr-2609300107513982, which supersedes adr-2609221009491186 and changes its decision 2 only).
2. OpenRouter is configuration of this adapter, not code.
3. The first models listed are decision models; frontier models stay on the host.
4. **The setup is a one-time walkthrough at `ahoy`, and the key's home is the person's choice of three** (ruled 2026-09-22): a setup outside abcd, abcd-only on the machine, or the platform keychain, recommended in prose. Basic by default, the adapter as the optional upgrade.

Taken in the implementing lane (autonomous run A, 2026-09-26), within the rulings above:

5. **The external and keychain homes are deferred to the credential store (2026-09-26).** The key is read through the interim credential source, `internal/core/credential` (`~/.abcd/credentials.json`, mode 0600, owner-only, no symlink), which reads one home. This lane builds the write for that home alone, the abcd-only one. The environment-variable-or-external-tool home and the platform keychain (the `security` command on macOS, the secret service on Linux) are built by itd-2609221017023290, the credential store, which replaces the source's backing and not its interface; until it lands, `abcd ahoy connect` refuses either home naming itd-2609221017023290, before any call and any write, and the explanation names all three homes with the keychain recommended in prose. A fourth answer, `none`, sets up a local server that takes no key.
6. **The walkthrough is a verb the person runs, with the key on stdin (2026-09-26).** `ahoy` explains the adapter as an optional gap and on `abcd ahoy --providers`, and the setup is `abcd ahoy connect <provider>`, rather than a question the install pass asks: the install prompter echoes every answer into its transcript, a host's question tool would put the key in an agent's context, a flag would leave it in the process listing and the shell history, and a terminal would echo it as it is typed. Declining is not running it, and changes nothing.
7. **Verify, then write (2026-09-26).** The verification call is made with the key in memory before anything is written, and a failed verification writes nothing, so a wrong key or an unlisted model never leaves a half-configured provider behind.
8. **A provider block sits on the machine alone (2026-09-26), and so does a route to a provider that holds a key (amended 2026-09-29).** `oracle.api.<provider>` names the address a key is sent to, so a repository's `.abcd/config.json` declaring it is refused; a checkout must never be able to aim the person's key at a server of its choosing. Denylist extensions (`oracle.denylist`) may sit in either layer. A route (`oracle.roles.<agent>`, `oracle.judgements.<type>`) to a provider that holds a key sits on the machine alone: only a route the person set up on their own machine may spend their paid key, so a repository's route to such a provider is skipped when the configuration is read, with a diagnostic naming the route and `~/.abcd/config.json` as where to set it, and the rest of the configuration loads, the machine's own route to that name applying in its place (the technical facilitator's ruling CD2 of 2026-09-29 chose the skip over refusing the whole configuration; a route the denylist matches is still refused). A provider holds a key when its block names a `key` credential; that is judged from the block alone, never by reading the credential store, so no secret is read to decide it and a block that names a key counts as keyed before its key is stored. A repository's route to a provider whose block names no key (a local server) is admitted and wins per name, a `--route` the person types is unaffected, and a route naming a provider this machine has not configured stays on the host with a diagnostic. **Amendment, 2026-09-29:** the route half of this decision as ruled on 2026-09-26 said routes "may sit in either layer"; the product thinker's ruling AA(b) of 2026-09-29 reverses that for a provider that holds a key, and this text is the decision as it now stands (adr-2609221009491186 carries the consequence).
9. **A provider claims no tier (2026-09-26).** A provider is reached by a role or a judgement type pointed at `<provider>/<model>`, or by a `--route` naming it, never by a tier alone, so `Connections.Serves` answers false for every tier. The bundled denylist is `anthropic/*`, the minimum ruled, and a reported model it matches discards the answer. **Amendment, 2026-09-30:** no denylist is bundled (ruling H9, adr-2609300107513982); a reported model an `oracle.denylist` entry the configuration writes matches discards the answer.

Taken in the wiring lane (autonomous run A, 2026-09-30), within the product thinker's ruling DR5 of 2026-09-29, which the decision log records on its dated entry for that ruling ("file-reading agents on a paid provider: only self-contained agents by default ... plus a central override, a settings list the person keeps, machine-level, not changeable by agents or repos"):

10. **A paid provider takes only a self-contained agent by default (2026-09-30, DR5).** A provider call carries no tools, so an agent that reads files cannot read them there. A provider whose block names a key admits an agent only if it is on a list compiled into the binary, default deny, as ruling AA(a) asked for an allow list: the four cold-reading positions, each handed one assembled, manifest-hashed bundle and nothing else. Every other agent is refused before any call, and before its verb writes anything, naming DR5 and the override. An agent joins the list only with a test proving its request carries all its input. A provider whose block names no key (a local server) spends nothing of the person's and is outside DR5.
11. **The override is `oracle.bundled_context_providers`, on the machine alone (2026-09-30, DR5).** It lists the providers that may take bundled-context requests for file-reading agents, read from `~/.abcd/config.json` through the same machine-layer seam the provider blocks use; a repository's `.abcd/config.json` declaring it is refused, as a repository's provider block is, and a name this machine has not configured is a diagnostic that admits nothing. It inherits the machine layer's trust in `$HOME`, pending iss-2609300012273350. A named provider takes a file-reading agent only once abcd builds that agent's bundle, and none is built: which files travel, a size cap and a scan before sending are choices no ruling makes, so every file-reading agent stays refused with the override too, and the refusal says so.
12. **The verb that emits the request sends it (2026-09-30).** `intent audit <itd-N>`, `intent consistency`, `intent audit --owed` (its head), `launch ship` and `spec close`'s review re-emit send the request they emitted, with the agent's own prompt from the plugin root, and run the answer through their own ingest; the receipt names the provider as the connection used, with the call record. A cold reading is sent by `reading ingest --dispatch <rdg-N>`, the parked run's bundle with the position's definition, because `reading assemble` keeps its invocation to a position and a target. A provider that could not be reached leaves the step to the host with one stderr line; any other failure exits 2.
13. **A host payload on a provider route is refused (2026-09-30).** An ingest handed a payload the host produced while its agent is routed to a provider refuses it at exit 2, since its receipt would name the provider for work the host did; `--route <agent>=host-decides` keeps one run on the harness. The four disembark agents read the packed lifeboat, and no verb builds a request carrying it, so none of them is sent to a provider and their ingests refuse a provider route this way.
14. **The close stands whatever its review does (2026-09-30).** `spec close` sends the review it emits when the auditor is routed to a provider and says what came back on stderr; a refusal (DR5 on a keyed provider) or a failed call leaves the review owed, as a warning. A cut the composer answers through a provider stages no `--payload-dir`.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** `oracle.api.openrouter` configured with a base URL, a key name and a model list, **when** a role or judgement pointed at a listed model runs, **then** the call goes over the OpenAI-compatible protocol with the host sub-agent's prompt, inputs and output contract; unconfigured, nothing changes and no call is attempted.
- **Given** a role or judgement configured for a model not on its provider's list, **when** the configuration is read, **then** it is refused before any call, naming the list.
- **Given** a model on its provider's list, whichever vendor made it, **when** the configuration is read, **then** it is admitted, because the provider's list alone decides and abcd bundles no vendor denylist; a listed model an `oracle.denylist` entry in the repository's or the machine's configuration matches is refused the same way, naming the entry.
- **Given** the key, **when** the adapter reads it, **then** it comes from the machine's configuration or the environment by name, and no credential is written into the repository.
- **Given** a call through the adapter, **when** the run record is read, **then** it names the provider, the model asked for and the model the provider reported.
- **Given** a repository abcd takes over with no provider configured, **when** `ahoy` runs, **then** it explains the aggregator, its use and what works without it, and offers the walkthrough; declining leaves the host as the only route and says so.
- **Given** the walkthrough accepted, **when** it asks where the key lives, **then** it offers the three homes (outside abcd; abcd-only, owner-only; the platform keychain) with the keychain recommended in the prose above the choice, writes the provider block with the first allowlist, verifies with one call, and writes nothing into the harness's settings or the repository.
- **Given** the lane, **when** it ships, **then** a security review of the network path and the credential handling is on its record.

## Audit Notes

Changed on 2026-09-30 by the technical facilitator's ruling H9 of 2026-09-29, recorded as iss-2609300110451242: criterion 3 read "**Given** a listed model whose prefix matches the vendor denylist, **when** the configuration is read, **then** it is refused the same way, and an allowlist entry does not override it." H9 retires the bundled `anthropic/*` denylist, so the criterion's reading changes from a vendor refusal to the allowlist alone, and the text above is the criterion as it now stands. adr-2609300107513982 supersedes adr-2609221009491186, whose decision 2 it revises.

- 2026-10-02 — Amendment to the closing clause of decision 8 ("a route naming a provider this machine has not configured stays on the host with a diagnostic"), which stands as written as the decision ruled on 2026-09-26 and 2026-09-29. The technical facilitator's rulings CD3 and CD4 of 2026-10-02 narrow it for a repository's route. CD4, verbatim: "OWNER'S SETTING APPLIES — skip the repo route (with a warning) when a machine-layer route for that name exists; adr-25's 'stays on host' sentence amended": a repository's route to a provider this machine has not configured no longer displaces the machine's own route of the same name, which applies in its place, and the route stays on the host with a diagnostic only where the machine has no route for that name. CD3, verbatim: "SKIP THAT ROUTE, WARN — skip with a diagnostic; the machine's route for the role applies; adr-25 amended": a repository's route naming a model a configured keyless provider does not offer is skipped with a diagnostic rather than refusing the configuration, and the machine's route for the role applies. Both are built in lane routeSkip of autonomous run A, not on the default branch at this note's base; recorded by lane rd2.
- 2026-10-02 — Declared mutual pair with itd-2609170822093401 (the technical facilitator's ruling CY2 of 2026-10-02, verbatim: "SAME, DECLARED PAIRS — each direction names the piece it needs"). itd-2609081951381895's builds_on itd-2609170822093401 needs the routing seam: the per-agent route and the invocation-time `--route` override that point a role or a judgement type at `<provider>/<model>`, the only ways a provider is reached (the adapter's decision 9). itd-2609170822093401's builds_on itd-2609081951381895 needs the provider connections its winning rows are resolved against, and the adapter's refusal of a setting its provider does not accept (its out-of-scope list names the adapter for both). Both edges stand. The record has no field record-lint's edge_cycle rule reads as a declared pair, so the linter still reports the cycle at warn until it does (iss-2610020836255174). Each piece is read from the two records' own scope, criteria and decisions; the ruling names none. Recorded by lane rd2 of autonomous run A.

<!-- abcd-review: OWED receipt=rcp-ac4127f9199c -->
Fidelity review OWED (receipt rcp-ac4127f9199c).
<!-- abcd-review-end receipt=rcp-ac4127f9199c -->

## Grounds

- pursued: Jev is reachable only this way and the person's subscription must stay where Opus runs; we expect the listed route to serve only what was meant; shown wrong if a frontier model is ever billed through the adapter
