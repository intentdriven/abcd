---
id: itd-2610030821294016
slug: connecting-a-model-service-from-inside-claude-code-works-out
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
refines: [itd-2609081951381895]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Connecting a model service from Claude Code looks up the details and offers the choices

## Press Release

> Connecting a model service from inside Claude Code works out the details with the person instead of asking them to type them. It looks up what it can: the service's own list of models from the address given, where the person's key could be kept on this machine, and abcd's settings folder. It offers the choices one question at a time, and then either sets the connection up itself or prints the exact command to paste into a terminal. When a service lists hundreds of models, the person sees three suggestions and can type part of a name to narrow the list; the full searchable list is the plain-Terminal picker's (itd-2610030810370060).

> _Proposed; to be confirmed at the planning interview._

## Why This Matters

The product thinker asked on 2026-10-03, as relayed by the technical facilitator (the routing row in the decomposition-calibration note), that `ahoy connect`, used from the harness, probe for its flag values (`--base-url`, `--model`, `--home` and the rest) and offer options, so that it either runs the setup itself or tells the person the exact terminal command, because nobody should have to work out exact model names: easy with a handful of models, hard with hundreds (a routing service such as OpenRouter lists several hundred).

Today `abcd ahoy connect <provider>` needs every value typed: the base URL, `--model` repeated for each model the provider may serve (the first becomes the verification call's model), `--home` (none, external, abcd, or keychain) with `--env` or `--file` and `--field` for an external home, and `--key`, which names the credential rather than carrying its value. It verifies the provider with one call and refuses a key typed at a terminal. The shipped intent itd-2609081951381895 owns the adapter and this verb; its decision 5 hands the external and keychain homes to the credential store, itd-2609221017023290 (shipped), whose standard-input discipline the key's handling rests on.

What the binary could look up without asking, once built (no such call exists today): an OpenAI-compatible service's model list (its `/models` endpoint, at the base URL the person names), the homes a key can live in on this machine (environment variables and known tool configuration files, named but never read for their value until the person chooses), and abcd's own settings folder. The proposed promise (see the review findings below: it is a discipline, not a property): connect never prints, echoes or records the key, and the guided path never asks for it; the person stores it, and the binary reads it from the chosen home.

Routed out: whether a request is judged against a model's published size before it is sent is the adapter's question, not connect's; it is captured as its own record, a follow-up of the streaming fix iss-2610030931521214. <!-- record-lint: forward-looking -->

Typed links: refines itd-2609081951381895 (its connect verb gains a guided path); the arrow-key list the product thinker wants later is routed to itd-2610030810370060 (abcd's interviews in a plain Terminal), with model choice as its first example (the product thinker's routing, 2026-10-03); the questions it asks follow itd-2610030810350727 (one question layout).

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## State of the art (2026-10-03; full report in the local tier, reports/sota-guided-connect.md)

- The standard model list (a request for `/models` at the address given) has no context-size field; servers add their own, and they differ in meaning: some give the size the server actually runs, others the size the model was trained for (a 128k trained figure can sit under a 4k served one). The figure's source must travel with it.
- Several local servers list models with no key; OpenAI, a key-protected vLLM, and (per its documentation) OpenRouter need the key to list. OpenRouter can search and sort its list on its own side (by name, popularity, price, context).
- In the harness, three suggestions plus a search beats paging: three at a time, 300 models would take about 100 questions.
- OpenRouter shortens an oversize prompt for small models unless told not to, which would break "the record shows what answered".
- A request near a model's limit can only be judged exactly by the server (a few have a token-count endpoint); the error text names both numbers.
- abcd's adapter already refuses to follow a redirect, so a look-up built on it cannot carry a key to another address.

Two questions this puts to the product thinker before criteria are written:

- The order for a service that needs the key to list its models: today abcd stores a key only after a test call with a model, so it cannot list first and choose after. Options: accept the list itself as the test where a wrong key is refused; a service-specific check; or have the person store the key and choose the model in one Terminal step, outside the session.
- A key the person has already put into the computer's own password store, with the input hidden, typed into the system's tool and not into abcd: may abcd adopt it by name?

## Review findings (design and feasibility, then record discipline, 2026-10-03, reports/review-connect-design.md and reports/review-connect-records.md in the local tier)

Nothing here is settled; each item is put to its addressee at the interview.

- Blocking: "the key never passes through the conversation" is stated as a property but is a discipline: the agent runs as the person and could read a stored key with one command. Proposed promise: connect never prints, echoes or records the key, and the guided path never asks for it; the agent's own reads are governed by the host's permission deny list, with a documented recipe, as a scope condition.
- Blocking, which homes "run it" may use, decidable from the code: it may run for no key, for an environment variable already set in the session, and for a file pointer; for abcd's own credential file and the system password store, whose value arrives on standard input and is refused from a terminal, it prints the command. Adopting a password-store item the person created by hand does not work today (it reads as not held); that needs its own scope and a ruling on "refuses a key typed at a terminal", or the password-store path prints the command this release.
- Blocking, the circular order, ranked by safety and generality: (c) one terminal step outside the session (no model given, the key on standard input, list with the key in memory, the person chooses, a real completion verifies, then both writes land) works for every service; (a) a listing that refuses a bad key as the test is weaker (a key with no credit lists fine) and needs a new seam; (b) a service-specific check contradicts the shipped adapter's decision 2 ("OpenRouter is configuration of this adapter, not code").
- Should-fix: "run it" is one more question showing the exact command and every path it writes, answered yes, print it instead, or decide later (the connect page already says it announces itself).
- Should-fix, long lists: with three models per question, the free-text "Other" row is the search box (so that question carries no side preview, which would remove the row); suggestions come from the service's own order (for OpenRouter, most popular; for a local server, the loaded models), and from the person's routes once one exists; the binary narrows over a cached list without a second call.
- Should-fix, the look-up: http is already refused except on this machine, so a local-network address over http is refused; the look-up is a GET with no key, no redirect followed, a short timeout and the size cap, after one question showing the scheme and host.
- Consider: request size is the adapter's (iss-2610030931521214 and its follow-up), not connect's; connect may show a listed size labelled with its meaning and source, storing none of it. <!-- record-lint: forward-looking -->

Applied from the record-discipline review: the press release no longer promises the full list (the plain-Terminal picker's), the request-size question is routed out, the `refines` link is in the front matter, and the statements about `--home`, `--key`, the look-up and the key are corrected. Reversals of the shipped intent's decisions for the product thinker to confirm at the interview: setting up from inside the session (its decision 6); the list itself as the test (decision 7); any service-specific check or service-named parameter (decision 2); a look-up before setup (a network call adr-25's default does not make); a setting that runs without asking (invariant 1); a service-ordered shortlist inside a question (the asking rules put recommendations in prose).

Proposed acceptance criteria (agent-seeded, unconfirmed): C1 keyless listing call rules and the host shown first (facilitator); C2 three suggestions, search, narrowing (product thinker); C3 run only for no key, a session variable or a file pointer, else print the command (facilitator); C4 one confirming question showing the command and every path written (product thinker); C5 the one-terminal-step path for a service that refuses keyless listing (facilitator); C6 no key value in any question, answer, printed command or error (facilitator); C7 a service with no model list falls back to typing, with the reason told (product thinker); C8 a listed size labelled with its meaning and source, nothing written (facilitator). Full text in the review report; each is walked with an example at the interview.

## Open Questions

- Run it or print it: whether the person chooses each time, or a setting decides, and whether "run it" may write the person's machine settings from inside a session without a confirmation step.
- Suggestions for a long list: by what the person's routes already use, by a short curated list abcd ships, by the service's own ordering, or by a search the person types; how many per page, given that the harness question view holds at most four options (itd-2610030810350727), so paging through hundreds means a question per page or a search step.
- The look-up call: listing models is a call to the service before the person has confirmed it, and some services need the key to list; whether that call waits for the key to be stored first.
- A service that publishes no model list: fall back to typing, or to the curated list.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
