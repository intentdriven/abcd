# Adapters

Adapters are abcd's **central architectural model**, not peripheral plumbing. No
external tool is a hard dependency (adr-22): every capability abcd could take from
one is instead a **seam over the Go core** — a Go interface, a native default that
ships in the binary, and an optional external plug-in behind the same interface.
abcd runs fully with none of the external tools installed; a present tool is an
upgrade the seam keeps cheap, never a floor abcd stands on.

Consumers in `internal/core` depend on the **interface**, never on a vendor — they
consume "an oracle", "a transcript store", "a spec store", not "RepoPrompt" or
"specstory". The pattern is stated in
[`04-universal-patterns.md § 7`](04-universal-patterns.md#7-vendor-agnostic-adapters-with-environment-branching);
this file is the per-seam catalogue.

## The five capability seams

Each dropped hard dependency maps to exactly one seam under `internal/adapter/`:

| Seam (`internal/adapter/<seam>`) | Interface (what the core consumes) | Native default | Optional external plug-in(s) |
|---|---|---|---|
| **oracle** | hand a prompt to a model, receive a structured verdict/result | host-delegated LLM — abcd emits the prompt, the host's subagent dispatch runs it (adr-25) | `native` (abcd calls a local model), `cli` (e.g. `codex exec`), `api` (direct provider API), `mcp` (RepoPrompt / codex over MCP) |
| **history** | capture and read session transcripts | native local redacted transcript store — root-SHA-keyed, gitignored, redacted on capture (adr-29) | specstory capture source (imported over the same store) |
| **spec** | store and query specs/tasks + their dependency graph | native minimal store — directory-as-truth (adr-3) + dependency graph (adr-26) | the companion harness `ccpm`, read/written at the convention level (adr-24) |
| **run** | iterate ready work, gate each step on a receipt, enforce a safety guard | thin native Go loop (adr-27) | Claude Workflows, the companion harness's agent loop |
| **scanner** | scan content for secrets/PII, return findings | native secret/PII scan (built-in patterns) | gitleaks, Presidio, TruffleHog |

**Backend resolution.** `.abcd/config.json` → `<seam>.backend` selects the backend;
its default is the seam's native path (host-delegated for `oracle`, the native
store/loop/scan for the rest). A missing or misbehaving external backend degrades
to the native default rather than breaking abcd — each seam carries its own thin
capability contract. Adding a backend = implement the interface and register it in a
registry (the design target `internal/registry`, which does not exist yet);
consumers are untouched. Of the five, only `scanner` is a directory under `internal/adapter/` in the tree; `oracle`, `history`, `spec` and `run` are design targets, each introduced by the first intent that consumes it, and today their native paths live in `internal/core` (`oracle`, `history`, `spec`, `implement`). `internal/adapter/` also holds adapters that are not capability seams: `gitleaks`, `hosting`, `openaiapi`, `repoprompt` and `tailscale`. [`internal/README.md`](../../../../internal/README.md) § Planned seams is the gated list. See
[`03-configuration.md`](03-configuration.md) for the config schema.

**Oracle consumers.** `lifeboat-reviewer`, `press-release-composer`, and
`intent-auditor` all reach a model through the `oracle` seam. Host
delegation is the default; when an operator wires two oracle adapters for a
high-stakes review, the adapter layer offers the scoped-vs-broad,
asymmetric-trust guidance of adr-25 — advice, never a cascade the core imposes.

### The OpenAI-compatible API adapter — a provider serves only what it lists

The `api` oracle plug-in is `internal/adapter/openaiapi`, one client over the
chat-completions protocol that OpenRouter and a local OpenAI-compatible server
both speak, so a provider is configuration and never code (itd-2609081951381895).
The invariant it serves is adr-2609300107513982's: **a provider adapter serves
only the models it lists, its allowlist alone decides, and everything else runs
on the host.** abcd bundles no vendor denylist. `internal/core/oracle` enforces
the invariant before a client is ever built: a route to an unlisted model is
refused naming the list, and a listed model of any vendor is served. The
optional `oracle.denylist` the configuration writes refuses a listed model it
matches, naming the entry, and a reported model it matches discards the
answer. The configuration is in
[`03-configuration.md`](03-configuration.md#the-provider-adapters-keys).

The client's own guarantees are the network path's. The base URL is pinned per
provider block, plain HTTP is admitted only to this machine, and a redirect is
never followed, so a provider cannot move the key or the brief elsewhere. Every
response is bounded in size and every call in time. The key travels only as the
bearer header of a request to the pinned address. A provider's own text, its
error, the model it reports and the answer itself, is decoded (JSON escapes
undone, HTML character references resolved) and scrubbed of the key in every form
an encoder gives it (literal, JSON-, HTML- and URL-escaped, quoted) before it
reaches an error, a payload or a record, because a provider may echo what it was
sent; the error and the model are bounded and sanitised as well. An answer that
carries no key keeps the provider's bytes, and one that does is returned with the
key replaced, a JSON answer rendered again from its decoded values. A
setting the protocol does not take is refused before the call, and the answer is
judged by the caller's output contract, the one the host sub-agent's payload is
judged by. The request is the host's brief in the protocol's two roles: the
agent's prompt as the system message, the verb's request as the user message.

The call asks for a streamed answer and assembles the protocol's server-sent
events (one chunk per `data:` line, `data: [DONE]` at the end, keep-alive
comments read past) into what a single body would carry: the content, the model
reported and the finish reason, with the token usage when the server reports it
in a last chunk. A server that answers with one body instead is read as that. A
stream that closes before any finish reason or `[DONE]` was cut off, and is
refused rather than used; one that reports two models in one answer is refused,
because the denylist could not tell which one answered. The size bound holds
across the stream: one event, the assembled answer and the whole stream are
each bounded. Time is bounded three ways rather than by one deadline, because a
reasoning model's answer can run past ten minutes and a deadline on the whole
call abandons it while it is still arriving: a first-byte limit (five minutes,
for a local server reading a long prompt), an idle limit between reads once the
answer has begun (two minutes; a keep-alive counts as the server being alive),
and a total cap (thirty minutes), which a caller of the adapter may set for one
call and no route setting carries. A stream is also bounded between its events
(ten minutes, the window the record saw a gateway hold a request that had sent
nothing): a keep-alive is a read but not an answer, so a server that sends only
keep-alives is refused at that limit and the refusal says so. A failed status is
reported on the status, its body waited for half a minute at most and quoted
only when it arrived whole: a body cut off by that wait or by its size bound
can end inside an echoed key the scrub cannot recognise, so it is named as cut
off instead. A caller's own earlier deadline is named as the caller's, not as
the adapter's cap. Whichever limit fires, and a cancelled caller, closes the connection, so a
server still generating sees the client go away and can stop. The setup's verification call is one short exchange and
keeps its own two-minute bound.

The key is resolved by name through the credential store
(`internal/core/credential`, `Store(home).Resolve`), the one reader, from
whichever of its three homes the person chose; the adapter reads no file and no
store of its own, and a key that resolves to nothing refuses before any call,
naming `abcd ahoy credential <name>`. A call's record names the credential it
used, never the key. The one environment it
honours is the HTTP stack's: the standard proxy variables (`HTTPS_PROXY`,
`NO_PROXY`) and the platform's trust roots. An https call through a proxy is a
tunnel, so the key and the brief stay inside TLS, and a call to this machine is
never proxied. Its
connection (`oracle.Connections`) carries the provider's allowlist, the
settings the adapter accepts and the model each role pointed at it asks for, and
the model tier's `Resolve` holds a provider leg to them before the step runs
(spc-2609251028149555): a connection whose allowlist lists no model admits no
route; the model the agent's `oracle.roles.<agent>` points at on the connection
must be on its allowlist, or the leg is refused naming the agent, the
connection, the model, the allowlist and the remedy; a leg to a connection the
agent's role does not point at names no model and is refused, naming the
`oracle.roles.<agent>` setting to add; a merged setting outside the
accepted set is refused naming the setting, where it was set and what the
adapter accepts, never dropped; and on a connection that holds a key and that
the person did not type with `--route`, a setting the repository's routing row
names is refused, never dropped, naming it and `~/.abcd.noindex/oracle-routing.json` as
where to move it, because only the person's own machine configuration shapes a
call that spends their key. A keyed leg the person typed with `--route` is
theirs, and there the repository row's settings merge within the accepted set. `model` is the adapter's own and is never a setting, so no setting can
choose a model past the allowlist. A provider claims no
tier: it is reached by a role or a judgement type pointed at it, never by a tier
alone. An agent whose role is pointed at a provider resolves to that provider with
no `--route`, whatever tier the routing tables name; a `--route` governs the step
over it for that run. The core's dispatch (`APIConfig.Dispatch`) sends a step on a
provider leg through the adapter with the host's brief and the settings as sent,
and takes the model and the key's reach from the machine's configuration again,
never from the route alone: a provider that holds a key is reached only through a
route set on this machine. Every refusal comes before the provider is contacted and
names the setting to change. A provider that could not be reached at all, so that
nothing was sent, leaves the step to the harness, and the route records the
connection tried and the reason (`Route.FellBack`); a provider that answered, or
took the brief and did not answer, is a failure, never a fallback. A provider
that holds a key takes only a **self-contained** agent (the product thinker's
ruling DR5 of 2026-09-29): a provider call carries no tools, so an agent that
reads files cannot read them there, and only an agent whose request carries all
its input is admitted, from a list compiled into the binary, default deny. The
list is the four cold-reading positions, each handed one assembled bundle; every
other agent pointed at a keyed provider is refused before any call, naming the
rule and the person's override, `oracle.bundled_context_providers`
(configuration chapter). A provider whose block names no key (a local server)
spends nothing of the person's and is outside the rule. The delegating verbs
read the machine's provider configuration when they resolve a route, and a verb
whose route is on a provider sends the step there itself: `intent audit`,
`intent consistency`, `intent audit --owed` (its head), `launch ship` and `spec
close`'s review re-emit send the request they emitted with the agent's own
prompt, and `reading ingest --dispatch <rdg-N>` sends a parked run's bundle with
the position's definition. The verb runs the answer through its own ingest, and
the receipt names the provider as `connection_used` with its call record
(`provider_call`: provider, model asked, model reported, credential name), so a
host reading such a receipt knows the step already ran. The call record carries
both sides of the call (itd-166), built by the dispatch from the request it sent
and the response it read, never by asking the model about itself: the asked side
beside the receipt's `settings_sent` (the SHA-256 of the rendered messages,
never their text, and `adapter_added`, the request field the adapter sent on
its own, `stream_options.include_usage`), and the served side as the response
names it (`served_provider`, the upstream provider an aggregator names;
`generation_id`; `finish_reason`; `usage`, with prompt, completion, reasoning,
cache-read and cache-write tokens; and `cost`, in the provider's credits). Every
string the provider supplied is scrubbed of the key, cleaned and bounded as the
reported model is, and a usage member of an unexpected shape is a detail not
reported, never a refused answer: each count it does not carry readably is
listed as not reported on its own, and a later usage chunk that reads fewer
fields than an earlier one does not replace it. `unpinned` lists each field of the fixed asked
and served lists the call could not pin, with its reason: `not_reported` (the
response does not carry it), `self_reported`, `harness_computed`,
`not_configured` (no setting or pin names it; a route to this machine lists the
weight file's fingerprint, its quantisation and the runtime's version so unless
a local pin names them) or `not_verified` (abcd configured the value and the
record carries it, but nothing on the route shows it reached the provider). A
route to this machine also lists the served local fields
(`served.weights_sha256`, `served.quantisation`, `served.runtime_version`,
`served.context_length`, `served.kv_cache_type`, `served.tool_calls`): each is
`self_reported` when the server's probe read it, since it is the server's own
word, and `not_reported` otherwise, except `served.kv_cache_type`, which no
runtime reports: on a pinned route it is always `not_verified`, since the
pin's cache type is the person's declaration and nothing on the route
confirms it. A model a block on this machine pins
(`oracle.api.<provider>.pins`, a local pin, iss-2610161734428291;
[`03-configuration.md`](03-configuration.md)) is probed before it is used,
through the runtime's own endpoints: a call reuses a report the pin held for
a minute at most, a runner's route is probed before every run, and a report
the pin refused is never reused, so a server put right holds the route at its
next use. The record carries the pin as `local_pin` and what the server
reported as `local_served`: a pinned field the server confirmed is no longer
listed, one it does not report (llama-server and mlx_lm.server report no
digest) is `not_verified`, and the pin's context is the asked context. A field
the server reports otherwise refuses the call before the brief is sent; on
llama-server, which reports no digest, the model file it has loaded is held to
the pin's `weights_file`. Recording adds no request to the ones the call
makes. A model its provider block pins to one OpenRouter provider
(`oracle.api.<provider>.pins`, iss-2610161734413178) is sent with OpenRouter's
provider routing object (one provider in `order`, `allow_fallbacks: false`,
`require_parameters: true`, `data_collection: "deny"`, one quantisation), and
the record carries it as `provider_pin` with the model's `canonical_slug`, so
the provider pin is no longer listed as not configured. The served side can
also be read from OpenRouter's generation record (`GET
/api/v1/generation?id=`), but that is one extra request for every call, so it
is made only when the person turns it on for the provider in their own
machine's configuration (`oracle.api.<provider>.read_generation: true`,
OpenRouter's block alone, never a repository's;
[`03-configuration.md`](03-configuration.md); itd-2610161733316518 decision
12) and never otherwise: `generation_readback` says `not_asked`, `read` or
`failed`. Read, it replaces the served provider and the cost with the
record's own, and `attempts` lists every provider OpenRouter tried, in order,
so a fallback shows as an attempt before the one that served. The record's
native token counts come from the serving model's own tokenizer and the
response's from another, so they never mix: when the record carries all four
of its counts (prompt, completion, reasoning, cache-read) they replace the
response's all together, and the cache-write count, which the record does not
carry, is then listed as not reported; a record missing any of the four
leaves the response's counts as they were. A null is a count missing, never
a count of zero (OpenRouter answers null for a count the model has none of,
the reasoning count of a model that does not reason, say), and so is a null
cost. The record says whose counts it holds in `usage_source`: `native` when
the record's four replaced the response's, `response` when the response's
were kept; it is absent when no record was read. The cost is taken from the
record whenever it carries one. The setup's verification call (`ahoy
connect`, fresh or verifying a configured provider's key again) never makes
the readback, whatever `read_generation` says: its answer is discarded. It is made with the key the
call used, resolved once by name, on the same pinned client (no redirect, the
body bounded), and bounded in time as a whole (twenty seconds, a not-found
asked again after a short doubling wait, since OpenRouter records a
generation a moment after its answer ends); a failure, a provider that is not
OpenRouter or an answer that named no generation id leaves the response's own
served side, says `failed` with the reason in `generation_readback_error`, and
never fails the call. Every string the record carries is scrubbed, cleaned and
bounded as the response's are. On the harness leg the receipt's own `unpinned` lists the payload's
`model_reported` as self-reported and every other field as not configured or not
reported, since the host reports none of it. DR5 is checked before a
verb writes anything (`APIConfig.Admitted`), and again by the dispatch. An
ingest handed a payload the host produced while its agent is routed to a
provider refuses it, since the receipt would name work the provider never did;
`--route <agent>=host-decides` keeps one run on the harness. The four disembark
agents read the packed lifeboat and no verb builds a request carrying it, so none
is sent to a provider. No test reaches a real provider: the client and every
dispatching verb are exercised end to end against a fake on the loopback
address, and the key is built at run time.

### Reviewer routes — the command-line runner, and RepoPrompt over MCP

A build lane's ruthless and security reviews, which the validate stage hands
to the host's own agents by default, can be routed elsewhere two ways. Each is
opt-in, each is configured on the person's own machine, and each hands a
review it does not return back to the host with a fallback receipt, so no
review is skipped.

- **The command-line runner** (`internal/core/runner`, itd-2609201916056194)
  starts a model CLI, `claude` or `opencode`, as a subprocess for a role that
  `roles.<role>.runner` routes to it. The opt-in is that key in
  `~/.abcd.noindex/config.json`; the route, the sealing and the fallback are in
  [`34-build.md`](../04-surfaces/34-build.md). A claude launch whose role's
  contract grants the shell carries the guard as the harness's permission
  prompt tool, `abcd guard approve` over MCP on stdio, set on that launch's
  command line ([`17-guard.md`](../04-surfaces/17-guard.md#answering-a-headless-agents-prompts)).
  A run records both sides
  (itd-166) in the same vocabulary as the provider adapter's call record: the
  model the runner passed and the prompt's SHA-256, beside what the harness's
  events report. The claude CLI's result event gives the token counts, the
  stop reason, the cost and the models its per-model usage names, and its init
  event the served model and the harness's version; opencode's run mode names
  no model, so its served model is empty and listed as not reported, never the
  asked one, and its step-finish events' tokens and cost are summed, a floor,
  since a sub-agent's steps are not among them. Either harness computes the
  cost from its own price table, so it is listed as `harness_computed`. A
  count is read only when present and not negative, and a cost only when it
  is zero or more and finite, the rule the provider adapter holds its usage
  member to; a summed count that would overflow, or a summed cost that stops
  being finite, is listed as not reported rather than recorded.
  Besides the reviewers and the
  implementer, the drain's judge, `drain-judge`, takes the same route
  ([`35-drain.md`](../04-surfaces/35-drain.md#the-judgement-before-a-lane-opens)),
  to the claude runner alone, in a private directory of its own outside every
  working tree, granted reading that directory and writing its answer file
  there alone and refused the checkout, each rule naming its real absolute
  path.
  A role routed
  through opencode to a model the machine pins to one OpenRouter provider
  (iss-2610161734413178) runs against a sealed configuration abcd writes for
  that run alone, carrying the pin under the model's options
  (`provider.<name>.models.<model>.options.provider`), in a directory made for
  the run under the system's temporary directory (0700, the file 0600, refused
  when it resolves inside the repository or its checkout, and removed when the
  run ends). opencode is pointed at it with `OPENCODE_CONFIG`, which opencode
  1.18.31 merges over the person's own global configuration without writing
  it. After it, opencode merges the project's configuration and each
  `.opencode` directory's `opencode.json` (both turned off by the seal), then
  `~/.opencode/opencode.json`, then the `opencode.json` in
  `OPENCODE_CONFIG_DIR`, then `OPENCODE_CONFIG_CONTENT`, then an
  organisation's remote or managed configuration, which is the person's own
  and still applies. A person who points opencode at a configuration of their
  own through `OPENCODE_CONFIG`, `OPENCODE_CONFIG_DIR` or
  `OPENCODE_CONFIG_CONTENT` (their key through `{env:}`, permission denies,
  MCP servers) would lose it to the sealed file or have it override the pin,
  and abcd never changes how the person's tools work, so a pinned run is
  refused before anything is written or launched while any of the three is
  set and not empty, naming the variable, never its value: unset it for
  pinned runs, or route the role to a model without a pin. An unpinned run
  passes all three through as they are. `~/.opencode/opencode.json` is not
  refused, since refusing it would block every pinned run of anyone who
  keeps one, so a routing entry for the pinned model there overrides the
  pin; this is one reason the pin is recorded as `not_verified` on this
  path. The route records
  the pin (`provider_pin`), the configuration's SHA-256
  (`pin_config_sha256`) and opencode's version, asked with `--version` under
  the same seal. opencode holds the key on this path and abcd never reads it,
  so the generation readback is not made there, whatever `read_generation`
  says: the served provider and the provider's own counts stay not reported,
  and nothing shows the pin reached OpenRouter, so the provider pin is
  listed as `not_verified` beside the pin the record carries: configured,
  not verified on the wire. A runner route whose provider is a block on this
  machine says so (`local: true`) and lists the local fields as the
  adapter's route does (iss-2610171545099084): not configured without a
  local pin. A role routed
  through opencode to a model a local pin names is refused on the same three
  variables, then the server is probed and held to tool calls and a context
  it can say it serves (every role a runner runs uses tools), and only then
  is a sealed configuration written, carrying the provider's `baseURL` alone
  so opencode reaches the server the probe read. opencode pins no served
  context on any of the three runtimes (its `limit.context` is its own
  reckoning and needs `limit.output` beside it, which would replace the
  person's), so the probe is what confirms the context. The route records
  `local_pin`, `local_served`, the configuration's SHA-256 and opencode's
  version. It lists the asked local fields and the context as `not_verified`
  even where the probe confirmed them: opencode merges the person's
  `~/.opencode/opencode.json` after the sealed file, and an entry there can
  point the provider at another server than the one probed, as it can
  override a provider pin. The claude runner holds neither pin, so a role
  routed through it to a model the machine pins is refused before launch,
  naming `runner.opencode.model` as the route that holds it.
- **RepoPrompt over MCP** (`internal/adapter/repoprompt`, itd-6) sends the
  review to the RepoPrompt the person runs, which answers with whichever model
  the person configured in it. The opt-in is `oracle.review = rp` in the
  repository's `.abcd/config.json` or the machine's
  `~/.abcd.noindex/config.json` (the repository's file wins a key both set),
  with the MCP server's address, `oracle.rp.url`, in the machine's file alone
  ([`03-configuration.md`](03-configuration.md#the-reviewer-routes-keys)).
  Unset, or set to `host`, every review runs on the host exactly as before.

The RepoPrompt adapter speaks MCP's Streamable HTTP transport with the
standard library, no MCP dependency: one session per review (initialize, the
initialized notification, `tools/list`, `tools/call`, and a closing DELETE),
the request sent as the `message` of the configured tool (`oracle_send`
unless `oracle.rp.tool` names another), with `new_chat: true` when the tool
declares it, so each reviewer is fresh. It connects and nothing else: it
never starts RepoPrompt or any process, never reads or writes RepoPrompt's
configuration, and never looks for a server, and a test holds its package to
importing no process, file or environment API. The address is admitted only
on this machine (localhost or a loopback address, no credentials, query or
fragment), only a loopback address is dialled whatever a host name resolves
to, the connection is never proxied, and a redirect is never followed, so a
review request leaves the machine through no other door. RepoPrompt's MCP
server must therefore answer over HTTP at that address; abcd starts no bridge
to a server that speaks only stdio. The set-up is bounded to ten
seconds, the call to thirty minutes and an answer to 8 MiB, and a failure
names its kind (unreachable, timeout, refused, unparsable, failed) in abcd's
own words, never the server's.

The loop renders the request beside the reviewer's brief
(`validate/round-<n>/<role>/request.md`): the reviewer's agent definition, the
lane's brief with the record it was rendered from, the implementer's report
and each earlier fix round's, and the lane's own changes whole as git's own
diff (no external diff program, textconv or colour; at most 512 KiB), since the
route reads no file of the checkout. The answer is the reviewer's whole
return: the loop writes it to the return path the host's agent would have
written, under the run's lock and only while the review awaits it and nothing
stands there, and verifies and records it through the same receipt, so a
review the route ran differs from a host-run one in its `route` alone (`asked`
and `ran` both `rp`); a return the host's agent wrote first stands, and the
route's answer is kept aside beside it. The intent-auditor stays on the host
(reviews, not audits), and a reviewer the machine routes to a command-line
runner keeps its runner. A review the route does not return (no server named
on this machine, nothing listening, a bound run out, the server's error, an
answer that is not the protocol's, a return with no verdict the loop can record
or larger than a return may be, a request that could not be composed, or a
step interrupted while the route worked) is handed to the host's own agent
with the same brief and return path, and one fallback receipt (the role,
`asked: rp`, the reason, abcd's detail, `ran: host`) is recorded in the run's
state and record, which `implement status` and `implement record` count per
route and per role. The route is reached outside the run's lock, as a runner
is. No test reaches a real RepoPrompt: every one runs against an MCP server the
test owns on the loopback address.

### RepoPrompt oracle adapter — `dev-sync reviews` harvesting

This harvesting is a design target, separate from the reviewer route above:
`internal/adapter/repoprompt` reads no file, and nothing here is built. When it
is wired, `dev-sync reviews` harvests **ad-hoc oracle reviews not tied to a spec** from two
RepoPrompt-local sources (spec-tied reviews go straight to the native spec
review store and are never swept here). Vendor filesystem layout lives here with
the adapter, not in the core config brief:

- **Chat store** — `~/Library/Application Support/RepoPrompt/Workspaces/Workspace-<project>-<UUID>/Chats/ChatSession-*.json` (plain JSON, well-structured).
- **Prompt exports** — RepoPrompt's `export_response: true` writes ad-hoc reviews to `<cwd>/prompt-exports/` with a **hardcoded, non-configurable path**, which ahoy redirects into `~/.abcd.noindex/history/<root-sha>/prompt-exports/` via the `<repo>/prompt-exports` symlink.

**Workspace matching.** One project may map to multiple RepoPrompt workspaces —
match by content of `workspace.json` (RepoPrompt's own on-disk workspace file),
not by name.

**Stability.** Vendor JSON schemas may change between RepoPrompt releases. The
adapter probes defensively, version-stamps in `_provenance.json`, and on a parse
failure logs a warning and falls back to the existing
`.abcd/work/reviews/<YYYY-MM-DD>-<scope>/*.md` (the charter grammar — see `.abcd/work/reviews/README.md`) — never losing what was already synced.

**Privacy.** Filter strictly by workspace → project-path match before reading any
chat content.

**Workspace-state pull (itd-7, distinct from reviews harvesting).** The same
opt-in adapter pulls RepoPrompt's own workspace state into `.abcd/rp/workspace.json`.
It walks `~/Library/Application Support/RepoPrompt/Workspaces/`, parses each
workspace's root-path field, and matches against `git rev-parse --show-toplevel`
(multi-match → most-recently-modified). The pulled state is written with
`~/`-relative path normalisation. Source layout:
`~/Library/Application Support/RepoPrompt/Workspaces/<id>/workspace.json`.

## Hosting providers

`abcd site setup` routes a rendered site through a provider seam,
`internal/adapter/hosting`, which is not one of the five capability seams: no
dropped dependency stands behind it, and there is no native default, because a
site is served by somebody. An adapter has two halves. The repository half is
data — the deploy step, the secret names that step reads, and the host
configuration file it deploys from — rendered into files the verb writes. The
host half connects with the person's credential and creates the host, routes a
domain to it and reports the address, after a read that writes nothing.

One provider ships: an assets-only Cloudflare Worker, the host abcd's own site
uses. A second is one implementation of the interface and one entry in the
site package's provider list; the verb does not change. The credential is
resolved by name through the credential store (`internal/core/credential`),
the one reader; the connected adapter holds it, and it is scrubbed from every
host message before one can reach an error. The provider's `Verify`, the same
account read the host stage begins with, is the verification call the
credential walkthrough makes before it stores a token.

**Every external credential goes through one store** (adr-2609221017021499):
configuration names a credential, and its value lives in the home the person
chose once at `abcd ahoy credential`, the external setup, the abcd-only file or
the platform keychain ([`04-surfaces/01-ahoy.md`](../04-surfaces/01-ahoy.md)).
An adapter that reads a secret any other way is a defect, and a test walks the
production tree for one.

## Lifeboat source readers

Disembark reads a repo's **own settled artefacts** into the lifeboat through a set
of source readers, each implementing the `probe() → extract() → copy()` contract.
These are read adapters over the repo's files — distinct from the capability seams
above, and feeding the native stores rather than any external tool. Interactive
source confirmation (a transparent prompt) fires when assets are found or the docs
structure is ambiguous.

| Source reader | Source / Role | Notes |
|---|---|---|
| spec reader | native spec store (`internal/core/spec`) | Reads the native spec/task tree, newest-first; powers spec-essence |
| transcript reader | native transcript store (`internal/core/history`) | Reads the root-SHA-keyed local corpus; merge by timestamp/content hash when an imported specstory source is also present |
| memory reader | `.abcd/memory/` | Reads the curated memory substrate (repo by default; see [`07-memory.md § 0`](07-memory.md#0-memory-scopes-and-routing)). **Read-only on any vendor harvest source** — see invariant below |
| reviews reader | `.abcd/work/reviews/<YYYY-MM-DD>-<scope>/*.md` (charter grammar) + spec-tied reviews | Reads oracle/review artefacts written by the `oracle` seam's capture side; powers review-collator |
| `claude_md` reader | `AGENTS.md` and `CLAUDE.md` | Snapshot: either file grounds the invariants section; its `git log -p` history is a design target, not read yet |
| `adr` reader | ADR location varies per project | Probes common paths: `docs/development/decisions/adrs/`, `docs/adr/`, `docs/architecture/decisions/`, `adrs/`. Newest-first; respects `Superseded-By`. Configurable via `.abcd/config.json` → `adr.path` if non-standard |
| `git_log` reader | `git log` | Powers spec-window indexing for chat-distiller |
| `assets` reader | `docs/**/*.{png,jpg,svg,pdf}`, `Resources/Assets.xcassets/` | Walks; emits `_manifest.json` |
| `user_docs` reader | `docs/{tutorials,guides,reference,explanation}/` | Mirror verbatim |
| `work_dir` reader | `.abcd/work/` and `.abcd/.work.local/` | Working notes, drafts, status trackers; the `.abcd/work/issues/` ledger. Disembark reads the curated `.abcd/development/` outputs, not the working tiers directly |

**Vendor-memory read-only invariant.** The memory reader treats any vendor memory
directory it harvests (under Claude Code: `~/.claude/projects/<encoded-cwd>/memory/`)
as a **read-only source**. abcd reads it, distils it, and writes curated pages to
its own `.abcd/memory/` — it **never writes to, edits, or deletes anything under
`~/.claude/`**. That directory is the harness's domain and the user's. abcd keeps
`~/.claude/` minimal: only the abcd plugin install lives there; all abcd-authored
material routes to the scope-appropriate `.abcd/` (per
[`03-configuration.md` § The two `.abcd/` scopes](03-configuration.md#the-two-abcd-scopes)).
The vendor-format knowledge — following the `MEMORY.md` index to the per-fact files
and reading their frontmatter — exists purely to *read* structure, not to
round-trip it.
