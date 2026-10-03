---
id: itd-2610030821294016
slug: connecting-a-model-service-from-inside-claude-code-works-out
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610030810370060]
refines: [itd-2609081951381895]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Connecting a model service from Claude Code looks up the details and offers the choices

## Press Release

> When a person connects a model service from inside Claude Code, abcd works out the details with them instead of asking them to type each one. It asks for the service's address and, on a yes, looks up the models the service offers. It suggests the models the person already uses through their other connections, and finds anything else when they type part of a name. It then shows one command, with every file that command will write, for the person to paste into a terminal: nothing is set up from inside the session. Where a service needs the key before it will list its models, that one terminal step asks for the key with the typing hidden, lists the models, lets the person pick one, checks the pick with a real answer, and only then saves anything. A service that lists no models asks for the name to be typed, and says why.

_Proposed by the facilitator from decisions 1 to 8; to be confirmed by the product thinker._

Previous wording (superseded at the interview):

> Connecting a model service from inside Claude Code works out the details with the person instead of asking them to type them. It looks up what it can: the service's own list of models from the address given, where the person's key could be kept on this machine, and abcd's settings folder. It offers the choices one question at a time, and then either sets the connection up itself or prints the exact command to paste into a terminal. When a service lists hundreds of models, the person sees three suggestions and can type part of a name to narrow the list; the full searchable list is the plain-Terminal picker's (itd-2610030810370060).

> _Proposed; to be confirmed at the planning interview._

## Why This Matters

The product thinker asked on 2026-10-03, as relayed by the technical facilitator (the routing row in the decomposition-calibration note), that `ahoy connect`, used from the harness, probe for its flag values (`--base-url`, `--model`, `--home` and the rest) and offer options, so that it either runs the setup itself or tells the person the exact terminal command, because nobody should have to work out exact model names: easy with a handful of models, hard with hundreds (a routing service such as OpenRouter lists several hundred).

Today `abcd ahoy connect <provider>` needs every value typed: the base URL, `--model` repeated for each model the provider may serve (the first becomes the verification call's model), `--home` (none, external, abcd, or keychain) with `--env` or `--file` and `--field` for an external home, and `--key`, which names the credential rather than carrying its value. It verifies the provider with one call and refuses a key typed at a terminal. The shipped intent itd-2609081951381895 owns the adapter and this verb; its decision 5 hands the external and keychain homes to the credential store, itd-2609221017023290 (shipped), whose standard-input discipline the key's handling rests on.

What the binary could look up without asking, once built (no such call exists today): an OpenAI-compatible service's model list (its `/models` endpoint, at the base URL the person names), the homes a key can live in on this machine (environment variables and known tool configuration files, named but never read for their value until the person chooses), and abcd's own settings folder. The proposed promise (see the review findings below: it is a discipline, not a property): connect never prints, echoes or records the key, and the guided path never asks for it; the person stores it, and the binary reads it from the chosen home.

Routed out: whether a request is judged against a model's published size before it is sent is the adapter's question, not connect's; it is captured as its own record, a follow-up of the streaming fix iss-2610030931521214. <!-- record-lint: forward-looking -->

Typed links: refines itd-2609081951381895 (its connect verb gains a guided path); the arrow-key list the product thinker wants later is routed to itd-2610030810370060 (abcd's interviews in a plain Terminal), with model choice as its first example (the product thinker's routing, 2026-10-03); the questions it asks follow itd-2610030810350727 (one question layout).

## Mechanism

We expect a person to connect a service listing hundreds of models without knowing an exact model name because abcd reads the names from the service's own list and the person only types part of one to narrow it; a person who still has to type a full model name for a service that publishes its list shows the claim wrong.

_Proposed by the facilitator from decisions 1 to 8; to be confirmed by the product thinker._

## Scope Conditions

- Services that publish an OpenAI-compatible model list; a service that publishes none falls back to typing the name (decision 5).
- A person who can open a terminal on the same machine to paste the printed command; the session never finishes the setup itself (decisions 1 and 2).
- Suggestions only for a person with an existing connection; a first-time user sees the search alone (decision 6).
- A key that reaches abcd through its own hidden input in that terminal step; a key saved by hand in the computer's password store is not adopted (decisions 2 and 7).

_Proposed by the facilitator from decisions 1 to 8; to be confirmed by the product thinker._

## Acceptance Criteria

_Proposed by the facilitator from decisions 1 to 8 and the design review's proposed criteria that survive them; each is unconfirmed until walked with its addressee. Review criterion C3 (it let the session run the setup for some homes) is rewritten to decision 1's always-print rule; C5 is rewritten to decision 2's single terminal step, with no listing taken as the test; C8 (a listed model size) is dropped, because decision 8 routes the size question out. How many suggestions a question shows is owed at planning; G2 proposes at most three._

- G1 (technical facilitator; unconfirmed) Given a service address, when the guided path offers to look up its models, then one question first shows the scheme and host, and only on a yes does a request go out, carrying no key, following no redirect, and keeping the adapter's short timeout and size cap (example: a stand-in service that answers with a redirect is refused and never sees a key header); a go test asserts each.
- G2 (product thinker; unconfirmed) Given a person whose existing connections use two models a service of 300 lists, when the model question is asked, then it suggests those two (at most three) and typing part of a name narrows the full list without a second request, while a first-time user sees no suggestions, only the search (example: typing "coder" narrows 300 names to the few that contain it); a go test asserts both cases.
- G3 (technical facilitator; unconfirmed) Given any guided connect run inside a session, when it ends, then it has printed exactly one command and written nothing: abcd's settings folder and every key home are byte-identical before and after (example: a run that reaches the end with a keyless local service still only prints the command); a go test asserts it.
- G4 (product thinker; unconfirmed) Given the values worked out with the person, when the command is printed, then the command and every path it will write are shown in full before the person runs it, and no setting skips that (example: the command is followed by the connection file it writes in abcd's settings folder); a go test asserts the output, and a dated receipt from a Claude Code session shows it on screen.
- G5 (technical facilitator; unconfirmed) Given a stand-in service that refuses to list its models without the key, when the printed command runs in a terminal, then the key is read from hidden input, the list is fetched with the key held in memory, the person picks a model, a real completion verifies it, and only then are both writes made, while a failed completion writes nothing and no listing counts as the test (example: a stand-in that lists with the key but fails the completion leaves no connection and no stored key); a go test asserts both paths.
- G6 (technical facilitator; unconfirmed) Given a known key value, when the guided path and its printed command run end to end, then the value appears in no question, answer, printed command, error or record (example: a canary key string is searched for in every output and file written, and found only in the chosen key home); a go test asserts it.
- G7 (product thinker; unconfirmed) Given a service whose model list request fails or returns nothing usable, when the model question is reached, then the person is asked to type the model name and told the service publishes no list (example: a stand-in answering not found on its model list); a go test asserts the fallback and its reason.
- G8 (product thinker; unconfirmed) Given a key the person saved by hand in the computer's password store, when the guided path asks where the key lives, then it does not offer that item, and the printed command takes the key again through abcd's hidden input (example: an item named for the service in the password store is not listed as a choice); a go test asserts it.

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

## Decisions

1. 2026-10-03, the product thinker at the planning interview, asked whether connecting from inside Claude Code may finish the setup itself when no key has to move (gain: one step for a keyless service; cost: two paths, and a reversal of the shipped intent's decision 6): always print the command. abcd works out every value with the person, then prints one command for them to paste into a terminal. The shipped intent's decision 6 (the walkthrough is a verb the person runs) stands, so reversal 5(a) of the record review does not arise.
2. 2026-10-03, decided without a question (follows from decision 1, the technical facilitator's): a service that needs the key to list its models is handled inside the printed command, in one Terminal step: the key arrives on hidden input, the models are listed with it in memory, the person picks one, a real completion verifies it, and only then are both writes made. The shipped intent's decision 7 (verify with the first model, then write) stands; no listing is accepted as the test, and no service-specific check is written (its decision 2 stands). The picker in that step is itd-2610030810370060's.
3. 2026-10-03, decided without a question (brief invariant 1 settles it): the printed command, and every path it writes, are shown in full before the person runs it; no setting skips that.
4. 2026-10-03, decided without a question (adr-25's default settles it): looking up a service's models before setup is a network call, so one question shows the scheme and host first, and the look-up runs only on a yes; it carries no key, follows no redirect, and keeps the adapter's short timeout and size cap.
5. 2026-10-03, decided without a question (the only option besides deferral): a service that publishes no model list falls back to typing the model name, and the person is told why.
6. 2026-10-03, the product thinker, asked where suggestions for a long model list come from (the person's own connections; the service's popular list in prose; search only): models the person already uses, from their existing connections; anything else is found by typing part of a name. The product thinker accepted the cost shown: a first-time user sees no suggestions, only the search. No service's ranking appears in or beside the question, so reversal 5(f) does not arise.
7. 2026-10-03, the product thinker, asked whether abcd may use a key already saved by hand in the computer's password store, by its name: no; the person stores the key again through abcd's own hidden input. The credential store's adoption of a hand-made item stays out of scope.
8. 2026-10-03, decided without a question (the record review's finding 2, both reviews agreeing): the request-size question is routed out to its own capture, iss-2610030956156354. <!-- record-lint: forward-looking -->

## Open Questions

None open: the interview of 2026-10-03 answered them (decisions 1 to 8). Owed at planning: the press release and the acceptance criteria rewritten to the decisions (the press release no longer says the connection may be set up from the session), for the product thinker to confirm; the Mechanism and Scope Conditions; and how many suggestions a question shows (at most three, under the layout's limits).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
