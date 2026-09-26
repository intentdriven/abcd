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
capability contract. Adding a backend = implement the interface and register it in
`internal/registry`; consumers are untouched. See
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
The invariant it serves is adr-2609221009491186's: **a provider adapter serves
only the models it lists, under a vendor denylist no listing overrides, and
everything else runs on the host.** `internal/core/oracle` enforces it before a
client is ever built: a route to an unlisted model is refused naming the list, a
listed model the denylist matches is refused whatever the list says, and a
reported model the denylist matches discards the answer. The configuration is in
[`03-configuration.md`](03-configuration.md#the-provider-adapters-keys).

The client's own guarantees are the network path's. The base URL is pinned per
provider block, plain HTTP is admitted only to this machine, and a redirect is
never followed, so a provider cannot move the key or the brief elsewhere. Every
response is bounded in size and every call in time. The key travels only as the
bearer header of a request to the pinned address. A provider's own text, its
error and the model it reports, is decoded (JSON escapes undone, HTML character
references resolved), bounded, sanitised and scrubbed of the key in every form an
encoder gives it (literal, JSON-, HTML- and URL-escaped, quoted) before it
reaches an error or a record, because a provider may echo what it was sent. A
setting the protocol does not take is refused before the call, and the answer is
judged by the caller's output contract, the one the host sub-agent's payload is
judged by. The request is the host's brief in the protocol's two roles: the
agent's prompt as the system message, the verb's request as the user message.

The key is resolved by name through `internal/core/credential`, the one reader;
the adapter reads no file, no environment and no store of its own. Its
connection (`oracle.Connections`) carries the provider's allowlist and the
settings the adapter accepts, which is what the model tier's allowlist check and
its accepted-settings refusal read (spc-2609251028149555). A provider claims no
tier: it is reached by a role or a judgement type pointed at it, never by a tier
alone. No delegating verb dispatches a step through it yet, and no test reaches a
real provider: the client is exercised end to end against a fake on the loopback
address that fails in every way a provider can.

### RepoPrompt oracle adapter — `dev-sync reviews` harvesting

RepoPrompt is one opt-in `oracle` (mcp) adapter. When it is wired, `dev-sync
reviews` harvests **ad-hoc oracle reviews not tied to a spec** from two
RepoPrompt-local sources (spec-tied reviews go straight to the native spec
review store and are never swept here). Vendor filesystem layout lives here with
the adapter, not in the core config brief:

- **Chat store** — `~/Library/Application Support/RepoPrompt/Workspaces/Workspace-<project>-<UUID>/Chats/ChatSession-*.json` (plain JSON, well-structured).
- **Prompt exports** — RepoPrompt's `export_response: true` writes ad-hoc reviews to `<cwd>/prompt-exports/` with a **hardcoded, non-configurable path**, which ahoy redirects into `~/.abcd/history/<root-sha>/prompt-exports/` via the `<repo>/prompt-exports` symlink.

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
resolved by name through `internal/core/credential`, whose interim source is
`~/.abcd/credentials.json` until the credential store (itd-2609221017023290)
replaces it; the connected adapter holds it, and it is scrubbed from every host
message before one can reach an error.

## Lifeboat source readers

Disembark reads a repo's **own settled artefacts** into the lifeboat through a set
of source readers, each implementing the `probe() → extract() → copy()` contract.
These are read adapters over the repo's files — distinct from the capability seams
above, and feeding the native stores rather than any external tool. Interactive
source confirmation (a transparent prompt) fires when assets are found or the docs
structure is ambiguous.

| Source reader | Source / Role | Notes |
|---|---|---|
| spec reader | native spec store (`internal/adapter/spec`) | Reads the native spec/task tree, newest-first; powers spec-essence |
| transcript reader | native transcript store (`internal/adapter/history`) | Reads the root-SHA-keyed local corpus; merge by timestamp/content hash when an imported specstory source is also present |
| memory reader | `.abcd/memory/` | Reads the curated memory substrate (repo by default; see [`07-memory.md § 0`](07-memory.md#0-memory-scopes-and-routing)). **Read-only on any vendor harvest source** — see invariant below |
| reviews reader | `.abcd/work/reviews/<YYYY-MM-DD>-<scope>/*.md` (charter grammar) + spec-tied reviews | Reads oracle/review artefacts written by the `oracle` seam's capture side; powers review-collator |
| `claude_md` reader | `CLAUDE.md` + `git log -p CLAUDE.md` | Snapshot + history |
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
