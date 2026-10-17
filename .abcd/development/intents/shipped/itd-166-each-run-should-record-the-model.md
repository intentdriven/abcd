---
id: itd-166
slug: each-run-should-record-the-model
spec_id: spc-2610170652111963
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2608290811463906, iss-2610161709506207, iss-2610161734413178, iss-2610161734428291]
impact: additive
---

# A run records what it was actually run with

## Press Release

> **Every model call abcd makes now records two sides: what abcd asked for, and what actually answered.** A role run through the host, a command-line runner or a provider connection names the model it requested and the settings it sent, then the model, provider, token counts and cost the answer reported. Whatever the route cannot show is written down as unpinned or self-reported, never left out and never guessed.
>
> "A review came back weaker than last week's and I spent an hour blaming the diff," said a technical facilitator. "The record now shows the provider had served a different model under the same name. It took one line to see."

## Why This Matters

A model identifier does not determine behaviour, so a record that names only the identifier does not make a run interpretable or reproducible. Whatever the model exposes belongs alongside it: reasoning or thinking depth, sampling parameters, a seed where one exists, the context-window variant, and any speed or quality mode.

Two runs stamped with the same identifier can differ on all of these, so a record naming only the identifier invites a false comparison between them. An audit verdict, a benchmark, or a regression blamed on a code change may in fact be a settings difference. That matters most where a verdict is treated as evidence.

This is facilitator-tier diagnostic data and it never surfaces to the product thinker: its use is the trace backwards when a report arrives that the product does not match what was expected. That also settles where it lives, since a product-thinker-facing record carries none of it.

The hard constraint is that a model cannot reliably introspect the settings it runs under, so the dispatching layer has to supply them. Any design that asks the agent to state its own settings inherits the truthfulness problem the attribution gate already has.

Widened by the decomposition of itd-2610161733316518 (2026-10-16, routing confirmed by the product thinker): the record has two sides. The **asked** side is what the dispatching layer requested: model and snapshot, provider and any provider pin, sampling, seed, effort, context, and the prompt and tool definitions by hash. The **served** side is what actually answered: the model and upstream provider the response names, any fallback taken, token counts including reasoning and cached, and cost where the provider reports it. A local model adds the weight file's fingerprint, its quantisation, the runtime and its version, and the served context. Whatever a route cannot expose is written down as unpinned or self-reported rather than left out (see the principle that a model run is re-runnable, not reproducible).

Today (checked 2026-10-17): the provider adapter records the model asked and the model the API reported, and drops the usage it already parses; the claude runner reads the served model from its init event and ignores the usage and cost in its result event; the opencode runner stamps the asked model as the served one (iss-2610161709506207); no route records a prompt hash, an upstream provider, a generation id or a cost.

## Mechanism

We expect a call record built by the dispatching layer, from the request it sent and the response it read, to show a settings or model difference that a record naming only the asked model hides, because the layer that sends the request and reads the answer is the only one that sees both without asking the model about itself.

## Scope Conditions

- Holds for the routes abcd dispatches through today: the claude and opencode command-line runners, the OpenAI-compatible provider adapter, and the oracle's hand-off to the host harness. A loop role the host runs as its own sub-agent keeps the receipt it has today, which carries no route. <!-- cond: cond-2610170652113675 -->
- Holds only for what each route's own output carries: a field the response or event stream does not name is recorded as unpinned or self-reported, and no extra network request is made to fill it. <!-- cond: cond-2610170652112593 -->
- Holds for local models only as far as the asked side configured them: the weight fingerprint, quantisation and runtime version are recorded once a local route can be pinned (iss-2610161734428291), and are marked unpinned until then. <!-- cond: cond-2610170652113516 -->

## Acceptance Criteria

- Given a role routed to the provider adapter, when the call returns, then its route receipt carries the asked side (model, every setting sent, and a sha256 of the rendered messages) and the served side (the model the response names, the upstream provider when the response names one, the generation id, the finish reason, prompt, completion, reasoning and cached token counts, and the cost when the response reports it).
- Given a role run by the claude runner, when the run ends, then its route record carries the asked model it passed, the served models the result event's per-model usage names, the token counts and the cost from the result event, and the harness version from the init event.
- Given a role run by the opencode runner, when the run ends, then the served model is empty and listed as not reported (its run mode names no model), never a copy of the asked model, and the token counts and cost are summed across its step-finish events.
- Given any route, when a field of the fixed asked or served list cannot be read from what the route returned, then the record lists that field by name with its reason (not reported, self-reported, computed by the harness, or not configured) rather than omitting it.
- Given a role the oracle hands to the host harness, when its receipt is written, then the served model is the payload's own declaration listed as self-reported.
- Given a provider whose server refuses the usage-in-stream request, when the person turns that request off for the provider, then calls succeed with usage listed as not reported, and the refusal's error names the setting.
- Given the run state, a lane receipt and the consistency answer's route file, when one is written, then the two sides travel with the existing route records, every string a provider or harness supplied is cleaned and bounded, and no new file is written.
- Given a call, when its record is built, then building it adds no request to the ones the call already makes.

## Decisions

Recorded by the technical facilitator on the product thinker's delegation (2026-10-17: "build autonomously"), each with its reason, for review on return. A design review pair (Opus and Sonnet, 2026-10-17) returned NEEDS-FIXES on the first plan; every finding is applied below (import direction, opencode's run mode, the schema bump, masking of route files, the opt-out, the request-count test, the fixed lists, harness-computed cost).

1. **Who reads it (the open question).** The record stays facilitator-tier. The model-matrix report shows the product thinker a drift flag and the names of the fields that drifted, never their values; the values are read from the run record. Reason: the draft's own constraint, and itd-2610161733316518's decision that a re-run reports drift field by field.
2. **Extend, do not add, a receipt.** The two sides extend `oracle.CallRecord` and `oracle.ReceiptRoute` (the override's receipt) and `runner.RouteRecord` (the runner's). The shared vocabulary lives in `oracle`, which the runner package already imports, so neither imports the other the wrong way. Reason: itd-2610161733316518 decision 14 says the sides extend the route receipt the override already writes; a second receipt would be a second place to disagree.
3. **No extra request.** OpenRouter's generation record is not fetched here; what the response itself carries (its `id`, `provider`, `usage` with cost and token details) is recorded, and the readback belongs to the OpenRouter pin (iss-2610161734413178) inside a consented run. Reason: itd-2610161733316518 decision 12 (every network call inside the run the person started) and this record's scope condition 2.
4. **Usage in the stream, with an opt-out.** The adapter asks for usage in the stream with `stream_options.include_usage`, which OpenAI, vLLM, OpenRouter and Ollama accept and llama.cpp and LM Studio accept or ignore. A provider whose server refuses it is given an opt-out on its connection (default on), and a refusal names that setting. No automatic retry, since a retry is a second request. No provider-specific request field is added. Reason: a server that refused the field would otherwise turn a working route into a failure.
5. **The prompt by hash.** The rendered messages are recorded as a sha256, never their text. Reason: the run's transcripts already hold the text where the person has kept them, and a receipt that carries prompts carries whatever they quoted.
6. **The opencode fix lands here, as its remedy's second branch.** opencode 1.18.31's run mode names no model in its JSON events (checked in the installed binary by the review of 2026-10-17), so the runner stops copying the asked model, leaves the served model empty and lists it as not reported. That resolves iss-2610161709506207.
7. **Local fingerprints wait on the local pin.** The asked-side weight fingerprint, quantisation and runtime version need a configured pin that does not exist; until iss-2610161734428291 lands, a local route lists them as not configured. Reason: nothing can be recorded that was never configured, and guessing it from the server would be self-reported at best.
8. **Four reasons for a missing field.** Not reported (the route's output does not carry it), self-reported (the model or payload declared it), computed by the harness (claude's and opencode's cost come from the harness's own price table, not the provider), not configured (no pin exists). Reason: the review found that harness-computed cost is neither reported nor self-reported, and a reader must be able to tell the four apart.
9. **Run state moves to schema 13.** Older binaries read run state strictly, so the new route fields bump the version under the existing rule: a version-12 file reads as not recorded and is written back at 13. Reason: the run-state reader refuses unknown fields by design.
10. **Fixed field lists.** Asked: model, provider pin, each sampling setting, seed, effort, context, prompt sha256, harness version. Served: model, provider, generation id, finish reason, prompt, completion, reasoning, cache-read and cache-write tokens, cost. Each route lists exactly the fields it lacks. Reason: criterion 4 is testable only against a fixed list.

## Open Questions

None. The one open question was decided on delegation (decision 1).

## Typed Links

- **blocks `itd-2610161733316518`** (one feature built across a model matrix): the run's manifest is this record, and nothing it compares is interpretable without it.
- **resolves `iss-2610161709506207`** (the opencode runner stamps the asked model): decision 6.
- **refined later by `iss-2610161734413178` and `iss-2610161734428291`**: the OpenRouter and local pins fill fields this record marks unpinned.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-f25a69346de1 -->
Fidelity review OWED (receipt rcp-f25a69346de1).
<!-- abcd-review-end receipt=rcp-f25a69346de1 -->

## Grounds

- pursued: we expect a record built by the dispatching layer from the request it sent and the response it read to expose a model or settings difference that the asked model alone hides; a matrix re-run whose drift report names no field that the receipts show changed would show it wrong
