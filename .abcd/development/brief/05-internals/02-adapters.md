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
consumers are untouched. Of the five, only `scanner` is a directory under `internal/adapter/` in the tree; `oracle`, `history`, `spec` and `run` are design targets, each introduced by the first intent that consumes it, and today their native paths live in `internal/core` (`oracle`, `history`, `spec`, `implement`). `internal/adapter/` also holds adapters that are not capability seams: `gitleaks`, `hosting` and `openaiapi`. [`internal/README.md`](../../../../internal/README.md) § Planned seams is the gated list. See
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
host reading such a receipt knows the step already ran. DR5 is checked before a
verb writes anything (`APIConfig.Admitted`), and again by the dispatch. An
ingest handed a payload the host produced while its agent is routed to a
provider refuses it, since the receipt would name work the provider never did;
`--route <agent>=host-decides` keeps one run on the harness. The four disembark
agents read the packed lifeboat and no verb builds a request carrying it, so none
is sent to a provider. No test reaches a real provider: the client and every
dispatching verb are exercised end to end against a fake on the loopback
address, and the key is built at run time.

### RepoPrompt oracle adapter — `dev-sync reviews` harvesting

RepoPrompt is one opt-in `oracle` (mcp) adapter. When it is wired, `dev-sync
reviews` harvests **ad-hoc oracle reviews not tied to a spec** from two
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
