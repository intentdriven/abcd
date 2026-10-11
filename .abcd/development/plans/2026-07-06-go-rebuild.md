# Plan: Rebuild abcd in Go — a host-agnostic configuration layer for development

> The record information architecture and every locked decision below are
> ratified as ADRs in [`../decisions/`](../decisions) (ADR-21…ADR-30). This plan
> sequences the build; it does not re-argue the decisions — follow the ADR links
> for the *why* and the alternatives rejected.

## Context and thesis

The prior abcd is a large Python codebase wrapped in a Claude Code plugin
(markdown `commands/`, `agents/`, `skills/`, `hooks/`). It delivers a lifeboat
round-trip (`disembark`/`embark`), an intent→brief→review methodology, install
(`ahoy`), and public `launch`. But a large fraction of that Python exists **only
to survive external tools**: the overlay, dispatcher, session-mirror, and
dep-watcher subsystems exist *solely* to keep abcd's modifications alive across a
bundled planning tool's re-vendoring.

**Drop the external-tool hard dependencies and that entire subsystem evaporates.**
The real rewrite is far smaller than the Python line count suggests. This plan
rebuilds abcd from scratch in **Go**, depending on **no external tools**
(transcript capture, an independent reviewer, a context-assembly tool, an
autonomous loop, a planning pipeline), so abcd becomes a portable, open-source
**configuration layer for development** — useful as a Claude Code plugin, inside
the companion harness's ecosystem, and later in any MCP host.

This is a from-scratch rebuild guided by the existing brief/intents/ADRs (the
design record *is* the spec), not a line-by-line port. It is the disembark/embark
philosophy applied to abcd itself: keep the hard-won wisdom, drop the cruft.

## Locked decisions

Each row is ratified in an ADR; see [`../decisions/`](../decisions).

| # | Decision | ADR |
|---|---|---|
| Rebuild | Rebuild abcd as a single Go binary; retire the Python plugin machinery. | [ADR-21](../decisions/adrs/0021-rebuild-in-go.md) |
| Adapters | Every bundled dependency becomes a **pluggable adapter over a native default** — "basic built-in, plug in for power". | [ADR-22](../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md) |
| Core shape | A **transport-agnostic Go core**: capabilities are functions taking a structured request, returning a structured result — no stdout/prompt/transport coupling. Thin front doors (CLI, markdown plugin, MCP) call the same core. | [ADR-23](../decisions/adrs/0023-transport-agnostic-core.md) |
| the companion harness | **Peer via conventions + MCP.** No Go dependency either direction; integrate through auto-load conventions and (later) an MCP server. | [ADR-24](../decisions/adrs/0024-companion-harness-peer-via-conventions.md) |
| LLM path | **Host-delegated by default.** The core does deterministic work (parsing, gates, file ops) and hands prompts back to the host's subagent dispatch. Native/CLI/API/MCP oracles are opt-in adapters. | [ADR-25](../decisions/adrs/0025-host-delegated-llm-default.md) |
| Spec/task | **Native minimal MVP**, with the companion harness `ccpm` as the primary deeper backend (convention-level, no binary dep). The prior bundled planning pipeline is **dropped** from the default plan. | [ADR-26](../decisions/adrs/0026-native-spec-layer-ccpm-backend.md) |
| Autonomous run | **A pluggable seam, not a native port** of the prior autonomous loop: Claude Workflows on Claude Code, the companion harness's agent loop on the companion harness, a thin native Go loop as the headless fallback. | [ADR-27](../decisions/adrs/0027-autonomous-run-pluggable-seam.md) |
| Repo topology | **Single repo, curated release** (no dev→public mirror). `.abcd/**` stays in-tree but is excluded from the release artifact by packaging; the repo *is* the marketplace. | [ADR-28](../decisions/adrs/0028-single-repo-curated-release.md) |
| Transcripts | **Native local store by default** (redacted, gitignored, private); a private companion/remote is optional for a shared corpus; hosted transcript cloud is an optional convenience, never the MVP default. | [ADR-29](../decisions/adrs/0029-native-transcript-corpus.md) |
| Record IA | **Flat artefact-type folders** under `.abcd/development/`; Diátaxis for user docs; generated CLI reference. | [ADR-30](../decisions/adrs/0030-record-information-architecture.md) |

**Delivery order:** ship an **MVP**, then extend via **the companion harness**, then **Claude
Code**. The first milestone is **install + launch** — abcd ships *itself* as a Go
Claude Code plugin.

## Core architecture — transport-agnostic core

Every capability is a function in `internal/core` taking a structured request and
returning a structured result. No stdout, prompt text, or transport coupling
lives in the core. Three thin front doors call the same core:

```
                 ┌── surface/cli   (Cobra)      → default, reliable, no daemon
internal/core ───┼── plugin surface (markdown)  → auto-loaded by Claude Code and the companion harness; shells to `abcd <verb> --json`
                 └── surface/mcp   (MCP server)  → added anytime; exposes core verbs as mcp:abcd:* tools
```

This single constraint makes "add MCP later" trivial and answers the
MCP-reliability worry: **the CLI is the default; MCP never becomes the only
path.** See [ADR-23](../decisions/adrs/0023-transport-agnostic-core.md).

### Go module layout

Module path matches the repo origin (`github.com/<owner>/abcd-cli`), so it is
stable and never forces an import-path rename.

```
cmd/abcd/main.go              # CLI entry (Cobra — matches the companion harness's stack)
internal/core/                # transport-agnostic engine; one package per capability
  ahoy/  launch/  capture/  memory/  intent/  brief/  review/
  spec/  run/  lifeboat/  history/
internal/adapter/             # pluggable seams (interface + native impl + external plug-in)
  oracle/     # HostDelegated (default) | ClaudeCLI | AnthropicAPI | MCPOracle
  history/    # native store (default) | hosted transcript adapter
  spec/       # native store (default) | the companion harness ccpm (primary deeper backend)
  run/        # native thin loop (fallback) | ClaudeWorkflows (CC) | the companion harness's loop
  scanner/    # native secret/PII (default) | gitleaks | trufflehog
internal/surface/cli/         # Cobra tree → core
internal/surface/mcp/         # MCP server → core (added once a surface is worth exposing)
internal/config/  internal/registry/   # declarative adapter registry
# Plugin surface (shipped in the CC plugin, auto-loaded by the companion harness too):
commands/abcd/*.md            # each shells to `abcd <verb> --json`
agents/*.md                   # markdown host-delegated reviewers
skills/**  +  the companion harness's skills-directory mirror   # the companion harness does NOT read .claude/skills
hooks/                        # prompt-router + guard hooks (Go binaries or thin shims)
.claude-plugin/{plugin.json,marketplace.json}
```

Every adapter is a seam with a **native default** and an **optional external
plug-in** ([ADR-22](../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)).

## Working-tree layout — record / work / local

All development material lives under `.abcd/`; `docs/` is user-facing only. Three
tiers on two axes (durability × sharing), aligned to abcd's git *and* publish
boundaries. See [ADR-30](../decisions/adrs/0030-record-information-architecture.md).

| Tier | Path | Committed? | Ships in launch? | Contents |
|---|---|---|---|---|
| 1 · Durable record | `.abcd/development/` | yes | **no** (`.abcd/**` excluded) | brief, roadmap/{intents,phases}, decisions/adrs, plans/ (dated), research/, principles |
| 2 · Shared working | `.abcd/work/` | yes | no | `CONTEXT.md` (team orientation — active workstreams, current phase, live constraints); `DECISIONS.md` (append-only decision log) |
| 3 · Local ephemeral | `.abcd/.work.local/` | **no** (gitignored) | no | `NEXT.md` (per-worktree handover), `scratch/`, `logs/`, runtime |
| — · User docs | `docs/` | yes | **yes** | Diátaxis user-facing docs only |

- **One rule, two boundaries:** "dev material under `.abcd/`, user docs under
  `docs/`" satisfies both what-is-committed and what-`launch`-ships. `.abcd/**` is
  excluded from the public payload by default-deny; only `docs/` reaches users.
- **Three doc roles, kept distinct:** `brief` = durable "what this *is*" (low
  churn); `CONTEXT.md` = shared "what to know *now*" (medium churn); `NEXT.md` =
  one engineer/agent's session handover (local, high churn).
- **DECISIONS scaling path:** start with a single append-only
  `.abcd/work/DECISIONS.md`; graduate to one-file-per-decision
  `.abcd/work/decisions/<date>--<slug>.md` when size or parallel-agent merge
  contention bites. Name by date-prefix + slug, not sequential `DNNNN` (sequential
  IDs race across branches).

### Record information architecture (ADR-30)

Flat artefact-type folders — durable-vs-working is already carried by the
`development/` vs `work/` vs `.work.local/` tiering, so a second such axis inside
`development/` would double-classify:

```
.abcd/development/
  README.md      # the map — what lives where
  brief/         # the living canvas: what this IS + glossary + invariants
  intents/       # the WHY — press-release intents, dir-as-status
  principles/    # distilled cross-cutting design principles
  decisions/     # ADRs — MADR, NNNN-title.md, one canonical home
  roadmap/       # sequencing — phases/ + rfcs/
  plans/         # dated design/impl plans (YYYY-MM-DD-*)
  research/      # investigations — notes/ (dated) + spikes/
```

**Numbering split:** ADRs use sequential `NNNN` (a stable cross-reference handle);
plans, research notes, and the Tier-2 `DECISIONS.md` log use a date-prefix
(chronological artefacts read newest-first). Issues are **not** a record folder —
design-significant ones graduate into `intents/` or `principles/`; trivia lives in
local `NEXT.md`.

**User-facing docs — Diátaxis, generated CLI reference:**

```
docs/
  README.md   tutorials/   how-to/   explanation/
  reference/          # hand-written reference (config, schemas)
    cli/              # GENERATED from Cobra (doc.GenMarkdownTree) — never hand-edit; CI freshness-checked
  assets/
CONTRIBUTING.md → repo root (out of docs/)
```

The SOTA basis for this IA is captured in
[`../research/notes/2026-07-06-docs-and-record-ia-sota.md`](../research/notes/2026-07-06-docs-and-record-ia-sota.md).

## Repo topology & the `launch` reframe

See [ADR-28](../decisions/adrs/0028-single-repo-curated-release.md).

- **Single repo.** `.abcd/**` stays in-tree (transparent) but is **excluded from
  the release artifact** the way `.npmignore`/`files:`/goreleaser work — packaging,
  not a second repo. The repo *is* the marketplace.
- **`launch` = cut a curated release, not mirror a repo.** On a `v*` tag, build
  the plugin payload (markdown surface + compiled `abcd` binary +
  `plugin.json`/`marketplace.json`), run the secret/PII gates on the *bundle*,
  publish a GitHub Release with newest-per-line retention, attest provenance
  (SLSA). The former cross-repo mirror/manifest/overlay machinery is **deleted**.
- **Visibility:** develop privately during MVP incubation, then **flip the same
  repo to public** at maturity — single repo throughout, no mirror. The
  public-flip checklist: (1) full-history secret scan (gitleaks in CI); (2) apply
  the default-branch protection ruleset — deferred because a free-tier private repo
  cannot carry a ruleset, with local pre-push and git-guardrails hooks as interim
  protection.
- **Private companion repo = optional, deferred.** Warranted only for genuinely
  private material; the identified trigger is a **shared transcript corpus**. Never
  a dev↔public mirror.

## Transcripts & history

See [ADR-29](../decisions/adrs/0029-native-transcript-corpus.md). Transcripts are
a **research corpus** — for benchmarking, tracing when a flaw was introduced,
cross-model comparison, and hardening the harness via reflection — not archival
junk.

- **MVP default — native local store.** Write transcripts to a machine-local store
  (`~/.abcd/history/<root-sha>/`, keyed on the root-commit SHA), gitignored,
  private by construction, **redacted on capture** (secrets/PII/absolute paths
  scrubbed). Never enters the repo or the release artifact. This is the native
  transcript-capture replacement.
- **Optional — private companion / remote.** Sync the local store to a private git
  repo or object store when the team needs a *shared* corpus. This is the concrete
  justification for a private companion repo, added only when the need is real.
- **Optional — hosted transcript cloud.** A convenience plug-in — never the MVP
  default.
- **Product tie-in:** this corpus feeds later `reflect`, benchmark-driven config
  optimisation, and the self-dogfooded SOTA audit — all unlocked without a cloud
  dependency.

## Parallel multi-agent development (git worktrees)

The build workflow supports multiple agents on multiple branches simultaneously,
isolated by **git worktrees** — each agent gets its own working directory over one
shared `.git`, so concurrent edits never collide.

- **One agent → one branch → one worktree → one PR.** Spawn with
  `git worktree add ../abcd-cli-<feat> -b feat/<x>` (or the harness's worktree
  isolation).
- **CI gates each PR independently.** Required checks + strict up-to-date means a
  parallel branch must rebase onto the latest default before merge — simultaneous
  PRs serialise safely at the merge point, never in the working trees.
  `delete_branch_on_merge` + `fetch.prune` keep the branch list clean.
- **Decompose to non-overlapping file scopes.** Assign each agent a disjoint area
  (e.g. `internal/core/launch/` vs `internal/adapter/scanner/`). Per-worktree
  `.work.local/NEXT.md` tracks each agent's own state; committed
  `.abcd/work/CONTEXT.md` + `DECISIONS.md` are the shared coordination surface.
- For now this is a **development practice, not a shipped feature** — though it
  seeds a future multi-agent-coordination intent built on the run seam.

## External-tool basic-build difficulty ranking

For each bundled dependency: how hard to build a **basic native** version into
abcd, and what **plugging the external tool** buys on top. Difficulty runs
easiest → hardest.

| Rank | Tool role | Native "basic" build | Difficulty | Plugging external adds |
|---|---|---|---|---|
| 1 | **Transcript capture** | Structured file writes to a native, redacted, machine-local history store keyed on root-commit SHA (gitignored). Optional sync to a private companion for a shared corpus. | **Trivial** | Hosted transcript storage/sharing as an optional convenience — never the MVP default. |
| 2 | **Independent reviewer** | The oracle basic already exists: host-delegated review (default). An independent local reviewer is a subprocess LLM call behind the `oracle` interface. | **Easy** | A genuinely independent second-model adversarial review (breaks host echo-chamber). Plug via a subprocess adapter. |
| 3 | **Context-assembly tool** | Host-delegated review + simple deterministic context assembly (glob/rank files). Review-tab lifecycle becomes plain records in abcd's store. | **Moderate** | Strong context selection + an independent MCP oracle. Plug via an MCP-client adapter — the case that justifies the MCP front door. |
| 4 | **Autonomous loop** | A thin Go loop that spawns host workers per ready task with receipt gating + a safety guard hook — a *fallback only*. Real orchestration is delegated to the host. | **Low** (fallback only) | Battle-tested unattended resilience — but the host now provides it. Claude Workflows on Claude Code; the companion harness's agent loop on the companion harness. |
| 5 | **Planning pipeline** | A native minimal spec/task engine: specs, tasks, dependency graph, status-by-directory. The data model is tractable; the full plan/work/interview pipeline is large. | **Moderate** (minimal only) | A mature planning pipeline. Plug the **primary** deeper backend = the companion harness `ccpm` at the markdown-convention level (epics/PRDs, no binary dep). The prior bundled pipeline is **not** in the default plan. |

**Design consequence:** the two former "hard" cases collapse. The autonomous loop
is **not ported** (host orchestrators do the work) and the planning pipeline is
**dropped** (native-minimal store + the companion harness `ccpm`). This is what lets abcd ship an
MVP and *extend via the host* rather than reinvent orchestration and planning.

## Reuse — port contracts, not code

The Python already has clean seams; port the **contracts** to Go interfaces rather
than reinventing them. References below are to the frozen Python reference
implementation (repo-relative paths within that codebase):

- `scripts/abcd/harness.py` (historical) — `AgentDispatch` / `MCPBridge` protocols →
  `internal/adapter/oracle` Go interfaces (the injection seam already proves the
  oracle is replaceable).
- `scripts/abcd/tools/<tool>/config.json` (historical) + `tools/registry.py` + `tools/probes/`
  — the declarative tool registry → `internal/registry` + `internal/adapter/*`.
- `scripts/abcd/oracle.py` (historical) cascade → an `oracle` chain with HostDelegated as the
  default first leg.
- Launch reference logic: `launch_ship.py`, `launch_preflight.py`,
  `public_manifest.py`, `plugin_payload.py`, `manifest_lockstep.py`,
  `launch_gate_*.py`, `scan.py`, `src/pii.py`.
- Install reference logic: `scripts/abcd/ahoy/` (historical; `_detect.py` folder-kind,
  gitignore, marker blocks), `hooks/prompt_router_hook.py`.
- History reference: `history_store.py`, the transcript-capture shim.
- **Behaviour spec** = the brief's per-command contracts under
  `brief/04-surfaces/` (one per verb), the mental model in
  `brief/01-product/03-mental-model.md`, and the ADRs in `decisions/adrs/`.
- **The companion harness's auto-load targets** (so the plugin surface lands correctly): the
  companion harness's command discovery, subagent definition, memory loader, and
  skill discovery surfaces (skills must be mirrored to the companion harness's skills directory — the companion harness
  does not read `.claude/skills`), plus its MCP surface for the MCP front door.

## Implementation phasing

Gated phases; each ends with something demonstrably wired and runnable. **STOP and
report if a phase's exit criteria are not met — never push through.**

### Phase 0 — Foundations (scaffold from two sources)

`abcd-cli` inherits its **design record** from the frozen Python reference
implementation and its **git/CI/dev-workflow** from a Go CLI project template (git
hooks, CI, release workflow — itself a near-exact Go CLI template).

**Carry from the Go CLI template verbatim (language-agnostic):**
`.git/info/exclude` (runtime-state ignores), `.github/dependabot.yml` (gomod +
github-actions, supply-chain cooldown), the `CHANGELOG.md` format (Keep a
Changelog + SemVer `v`), and the `CLAUDE.md`→`AGENTS.md`←`GEMINI.md` symlink
pattern. The `ci.yml` `gitleaks` + `zizmor` jobs unchanged, keeping SHA-pinned
actions, `persist-credentials: false`, per-job least-privilege `permissions`, and
concurrency-cancel except on the default branch.

**Carry from the template then adapt:** `.gitignore` (keep the Go block; drop
template-only lines); `Makefile` (`build`/`test`/`vet`/`preflight`/`clean`;
adjust binary name and `-ldflags -X …version`); `.githooks/pre-push` →
`exec make preflight`; the `ci.yml` `check` job (gofmt gate + build/vet/test +
`-race ./internal/...`), replacing the template's eval job with abcd-cli's Phase-1
install/launch integration tests; the `AGENTS.md` skeleton and `README.md` layout,
rewritten for abcd-cli. The working-tree layout is **not** the template's layout
verbatim — use abcd-cli's reformed three-tier `.abcd/` scheme.

**Carry from the reference implementation:** copy `.abcd/development/` (the design
record) — the spec.

**Generate via skills:** the `.claude/` layer via a repo-scaffold skill; git-safety
hooks via a git-guardrails skill.

**Reuse win:** the template's release workflow (cross-compile, `checksums.txt`,
SLSA `attest-build-provenance`, newest-per-line retention, no-branch-push
tripwire) is a strong template for abcd's own `launch`, since the public payload
ships a compiled binary and per-line retention is already wanted.

**Then build the abcd core skeleton:** the Go module (Cobra); `internal/core`
skeleton + `internal/registry` + `internal/adapter/*` interfaces (oracle, history,
spec, scanner) with **native default** stubs; plugin-surface scaffolding
(`commands/abcd/*.md` shelling to `abcd <verb> --json`;
`.claude-plugin/{plugin.json,marketplace.json}`; the companion harness's skills-directory mirror).

**Exit:** `make preflight` green; `abcd --help` runs; a trivial verb round-trips
CLI → core → JSON, invoked from a markdown command in a real host; CI green on a
PR.

### Phase 0.5 — Reconcile the design record (before any feature code)

A full pass over the copied `.abcd/development/` so it matches these decisions —
the record never contradicts the plan. Mark superseded ADRs and rewrite affected
intents where the old architecture is dropped (the abstraction-boundary/overlay,
the two-repo dev→public launch, the bundled planning tool as required, the
autonomous loop, cascade-based oracles). Add the new ADRs
([ADR-21…ADR-30](../decisions)). Reconcile the brief's surfaces/internals and the
roadmap to the new delivery order.

Executed in **two steps**: **PR A** = pure structure move (`git mv`/rename,
behaviour-preserving); **PR B** = content reconciliation (supersede/add ADRs,
realign intents/brief). The SOTA findings behind the record IA are captured as
[`../research/notes/2026-07-06-docs-and-record-ia-sota.md`](../research/notes/2026-07-06-docs-and-record-ia-sota.md);
the record-IA choice is [ADR-30](../decisions/adrs/0030-record-information-architecture.md).

**Exit:** the record is internally consistent and contradiction-free against the
plan; cross-links resolve (deterministic doc-lint + a fidelity read).

### Phase 1 — Install + Launch (first milestone: abcd ships itself in Go)

- `abcd ahoy` — folder-kind detection, visibility-driven gitignore, marker blocks
  in CLAUDE.md/AGENTS.md, prompt-router hook install; idempotent.
- `abcd launch` (`ship`/`dry-run`) — cut a curated **release artifact from this one
  repo** (no cross-repo mirror): default-deny bundle manifest excluding `.abcd/**`;
  native Go secret + PII scanners (pluggable gitleaks/trufflehog); strict SemVer;
  `marketplace.json` lockstep anti-drift; newest-per-line retention; SLSA
  provenance. On a `v*` tag → publish a GitHub Release.
- The abstraction-boundary/overlay/dispatcher machinery is **deleted, not ported**
  — with no bundled deps to hide, it largely ceases to exist.

**Exit:** `abcd launch dry-run` proves excludes never leak; abcd installs itself
into a scratch repo and loads as a plugin in **both** Claude Code and the companion harness.

### Phase 2 — History + capture + memory (native)

Native history store (SHA-keyed); `abcd capture` issue ledger; `abcd memory`
ingest/ask/lint. Retires the external transcript-capture dependency.

### Phase 3 — Intent + brief + review methodology

`abcd intent` (capture/refine/grill/shape/link), the brief model, and the
intent-fidelity reviewer via the **host-delegated oracle** (independent reviewer /
context-assembly tool pluggable); the doc-fidelity gate. **Enable the MCP front
door here** — the first surface genuinely worth exposing as `mcp:abcd:*` to
the companion harness/other hosts; validates the transport-agnostic-core decision end-to-end.

### Phase 4 — Native minimal spec/task engine + intent→plan→ship

Dir-as-status spec/task store + dependency graph. The `spec` adapter seam with
**the companion harness `ccpm` as the primary deeper backend** (read/write its epics/PRDs
markdown conventions — no binary dep). The prior bundled planning pipeline is
**not** built. Wire `intent plan` / `intent ship` onto the native store via the
host.

### Phase 5 — Autonomous run seam (Workflows / the companion harness / native fallback)

Define the `run` adapter interface; **do not port the prior autonomous loop.**
Backends: **Claude Workflows** under Claude Code (the abcd command emits/hands off
a workflow script for the host to run — deterministic, budget-aware); the
**the companion harness's agent loop / ccpm** under the companion harness; and a **thin native Go loop** (receipt
gating + a safety guard hook: block `git push`, protected-path writes) as the
headless fallback.

### Phase 6 — Lifeboat round-trip (crown jewel)

`abcd disembark` / `abcd embark` — packs decisions/principles/pitfalls/spine from
the now-native substrates (specs, ADRs, history, reviews, memory) with audit
gates; unpacks into a clean repo. Lands last because it *depends on* every prior
substrate being native.

## Verification (end-to-end, per phase)

- **Unit:** `go test ./...` on every touched package; each new behaviour gets a
  test watched fail → pass. Every gate/scanner is table-driven.
- **Phase 1 (milestone) end-to-end:**
  1. `go build ./cmd/abcd` → single binary.
  2. In a scratch repo: `abcd ahoy` → assert marker blocks in CLAUDE.md/AGENTS.md,
     gitignore entries, prompt-router hook registered; re-run → idempotent no-op.
  3. `abcd launch dry-run` → assert the rendered release bundle contains zero
     `.abcd/**` paths and the secret/PII gates run for real.
  4. Load the plugin in **Claude Code** (`/abcd:ahoy`) and in **the companion harness**
     (auto-load of commands + agents; skills via the companion harness's skills directory) → both invoke
     the same `abcd` binary and produce identical results.
- **MCP front door (Phase 3):** start `abcd mcp`, register the abcd MCP server in
  the host, confirm the host calls `mcp:abcd:*` and gets the same core results as
  the CLI — proving one core, three front doors.
- **Wiring check:** every verb must be reachable from the plugin markdown surface
  AND the CLI, and demonstrably execute there — not just compile and pass unit
  tests.

## Repo & module (resolved)

- **Repo:** the single `abcd-cli` repo; build here. It flips private→public at
  maturity and *is* the marketplace.
- **Go module path:** matches the clone origin (`github.com/<owner>/abcd-cli`) —
  stable, never forcing an import-path rename since no separate public repo exists.
- **Record migration (Phase 0, one-time, manual):** copy `.abcd/development/` from
  the frozen reference implementation into `abcd-cli`; thereafter `abcd-cli`'s
  record is canonical and the reference implementation is reference-only. Once
  Phase 6 lands, re-deriving `abcd-cli` from the frozen reference via
  `disembark`→`embark` becomes the first real end-to-end test of the lifeboat —
  validating abcd's core premise on abcd itself.
- **Repo settings (apply via `gh` once CI jobs exist):** `git config fetch.prune
  true`; the default-branch ruleset (require PR; required checks =
  `check`/`gitleaks`/`zizmor` + integration; strict up-to-date; block
  force-push/deletion; no bypass); enable `delete_branch_on_merge` and
  allow-auto-merge (armed only on green `docs:`/`chore:` PRs).

## New-dependency gate

Adding **Cobra** (CLI) up front, and later an **MCP Go SDK** and a
**secret-scanner** library, each needs explicit sign-off before `go get` — name
the dependency and its no-dependency alternative. Go itself is the sanctioned
premise of this work.
