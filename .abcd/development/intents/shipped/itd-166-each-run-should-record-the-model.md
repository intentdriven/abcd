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
4. **Usage in the stream, with an opt-out.** The adapter asks for usage in the stream with `stream_options.include_usage`, which OpenAI, vLLM, OpenRouter and Ollama accept and llama.cpp and LM Studio accept or ignore. A provider whose server refuses it is given an opt-out on its connection (default on), and a refusal that names the field names that setting. An ordinary call is never retried, since a retry is a second request; the one exception, decided in the build's review (2026-10-17), is `ahoy connect`, whose verification is asked once more without the field after a 400 or 422 and, when that succeeds, writes the opt-out, because a setup that cannot get past the refusal leaves the person nothing to turn off. No provider-specific request field is added. Reason: a server that refused the field would otherwise turn a working route into a failure.
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

<!-- abcd-review: INGESTED receipt=rcp-f25a69346de1 -->
Fidelity review — receipt rcp-f25a69346de1 (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:beafc9d930e3df84cf97e1f5dd434a99d9edc2c29b38c1df4d1b066c7e38d00d
Input attestations: diff:7ae8fdde8..f5a25ce84@sha256:9ad6cfb8962de4c4e72d23e030a86520334354f4f25506c6a29eab57ce603037; tree:f5a25ce84@-;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: callRecord builds the asked side (model asked, prompt sha256 of the rendered messages, adapter-added field) beside the receipt's existing settings_sent, and the served side (reported model, upstream provider, generation id, finish reason, prompt/completion/reasoning/cache-read/cache-write counts, cost in credits) from the adapter's Result; the dispatch attaches it to the route receipt and a test reads an OpenRouter-shaped answer through to the encoded receipt
  evidence: internal/core/oracle/call.go:168 — "func callRecord(p Provider, brief openaiapi.Brief, settings Settings, res openaiapi.Result) CallRecord {"
  evidence: internal/core/oracle/call.go:170 — "PromptSHA256: openaiapi.PromptSHA256(brief), AdapterAdded: []string{},"
  evidence: internal/adapter/openaiapi/client.go:655 — "func readUsage(raw json.RawMessage) *Usage {"
  evidence: internal/core/oracle/dispatch.go:57 — "return payload, r.Receipt(ModelReported(payload)).WithCall(rec), nil"
  evidence: internal/core/oracle/receipt.go:72 — "SettingsSent Settings `json:"settings_sent"`"
  evidence: internal/core/oracle/sides_test.go:138 — "func TestTheCallRecordTravelsInTheReceipt(t *testing.T) {"
  evidence: internal/core/oracle/sides_test.go:61 — "func TestAProviderCallRecordsBothSides(t *testing.T) {"
- ac-2 — MET: the claude runner sets the asked model it passes (after the provider prefix), reads the result event's modelUsage keys as served models, its usage counts (thinking tokens as reasoning) and total_cost_usd, and the init event's claude_code_version; a test over a recorded 2.1.302 stream asserts each, with cost listed harness_computed
  evidence: internal/core/runner/claude.go:99 — "ans.ModelAsked, ans.PromptSHA256 = c.askedModel(), promptSHA256(req)"
  evidence: internal/core/runner/claude.go:193 — "func meterClaude(ans *Answer, line []byte) {"
  evidence: internal/core/runner/claude.go:214 — "ans.Cost = &oracle.Cost{Amount: *m.TotalCostUSD, Unit: oracle.CostUSD}"
  evidence: internal/core/runner/claude.go:236 — "ans.ServedModels = append(ans.ServedModels, cleanHarness(k))"
  evidence: internal/core/runner/claude.go:272 — "ans.HarnessVersion = cleanHarness(ev.Version)"
  evidence: internal/core/runner/sides_test.go:83 — "func TestClaudeRecordsBothSidesFromARealStream(t *testing.T) {"
- ac-3 — MET: the opencode runner no longer copies o.model into ans.Model: it records o.model as ModelAsked only, leaves the served model empty so servedUnpinned lists served.model not_reported, and meterOpenCode sums tokens and cost across step_finish events; a test over a 1.18.31 fixture asserts Model empty, ServedModels empty and the summed usage and cost
  evidence: internal/core/runner/opencode.go:96 — "ans.ModelAsked, ans.PromptSHA256 = o.model, promptSHA256(req)"
  evidence: internal/core/runner/opencode.go:99 — "servedUnpinned(&un, ans.Model, ans.Sides, ans.counted)"
  evidence: internal/core/runner/sides.go:109 — "un.Unless(model != "", oracle.ServedModel, oracle.NotReported)"
  evidence: internal/core/runner/opencode.go:137 — "func meterOpenCode(ans *Answer, line []byte) {"
  evidence: internal/core/runner/opencode.go:152 — "sum += ans.Cost.Amount"
  evidence: internal/core/runner/sides_test.go:153 — "func TestOpenCodeNeverStampsTheAskedModelAsServed(t *testing.T) {"
- ac-4 — MET_WITH_CONCERNS: every route builds an Unpinned list over the fixed asked and served lists with the four reasons (adapter callRecord, runner askedUnpinned/servedUnpinned, host HandOffUnpinned), and tests hold each list to the fixed fields; concern: the local fields (weights_sha256, quantisation, runtime_version) are listed not_configured only on the provider adapter when its base URL is loopback, while a runner routed to a local model (the opencode test itself uses local/qwen3-coder) omits them, and its test helper excludes LocalFields from the known list
  evidence: internal/core/oracle/call.go:191 — "if openaiapi.IsLocal(p.BaseURL) {"
  evidence: internal/core/runner/sides.go:91 — "func askedUnpinned(un *oracle.Unpin, s Sides) {"
  evidence: internal/core/runner/sides_test.go:49 — "func assertFixedLists(t *testing.T, us []oracle.Unpinned) {"
  evidence: internal/core/oracle/receipt.go:117 — "func HandOffUnpinned(model string) []Unpinned {"
  evidence: internal/core/oracle/sides_test.go:97 — "func TestACallNamingNoServedDetailListsEachMissing(t *testing.T) {"
- ac-5 — MET: on the harness leg Route.Receipt attaches HandOffUnpinned(model), which lists served.model self_reported when the payload declared one (not_reported when it declared none) and every other fixed field with its reason; the model passed is ModelReported(payload); a test asserts the self_reported entry in the encoded receipt
  evidence: internal/core/oracle/receipt.go:108 — "rr.Unpinned = HandOffUnpinned(model)"
  evidence: internal/core/oracle/receipt.go:129 — "un.Add(f, SelfReported)"
  evidence: internal/core/oracle/sides_test.go:220 — "func TestTheHostHandOffMarksTheServedModelSelfReported(t *testing.T) {"
- ac-6 — MET_WITH_CONCERNS: a provider block's stream_usage false drops stream_options from the request and the call succeeds with counts listed not_reported, and a 400/422 refusal whose body names stream_options or include_usage gets the oracle.api.< provider>.stream_usage setting named; concerns: a refusal whose body does not name the field carries no hint on an ordinary call, so the setting is named only for self-describing refusals; and setup (ahoy connect) retries the verification once without the field on any 400/422 and writes stream_usage false, which the intent's decision 4 (no automatic retry) did not provide for
  evidence: internal/adapter/openaiapi/client.go:548 — "fields["stream_options"] = map[string]bool{"include_usage": true}"
  evidence: internal/adapter/openaiapi/client.go:492 — "if named {"
  evidence: internal/core/oracle/call.go:115 — "return fmt.Errorf("%w; if this server does not take %s, set %s.%s.%s to false in %s, "+"
  evidence: internal/core/oracle/sides_test.go:165 — "func TestAConnectionThatTurnsStreamUsageOffSendsWithoutIt(t *testing.T) {"
  evidence: internal/core/oracle/sides_test.go:194 — "func TestARefusedUsageRequestNamesTheSetting(t *testing.T) {"
  evidence: internal/adapter/openaiapi/served_test.go:300 — "func TestARefusalNotNamingTheUsageFieldCarriesNoHint(t *testing.T) {"
  evidence: internal/core/oracle/connect.go:526 — "case p.StreamUsage == nil && errors.Is(err, openaiapi.ErrUsageRefused):"
- ac-7 — MET: the runner's Sides are embedded in the existing runner.RouteRecord that the dispatch and the drive already store on lane receipts and the run state (bumped to schema 13 with version-12 files read as not recorded), the provider call record rides the existing ReceiptRoute the consistency route file holds, maskRoute masks every new string, adapter strings go through scrub and cleanModel and harness strings through cleanHarness, and a test asserts the reviews directory gains no new kind of file
  evidence: internal/core/runner/receipt.go:24 — "Sides"
  evidence: internal/core/implement/loop/drive.go:141 — "rec := runner.RecordOf(route.Runner, ran, ans)"
  evidence: internal/core/implement/loop/state.go:185 — "const SchemaVersion = 13"
  evidence: internal/core/implement/loop/state_mask.go:177 — "func maskRoute(r runner.RouteRecord) runner.RouteRecord {"
  evidence: internal/adapter/openaiapi/client.go:747 — "Provider: cleanModel(c.scrub(a.provider)), Usage: a.usage}"
  evidence: internal/core/runner/sides.go:75 — "func cleanHarness(s string) string {"
  evidence: internal/surface/cli/sides_test.go:60 — "func TestTheConsistencyRouteFileCarriesBothSides(t *testing.T) {"
  evidence: internal/core/implement/loop/sides_test.go:112 — "func TestTheDrivePassesBothSidesThrough(t *testing.T) {"
- ac-8 — MET: callRecord and the runners' meters are pure functions of the request sent and the response or event stream read, and a test counts requests on a sized vLLM-shaped server with and without the usage request and served detail and finds them equal; the setup retry in Connect is a second verification request, not part of building any record (recorded under ac-6 and the gap audit)
  evidence: internal/adapter/openaiapi/served_test.go:204 — "func TestRecordingTheServedSideAddsNoRequest(t *testing.T) {"
  evidence: internal/core/oracle/call.go:168 — "func callRecord(p Provider, brief openaiapi.Brief, settings Settings, res openaiapi.Result) CallRecord {"
  evidence: internal/core/oracle/sides_test.go:194 — "func TestARefusedUsageRequestNamesTheSetting(t *testing.T) {"

Gap audit:
- honoured:
  - every model call records what abcd asked for and what answered, on the provider adapter, both command-line runners and the host hand-off
    evidence: internal/core/oracle/sides.go:56 — "AskedFields = []Field{AskedModel, AskedProviderPin, AskedTemperature, AskedTopP, AskedSeed,"
    evidence: internal/core/oracle/call.go:168 — "func callRecord(p Provider, brief openaiapi.Brief, settings Settings, res openaiapi.Result) CallRecord {"
    evidence: internal/core/runner/claude.go:193 — "func meterClaude(ans *Answer, line []byte) {"
    evidence: internal/core/oracle/receipt.go:117 — "func HandOffUnpinned(model string) []Unpinned {"
  - whatever a route cannot show is written down with a reason (not reported, self-reported, harness-computed, not configured), never left out or guessed
    evidence: internal/core/oracle/sides.go:67 — "const ("
    evidence: internal/core/runner/sides.go:121 — "un.Add(oracle.ServedCost, oracle.HarnessComputed)"
  - the opencode runner stops stamping the asked model as served (iss-2610161709506207)
    evidence: internal/core/runner/opencode.go:96 — "ans.ModelAsked, ans.PromptSHA256 = o.model, promptSHA256(req)"
    evidence: internal/core/runner/sides_test.go:153 — "func TestOpenCodeNeverStampsTheAskedModelAsServed(t *testing.T) {"
  - a served model differing from the asked one under the same name is visible on the route line, with upstream provider and cost
    evidence: internal/surface/cli/dispatch.go:115 — "line += fmt.Sprintf(", model asked %s, model reported %s", pc.ModelAsked, pc.ModelReported)"
    evidence: internal/surface/cli/sides_test.go:26 — "func TestTheRouteLineNamesTheAskedModelAndCost(t *testing.T) {"
  - the prompt is recorded by sha256, never its text (decision 5)
    evidence: internal/core/oracle/call.go:170 — "PromptSHA256: openaiapi.PromptSHA256(brief), AdapterAdded: []string{},"
- diverged:
  - decision 4 says no automatic retry, since a retry is a second request; delivered: ahoy connect's verification is asked once more without stream_options on any HTTP 400/422 (whatever the body names) and the block is written with stream_usage false (iss-2610170934491794), while the intent's decision text was left unchanged
    evidence: .abcd/development/intents/shipped/itd-166-each-run-should-record-the-model.md:64 — "No automatic retry, since a retry is a second request."
    evidence: internal/core/oracle/connect.go:530 — "if err := verify(without); err != nil {"
    evidence: internal/core/oracle/connect_test.go:602 — "func TestConnectAsksAgainWithoutUsageWhenItIsRefused(t *testing.T) {"
  - a refusal names the setting (decision 4) only when the refusal body names stream_options or include_usage; a bare 400/422 on an ordinary call carries no hint
    evidence: internal/adapter/openaiapi/client.go:183 — "func namesStreamUsage(body string) bool {"
    evidence: internal/adapter/openaiapi/served_test.go:300 — "func TestARefusalNotNamingTheUsageFieldCarriesNoHint(t *testing.T) {"
  - a local model's weight fingerprint, quantisation and runtime are listed not configured on a local route (decision 7); delivered only on the provider adapter when its base URL is loopback, not on a runner routed to a local model
    evidence: internal/adapter/openaiapi/client.go:240 — "return err == nil && u.Host != "" && loopback(u.Hostname())"
    evidence: internal/core/runner/sides.go:91 — "func askedUnpinned(un *oracle.Unpin, s Sides) {"
- missing:
  - a generation id on the runner routes: both runners always list served.generation_id as not reported rather than recording one
    evidence: internal/core/runner/sides.go:111 — "// Neither harness names a generation id for the run: a run is many"

Scope-condition dispositions:
- cond-2610170652113675 — survived: the sides were added to exactly the four named routes (claude and opencode runners, the OpenAI-compatible adapter, the oracle hand-off), and a host-run loop role's RouteRecord keeps its old shape because Sides is embedded with every field omitempty and set only when a runner ran
  evidence: internal/core/runner/receipt.go:24 — "Sides"
  evidence: internal/core/runner/dispatch.go:156 — "o.Receipt.Route = RecordOf(o.Receipt.Route.Asked, route, ans)"
  evidence: internal/core/oracle/receipt.go:108 — "rr.Unpinned = HandOffUnpinned(model)"
- cond-2610170652112593 — survived: every served value is read from the response or the harness event stream, absent ones are listed unpinned, no generation readback is fetched, and the request count is equal with and without the record; the setup retry is a second verification request, not a fetch to fill a field
  evidence: internal/adapter/openaiapi/served_test.go:204 — "func TestRecordingTheServedSideAddsNoRequest(t *testing.T) {"
  evidence: internal/core/oracle/call.go:207 — "un.Unless(u != nil && u.Cost != nil, ServedCost, NotReported)"
- cond-2610170652113516 — narrowed: the local fields are marked not_configured until a pin exists, but only where the provider adapter's base URL is a loopback address; the runners never list them, even for a local model (the opencode test uses local/qwen3-coder), and a local server on a non-loopback address is not recognised as local
  narrowing: holds for the provider adapter route whose base URL is localhost or a loopback address only; command-line runner routes to local models and local servers reached on non-loopback addresses record no local-field entries at all
  evidence: internal/core/oracle/call.go:191 — "if openaiapi.IsLocal(p.BaseURL) {"
  evidence: internal/adapter/openaiapi/client.go:240 — "return err == nil && u.Host != "" && loopback(u.Hostname())"
  evidence: internal/core/runner/sides_test.go:49 — "func assertFixedLists(t *testing.T, us []oracle.Unpinned) {"
<!-- abcd-review-end receipt=rcp-f25a69346de1 -->

## Grounds

- pursued: we expect a record built by the dispatching layer from the request it sent and the response it read to expose a model or settings difference that the asked model alone hides; a matrix re-run whose drift report names no field that the receipts show changed would show it wrong
