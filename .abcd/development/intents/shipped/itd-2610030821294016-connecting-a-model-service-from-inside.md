---
id: itd-2610030821294016
slug: connecting-a-model-service-from-inside
spec_id: spc-2610031241482088
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610030810370060]
refines: [itd-2609081951381895]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# Connecting a model service from inside your coding session looks up the details and offers the choices

## Press Release

> When a person connects a model service from inside your coding session, abcd works out the details with them instead of asking them to type each one. It asks for the service's address and, on a yes, looks up the models the service offers. It suggests the models the person already uses through their other connections, and finds anything else when they type part of a name. It then shows one command, with every file that command will write, for the person to paste into a terminal: nothing is set up from inside the session. Where a service needs the key before it will list its models, that one terminal step asks for the key with the typing hidden, lists the models, lets the person pick one, checks the pick with a real answer, and only then saves anything. A service that lists no models asks for the name to be typed, and says why.

_Proposed by the facilitator from decisions 1 to 8; confirmed as written by the product thinker at the planning interview, 2026-10-03 (decision 9)._

Previous wording (superseded at the interview):

> Connecting a model service from inside your coding session works out the details with the person instead of asking them to type them. It looks up what it can: the service's own list of models from the address given, where the person's key could be kept on this machine, and abcd's settings folder. It offers the choices one question at a time, and then either sets the connection up itself or prints the exact command to paste into a terminal. When a service lists hundreds of models, the person sees three suggestions and can type part of a name to narrow the list; the full searchable list is the plain-Terminal picker's (itd-2610030810370060).

> _Proposed; to be confirmed at the planning interview._

## Why This Matters

The product thinker asked on 2026-10-03, as relayed by the technical facilitator (the routing row in the decomposition-calibration note), that `ahoy connect`, used from the harness, probe for its flag values (`--base-url`, `--model`, `--home` and the rest) and offer options, so that it either runs the setup itself or tells the person the exact terminal command, because nobody should have to work out exact model names: easy with a handful of models, hard with hundreds (a routing service such as OpenRouter lists several hundred).

Today `abcd ahoy connect <provider>` needs every value typed: the base URL, `--model` repeated for each model the provider may serve (the first becomes the verification call's model), `--home` (none, external, abcd, or keychain) with `--env` or `--file` and `--field` for an external home, and `--key`, which names the credential rather than carrying its value. It verifies the provider with one call and refuses a key typed at a terminal. The shipped intent itd-2609081951381895 owns the adapter and this verb; its decision 5 hands the external and keychain homes to the credential store, itd-2609221017023290 (shipped), whose standard-input discipline the key's handling rests on.

What the binary could look up without asking, once built (no such call exists today): an OpenAI-compatible service's model list (its `/models` endpoint, at the base URL the person names), the homes a key can live in on this machine (environment variables and known tool configuration files, named but never read for their value until the person chooses), and abcd's own settings folder. The proposed promise (see the review findings below: it is a discipline, not a property): connect never prints, echoes or records the key, and the guided path never asks for it; the person stores it, and the binary reads it from the chosen home.

Routed out: whether a request is judged against a model's published size before it is sent is the adapter's question, not connect's; it is captured as its own record, a follow-up of the streaming fix iss-2610030931521214. <!-- record-lint: forward-looking -->

Typed links: refines itd-2609081951381895 (its connect verb gains a guided path); the arrow-key list the product thinker wants later is routed to itd-2610030810370060 (abcd's interviews in a plain Terminal), with model choice as its first example (the product thinker's routing, 2026-10-03); the questions it asks follow itd-2610030810350727 (one question layout).

## Mechanism

We expect a person to connect a service listing hundreds of models without knowing an exact model name because abcd reads the names from the service's own list and the person only types part of one to narrow it; a person who still has to type a full model name for a service that publishes its list shows the claim wrong.

_Proposed by the facilitator from decisions 1 to 8; confirmed by the product thinker, 2026-10-03._

## Scope Conditions

- Services that publish an OpenAI-compatible model list; a service that publishes none falls back to typing the name (decision 5). <!-- cond: cond-2610031241487546 -->
- A person who can open a terminal on the same machine to paste the printed command; the session never finishes the setup itself (decisions 1 and 2). <!-- cond: cond-2610031241484784 -->
- Suggestions only for a person with an existing connection; a first-time user sees the search alone (decision 6). <!-- cond: cond-2610031241488563 -->
- A key that reaches abcd through its own hidden input in that terminal step; a key saved by hand in the computer's password store is not adopted (decisions 2 and 7). <!-- cond: cond-2610031241482561 -->

_Proposed by the facilitator from decisions 1 to 8; confirmed by the product thinker, 2026-10-03._

## Acceptance Criteria

_Proposed by the facilitator from decisions 1 to 8 and the design review's proposed criteria that survive them; each is unconfirmed until walked with its addressee. Review criterion C3 (it let the session run the setup for some homes) is rewritten to decision 1's always-print rule; C5 is rewritten to decision 2's single terminal step, with no listing taken as the test; C8 (a listed model size) is dropped, because decision 8 routes the size question out. How many suggestions a question shows is owed at planning; G2 proposes at most three._

- G1 (technical facilitator; CONFIRMED 2026-10-03) Given a service address, when the guided path offers to look up its models, then one question first shows the scheme and host, and only on a yes does a request go out, carrying no key, following no redirect, and keeping the adapter's short timeout and size cap (example: a stand-in service that answers with a redirect is refused and never sees a key header); a go test asserts each.
- G2 (product thinker; CONFIRMED 2026-10-03) Given a person whose existing connections use two models a service of 300 lists, when the model question is asked, then it suggests those two (at most three) and typing part of a name narrows the full list without a second request, while a first-time user sees no suggestions, only the search (example: typing "coder" narrows 300 names to the few that contain it); a go test asserts both cases.
- G3 (technical facilitator; CONFIRMED 2026-10-03) Given any guided connect run inside a session, when it ends, then it has printed exactly one command and written nothing: abcd's settings folder and every key home are byte-identical before and after (example: a run that reaches the end with a keyless local service still only prints the command); a go test asserts it.
- G4 (product thinker; CONFIRMED 2026-10-03) Given the values worked out with the person, when the command is printed, then the command and every path it will write are shown in full before the person runs it, and no setting skips that (example: the command is followed by the connection file it writes in abcd's settings folder); a go test asserts the output, and a dated receipt from a Claude Code session shows it on screen.
- G5 (technical facilitator; CONFIRMED 2026-10-03) Given a stand-in service that refuses to list its models without the key, when the printed command runs in a terminal, then the key is read from hidden input, the list is fetched with the key held in memory, the person picks a model, a real completion verifies it, and only then are both writes made, while a failed completion writes nothing and no listing counts as the test (example: a stand-in that lists with the key but fails the completion leaves no connection and no stored key); a go test asserts both paths.
- G6 (technical facilitator; CONFIRMED 2026-10-03) Given a known key value, when the guided path and its printed command run end to end, then the value appears in no question, answer, printed command, error or record (example: a canary key string is searched for in every output and file written, and found only in the chosen key home); a go test asserts it.
- G7 (product thinker; CONFIRMED 2026-10-03) Given a service whose model list request fails or returns nothing usable, when the model question is reached, then the person is asked to type the model name and told the service publishes no list (example: a stand-in answering not found on its model list); a go test asserts the fallback and its reason.
- G8 (product thinker; CONFIRMED 2026-10-03) Given a key the person saved by hand in the computer's password store, when the guided path asks where the key lives, then it does not offer that item, and the printed command takes the key again through abcd's hidden input (example: an item named for the service in the password store is not listed as a choice); a go test asserts it.

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
9. 2026-10-03, the product thinker: the revised press release confirmed as written.

## Open Questions

None open: the interview of 2026-10-03 answered them (decisions 1 to 8). Owed at planning: the press release and the acceptance criteria rewritten to the decisions (the press release no longer says the connection may be set up from the session), for the product thinker to confirm; the Mechanism and Scope Conditions; and how many suggestions a question shows (at most three, under the layout's limits).

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-8c051b55af1b -->
Fidelity review — receipt rcp-8c051b55af1b (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:bf4699f8dfbfc48a908c669c391e233decc4ef5004577abcf85cd93eb4cd5aeb
Input attestations: diff:5bf4aa93b^1..5bf4aa93b (PR #791, connect step 1)@sha256:b3a8df593ac475c1276b0e17f12df4e903ccc5065c1ac5d1941e5cc00bb47050; diff:93e7b7caa^1..93e7b7caa (PR #801, connect step 2)@sha256:a27f29cafda4286312a6a776f9c873009cb8be6c98110b5989d0d610b9c15368; diff:81548efe5^1..81548efe5 (PR #807, connect step 3 and the close)@sha256:ffaddf46964672ccd115d76287ecd0d7e79d4e33f1f0f557106bf5d8724f3c89; test-run:go test ./internal/adapter/openaiapi ./internal/core/oracle ./internal/core/credential ./internal/core/question ./internal/term/... ./internal/surface/cli/ask; go test -run 'TestGuide|TestConnect|TestKeyCanary|TestPicked|TestReadKey|TestHiddenKey|TestActAsConnect|TestPickQuestion' ./internal/surface/cli at 81548efe5: all ok@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The look-up question shows scheme and host and sends nothing until a yes; the guide lists with a keyless client; the adapter sets Authorization only when a key is held, fails on the pinned redirect, bounds the call by ListTimeout and the body by MaxResponseBytes+1; TestGuideAsksBeforeLookingUp (redirect target sees no request, redirecting service sees an empty Authorization) and TestListModelsSendsNoKeyAndFollowsNoRedirect / TestListModelsBoundsTimeAndSize assert each, and pass at HEAD.
  evidence: internal/core/oracle/connect_guide.go:447 — "abcd can ask %s://%s for the list of models it offers. The request carries no key, follows no redirect, and gives up after %d seconds."
  evidence: internal/core/oracle/connect_guide.go:619 — "client, err := openaiapi.New(base, "")"
  evidence: internal/adapter/openaiapi/models.go:87 — "if c.key != "" { req.Header.Set("Authorization", "Bearer "+c.key) }"
  evidence: internal/adapter/openaiapi/models.go:190 — "case errors.Is(err, errRedirect): return c.listFail(false, "answered with a redirect, which abcd never follows")"
  evidence: internal/adapter/openaiapi/models.go:80 — "lctx, cancel := context.WithTimeoutCause(ctx, c.listWait, errListTimeout)"
  evidence: internal/adapter/openaiapi/models.go:106 — "io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))"
  evidence: internal/core/oracle/connect_guide_test.go:148 — "func TestGuideAsksBeforeLookingUp(t *testing.T)"
  evidence: internal/adapter/openaiapi/models_test.go:56 — "func TestListModelsSendsNoKeyAndFollowsNoRedirect(t *testing.T)"
  evidence: internal/adapter/openaiapi/models_test.go:112 — "func TestListModelsBoundsTimeAndSize(t *testing.T)"
- ac-2 — MET_WITH_CONCERNS: suggestions() draws from routes then provider allowlists, exact-id matches only, capped at maxSuggestions = 3; TestGuideSuggestsTheModelsAlreadyUsed asserts two of 300 are offered and a first-time user gets the typed part alone; TestGuideNarrowsWithoutASecondRequest asserts 'CODER' narrows to the four matches with one request across every turn. Concern (signed off 2026-10-04): the resume object carries at most MaxCarriedBytes (32 KiB) of ids, so for a list past that budget 'narrows the full list' holds only over the carried prefix and a fragment matching none carried asks for the model's full name instead of narrowing; a 300-model list is carried whole, so the criterion's own example holds.
  evidence: internal/core/oracle/connect_guide.go:53 — "maxSuggestions = 3"
  evidence: internal/core/oracle/connect_guide.go:778 — "func (g *guide) suggestions(listed map[string]bool) []string"
  evidence: internal/core/oracle/connect_guide_test.go:212 — "func TestGuideSuggestsTheModelsAlreadyUsed(t *testing.T)"
  evidence: internal/core/oracle/connect_guide_test.go:236 — "func TestGuideNarrowsWithoutASecondRequest(t *testing.T)"
  evidence: internal/core/oracle/connect_guide.go:69 — "const MaxCarriedBytes = 32 << 10"
  evidence: internal/core/oracle/connect_guide.go:649 — "func carry(ids []string) (kept []string, more int)"
  evidence: .abcd/work/DECISIONS.md:2664 — "the guided connect carries at most 32 KiB of listed ids between turns, with the count not carried"
  evidence: internal/surface/cli/ahoy_connect_guide_test.go:491 — "t.Run("300", func(t *testing.T)"
- ac-3 — MET: Guide never calls Connect (only the write-free checkConnect) and ends in guideCommand; TestGuideWritesNothing snapshots ~/.abcd, the index, the abcd credential file and the fake keychain before and after a run to the end for the none, abcd and keychain homes against a keyless loopback stand-in, asserts byte-identity, zero keychain reaches and exactly one "command" printed; passes at HEAD.
  evidence: internal/core/credential/guide_test.go:133 — "func TestGuideWritesNothing(t *testing.T)"
  evidence: internal/core/credential/guide_test.go:155 — "if after := snapshot(t, roots.Home, roots.Repo, chain); !reflect.DeepEqual(before, after)"
  evidence: internal/core/credential/guide_test.go:148 — "strings.Count(strings.Join(said, "\n"), `"command":`) != 1"
  evidence: internal/core/oracle/connect_guide.go:205 — "if err := checkConnect(&probe, false); err != nil"
  evidence: internal/core/oracle/connect_guide.go:869 — "func guideCommand(provider, base, model, home, env string, picks bool) (*GuideDone, error)"
- ac-4 — MET_WITH_CONCERNS: The go-test half is demonstrated: TestGuideShowsTheCommandAndEveryPath asserts for none/abcd/external that the text form opens with the command, 'When it runs, this command writes:' and each path, and that running the printed command reports exactly those writes; TestGuideKeychainWritesAreTheRunsWrites covers the keychain home; TestGuideCannotRun refuses --guide with --home/--model/--key/--env, so no setting skips it. The receipt half is present at the path spec:439 names, .abcd/.work.local/logs/connect-guide-2026-10-04.md, carrying every element spec:439 lists (date, commit 81548efe5 = the shipped HEAD, Claude Code 2.1.289, each turn, every stand-in request, and the done turn as shown on screen), and its on-screen text, question ids, provider name and write order match the code at HEAD. Concerns, both named in the receipt itself: the answers were the Claude Code session's own, not a person's, so 'the values worked out with the person' is shown by the mechanism rather than by a person's run; and the receipt is a replacement produced after the close by the shipping session, in the gitignored tier, replacing one lost with a removed worktree, so it is not reproducible from the repository and the auditor did not observe the run.
  evidence: internal/surface/cli/ahoy_connect_guide_test.go:168 — "func TestGuideShowsTheCommandAndEveryPath(t *testing.T)"
  evidence: internal/surface/cli/ahoy_connect_guide_test.go:210 — "want := append([]string{done.Command, "When it runs, this command writes:"}"
  evidence: internal/core/credential/guide_test.go:222 — "func TestGuideKeychainWritesAreTheRunsWrites(t *testing.T)"
  evidence: internal/surface/cli/ahoy_connect_guide_test.go:145 — "{"--home", "none"}, {"--home", "abcd"}, {"--model", "vendor/coder"}, {"--key", "k"}, {"--env", "X_API_KEY"}"
  evidence: internal/core/oracle/connect_guide.go:888 — "writes := append([]string{}, credential.WritesFor(home, req.KeyName)...)"
  evidence: .abcd/development/specs/closed/spc-2610031241482088-connecting-a-model-service-from-inside.md:439 — "The dated receipt `.abcd/.work.local/logs/connect-guide-<yyyy-mm-dd>.md`"
  evidence: .abcd/.work.local/logs/connect-guide-2026-10-04.md:3 — "Date: 2026-10-04 / Commit: 81548efe55dd5cedba5b6b4dd22eb2355be47d1b / Claude Code: 2.1.289"
  evidence: .abcd/.work.local/logs/connect-guide-2026-10-04.md:24 — "abcd ahoy connect local --base-url http://127.0.0.1:48731/v1 --model example/coder-small --home abcd / When it runs, this command writes: / ~/.abcd/credentials.json / ~/.abcd/config.json"
  evidence: .abcd/.work.local/logs/connect-guide-2026-10-04.md:6 — "The answers are the session's own, not a person's."
  evidence: internal/surface/cli/ahoy_connect_guide.go:183 — "fmt.Fprintln(w, "When it runs, this command writes:")"
  evidence: internal/surface/cli/ahoy_connect_guide.go:193 — "Paste it into a terminal on this machine; nothing is set up until it runs there."
  evidence: internal/core/oracle/connect_guide.go:76 — "GuideQNarrow = "model-narrow""
- ac-5 — MET: readConnectKey prints one stderr line then reads through term.ReadHidden (echo off inside a RawSession); Connect with no model and Pick set lists through pickModel with the held key, hands the ids to Pick, then Walk verifies with a real completion and only on success stores the key and writes the provider block. The stand-in answers a keyless list 401 and lists with the key; TestConnectPicksAfterAKeyedListing asserts the keyed list, the pick, the completion to the picked model and both writes; TestConnectWritesNothingWhenTheCompletionFails asserts no provider block and no stored key; TestConnectReadsTheKeyHiddenAtARealTerminal and TestHiddenKeyRestoresTerminalOnInterrupt hold the hidden read on a real pty. All pass at HEAD.
  evidence: internal/surface/cli/ahoy_connect.go:308 — "Paste the key for %s and press Enter. It is not shown."
  evidence: internal/term/hidden.go:34 — "func ReadHidden(in *os.File, out io.Writer) (line string, err error)"
  evidence: internal/core/oracle/connect.go:134 — "if picking { m, err := pickModel(ctx, req, cfg.denylist, opts)"
  evidence: internal/core/oracle/connect.go:153 — "walked, err := credential.Walk(ctx, req.Roots.Home, svc, credential.Choice{Home: req.Home, Value: req.Key, Pointer: req.Pointer})"
  evidence: internal/core/oracle/connect.go:148 — "the verification call failed, so nothing was written"
  evidence: internal/surface/cli/ahoy_connect_terminal_test.go:100 — "if auth == "" { echo(http.StatusUnauthorized)"
  evidence: internal/surface/cli/ahoy_connect_terminal_test.go:237 — "func TestConnectPicksAfterAKeyedListing(t *testing.T)"
  evidence: internal/surface/cli/ahoy_connect_terminal_test.go:273 — "func TestConnectWritesNothingWhenTheCompletionFails(t *testing.T)"
  evidence: internal/surface/cli/ahoy_connect_pty_test.go:122 — "func TestConnectReadsTheKeyHiddenAtARealTerminal(t *testing.T)"
  evidence: internal/surface/cli/ahoy_connect_pty_test.go:148 — "func TestHiddenKeyRestoresTerminalOnInterrupt(t *testing.T)"
- ac-6 — MET_WITH_CONCERNS: TestKeyCanaryAppearsOnlyInItsHome runs the guide to its end and then the printed command with a canary key against a stand-in that echoes the Authorization header into error bodies and lists one id carrying the key; it searches every turn's JSON and text, the command's stdout, stderr and error, and every file under the fake HOME, and asserts the canary is found only in ~/.abcd/credentials.json (and nowhere when the completion fails); TestPickedConnectKeepsTheCanaryInItsHome holds the terminal half. Concern: the end-to-end canary covers the abcd home only; the keychain home, fakeable only inside the credential package, has no end-to-end canary search.
  evidence: internal/surface/cli/ahoy_connect_guide_test.go:289 — "func TestKeyCanaryAppearsOnlyInItsHome(t *testing.T)"
  evidence: internal/surface/cli/ahoy_connect_guide_test.go:322 — "store := filepath.Join(os.Getenv("HOME"), ".abcd", "credentials.json")"
  evidence: internal/surface/cli/ahoy_connect_terminal_test.go:354 — "func TestPickedConnectKeepsTheCanaryInItsHome(t *testing.T)"
  evidence: internal/adapter/openaiapi/models.go:207 — "func (c *Client) listFail(needsKey bool, reason string) error"
  evidence: internal/adapter/openaiapi/models.go:151 — "if c.carriesKey(strings.Join(l.IDs, ""))"
- ac-7 — MET: A ListError that does not need a key puts the guide on the typed model question whose material opens with '< host> publishes no model list: it < reason>.'; TestGuideFallsBackToTyping asserts the fallback and reason for not found, an empty list and a non-list body, and TestListModelsNamesWhyThereIsNoList holds each reason at the adapter; both pass at HEAD.
  evidence: internal/core/oracle/connect_guide.go:485 — "why = fmt.Sprintf("%s publishes no model list: it %s.", host, listed.Reason)"
  evidence: internal/adapter/openaiapi/models.go:101 — "return Listing{}, c.listFail(false, "answered not found")"
  evidence: internal/core/oracle/connect_guide_test.go:366 — "func TestGuideFallsBackToTyping(t *testing.T)"
  evidence: internal/adapter/openaiapi/models_test.go:197 — "func TestListModelsNamesWhyThereIsNoList(t *testing.T)"
- ac-8 — MET: The home question offers exactly the three homes with the keychain's meaning 'You paste it hidden in your terminal'; the guide never searches the keychain or index. TestGuideNeverOffersAHandSavedItem seeds an item named for the service in the fake keychain, asserts the options are exactly external/abcd/keychain plus decide later, that no turn carries the item, zero keychain reaches, and that the command run without a key is refused rather than adopting the item; TestConnectReadsTheKeyHidden asserts the keychain home reads the key through the hidden reader. Both pass at HEAD.
  evidence: internal/core/oracle/connect_guide.go:542 — "{Value: KeyHomeKeychain, Label: "The system keychain", Meaning: "You paste it hidden in your terminal; the system keychain keeps it."}"
  evidence: internal/core/credential/guide_test.go:164 — "func TestGuideNeverOffersAHandSavedItem(t *testing.T)"
  evidence: internal/core/credential/guide_test.go:213 — "want it refused, the hand-saved item never adopted"
  evidence: internal/surface/cli/ahoy_connect_terminal_test.go:177 — "func TestConnectReadsTheKeyHidden(t *testing.T)"
  evidence: internal/surface/cli/ahoy_connect.go:223 — "if home == oracle.KeyHomeABCD || home == oracle.KeyHomeKeychain { key, err := readConnectKey(cmd, args[0])"

Gap audit:
- honoured:
  - Nothing is set up from inside the session: the guide always prints one command and writes nothing
    evidence: internal/core/credential/guide_test.go:133 — "func TestGuideWritesNothing(t *testing.T)"
    evidence: commands/ahoy.md:649 — "Never ask for the key, never pass it, and never run the"
  - The look-up runs only after one question showing scheme and host, with no key, no redirect, the adapter's bound and cap
    evidence: internal/core/oracle/connect_guide.go:447 — "abcd can ask %s://%s for the list of models it offers."
    evidence: internal/adapter/openaiapi/models_test.go:56 — "func TestListModelsSendsNoKeyAndFollowsNoRedirect(t *testing.T)"
  - A key-gated service is handled in one terminal step: hidden key, keyed list, pick, real completion, then both writes
    evidence: internal/core/oracle/connect.go:134 — "if picking { m, err := pickModel(ctx, req, cfg.denylist, opts)"
    evidence: internal/surface/cli/ahoy_connect_terminal_test.go:237 — "func TestConnectPicksAfterAKeyedListing(t *testing.T)"
  - A service that lists no models asks for the name to be typed and says why
    evidence: internal/core/oracle/connect_guide.go:485 — "publishes no model list: it %s."
  - Suggestions come from the person's existing connections; a first-time user sees the search alone
    evidence: internal/core/oracle/connect_guide_test.go:212 — "func TestGuideSuggestsTheModelsAlreadyUsed(t *testing.T)"
  - A hand-saved password-store item is never offered or adopted
    evidence: internal/core/credential/guide_test.go:164 — "func TestGuideNeverOffersAHandSavedItem(t *testing.T)"
  - The command and every path it writes are shown in full, from the same list the run reports
    evidence: internal/core/credential/walk.go:134 — "func WritesFor(home, name string) []string"
    evidence: internal/surface/cli/ahoy_connect_guide_test.go:168 — "func TestGuideShowsTheCommandAndEveryPath(t *testing.T)"
- diverged:
  - 'Typing part of a name narrows the full list' is delivered over a 32 KiB carried prefix; past that budget a fragment matching none carried asks for the model's full name rather than narrowing (signed off 2026-10-04)
    evidence: internal/core/oracle/connect_guide.go:69 — "const MaxCarriedBytes = 32 << 10"
    evidence: .abcd/work/DECISIONS.md:2664 — "the guided connect carries at most 32 KiB of listed ids between turns"
  - The end-to-end canary search (G6) is delivered for the abcd home only, not for the keychain home
    evidence: internal/surface/cli/ahoy_connect_guide_test.go:322 — "store := filepath.Join(os.Getenv("HOME"), ".abcd", "credentials.json")"
  - Suggestions match an existing connection's model only by exact id, so a differently spelled id on this service yields none
    evidence: internal/core/oracle/connect_guide.go:778 — "kept only where the service lists the id exactly (open question 3, decided (a)), at most three"
  - G4's dated Claude Code session receipt is a replacement made after the close by the shipping session with the session's own answers, not a person's, held in the gitignored tier; the original was lost with a removed worktree
    evidence: .abcd/.work.local/logs/connect-guide-2026-10-04.md:6 — "The answers are the session's own, not a person's."
    evidence: .abcd/development/specs/closed/spc-2610031241482088-connecting-a-model-service-from-inside.md:439 — "The dated receipt `.abcd/.work.local/logs/connect-guide-<yyyy-mm-dd>.md`"
- missing: (none)

Scope-condition dispositions:
- cond-2610031241487546 — survived: The look-up decodes the standard data[].id list and nothing else; every other shape or status becomes a ListError whose reason the typed fallback question carries, exactly as the condition assumed.
  evidence: internal/adapter/openaiapi/models.go:118 — "func (c *Client) decodeListing(raw []byte, keep func(string) bool) (Listing, error)"
  evidence: internal/core/oracle/connect_guide_test.go:366 — "func TestGuideFallsBackToTyping(t *testing.T)"
- cond-2610031241484784 — survived: The guide never calls Connect, refuses every flag that would run or write, ends in a printed command, and the page tells the agent never to run it; the setup finishes only when the person pastes the command in a terminal where the pick and hidden read need every stream to be a terminal.
  evidence: internal/core/credential/guide_test.go:133 — "func TestGuideWritesNothing(t *testing.T)"
  evidence: internal/surface/cli/ahoy_connect_guide_test.go:141 — "func TestGuideCannotRun(t *testing.T)"
  evidence: internal/surface/cli/ahoy_connect.go:209 — "if len(models) == 0 && !atTerminal"
  evidence: commands/ahoy.md:591 — "never runs that command, and no flag makes it"
- cond-2610031241488563 — narrowed: Suggestions are drawn only from a person's existing routes and provider allowlists, so a first-time user sees the typed search alone as assumed; but a model is suggested only where this service lists the same id exactly, and at most three are shown, so some people with existing connections also see no suggestion.
  narrowing: Holds for a person whose existing connections name an id this service lists exactly, at most three of them; an existing connection whose model this service spells differently yields no suggestion, so that person sees the search alone like a first-time user.
  evidence: internal/core/oracle/connect_guide.go:778 — "func (g *guide) suggestions(listed map[string]bool) []string"
  evidence: internal/core/oracle/connect_guide.go:53 — "maxSuggestions = 3"
  evidence: internal/core/oracle/connect_guide_test.go:212 — "func TestGuideSuggestsTheModelsAlreadyUsed(t *testing.T)"
- cond-2610031241482561 — survived: For the abcd and keychain homes the key reaches abcd through readConnectKey's hidden read in the terminal step and is never asked for by the guide; a hand-saved keychain item is neither offered nor adopted, the keyless run being refused. The external home's variable is a separate, named route the condition does not speak to and does not contradict.
  evidence: internal/surface/cli/ahoy_connect.go:302 — "func readConnectKey(cmd *cobra.Command, provider string) (string, error)"
  evidence: internal/core/credential/guide_test.go:164 — "func TestGuideNeverOffersAHandSavedItem(t *testing.T)"
  evidence: internal/surface/cli/ahoy_connect_pty_test.go:122 — "func TestConnectReadsTheKeyHiddenAtARealTerminal(t *testing.T)"
<!-- abcd-review-end receipt=rcp-8c051b55af1b -->

## Grounds

- pursued: connecting a service with hundreds of models takes one paste, with no hunting for exact model names (the product thinker, 2026-10-03).
