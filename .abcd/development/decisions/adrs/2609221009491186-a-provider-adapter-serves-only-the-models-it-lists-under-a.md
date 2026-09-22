---
id: adr-2609221009491186
slug: a-provider-adapter-serves-only-the-models-it-lists-under-a
status: accepted
date: 2026-09-22
supersedes: null
superseded_by: null
related_intents: [itd-2609081951381895, itd-2609221009495079, itd-2609170822093401, itd-6]
related_rfcs: []
related_adrs: [adr-25]
---

# ADR-2609221009491186: A provider adapter serves only the models it lists, under a vendor denylist no listing overrides; everything else runs on the host

## Context

adr-25 makes the model host-delegated by default: the agent host owns model
choice, credentials and execution, and abcd's adapters (an OpenAI-compatible
API, a command-line runner, a reviewer route over MCP) are opt-in. The first
aggregator to reach abcd through the API adapter is OpenRouter, brought by a
new decision model (Jev, TypeSafe AI, September 2026) that is reachable
only that way. An aggregator serves every vendor's models under one key,
including the frontier models the person already pays for through the host
and its subscription. The adapter draft as written would route any model
name it was handed. The product thinker asked, on 2026-09-21, how to
integrate OpenRouter and how to ensure only certain models are reached
through it, so that a frontier model is never billed through the
aggregator or its data routed through a third party unasked.

## Decision

We will make every provider adapter default-deny by model.

1. **An allowlist per provider.** Each provider block in the configuration
   lists the model identifiers it may serve. A role or a judgement
   configured for a model not on its provider's list is refused when the
   configuration is read, before any call is made, naming the list.
2. **A vendor denylist above the allowlist.** A bundled denylist of vendor
   prefixes (`anthropic/*` at minimum) refuses a listed model whose prefix
   it matches, and no allowlist entry overrides it; the repository or the
   machine may extend the denylist and never shorten it below the bundled
   set.
3. **Everything else runs on the host.** A model that is neither listed nor
   the host's own is not a route; the host and the person's subscription are
   where frontier models run.
4. **The key is named, never stored.** The adapter reads its credential from
   the machine's configuration or the environment by name; nothing about a
   key enters the repository.
5. **The run record names the route.** Every call through an adapter records
   the provider, the model identifier asked for and the model the provider
   reports, so a substitution by the aggregator is visible.

## Alternatives Considered

- **Any model the key can reach, controlled at the provider.** Rejected: the
  aggregator's own controls are outside the record, invisible to the run,
  and a listing mistake on the provider's side would route a frontier model
  through it with nothing in abcd to say so.
- **An allowlist only, no denylist.** Rejected: a listing mistake in the
  repository's configuration would reach a frontier model; the denylist is
  the rule that a mistake cannot override.
- **A denylist only.** Rejected: default-allow leaves every newly listed
  aggregator model routable the day it appears.

## Consequences

- The API adapter intent (itd-2609081951381895) carries the list, the
  denylist, the refusal and the record as criteria; the model tier
  (itd-2609170822093401) reads the lists when it proposes a route; the
  decision adapter (itd-2609221009495079) can name only a listed model.
- A person who wants a frontier model through an aggregator edits the
  denylist on their machine, deliberately, and the run record shows it.
- The brief's adapters chapter gains the invariant.
