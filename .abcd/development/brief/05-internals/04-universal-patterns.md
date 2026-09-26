# Universal Patterns

All abcd commands share these patterns. Implement once in shared helpers, not per-command.

## 1. Transparent prompts

Every `AskUserQuestion` (or equivalent harness call) shows:

1. **Current state** ("Current: private")
2. **Consequence of each option** ("Switching to public removes the `.abcd/` namespace (design record, native spec store, memory) from tracking")
3. **The question + how to change later** ("Keep private? — change later by re-running `abcd ahoy install --visibility public`")

No silent defaults. No surprises.

## 2. Host-delegated by default, oracle adapters opt-in

Every step that needs a model follows this pattern:

```
host-delegated (abcd hands a prompt to the host's subagent dispatch)
    └─ or, when an operator wires one → an oracle adapter (native | CLI | API | MCP)
```

abcd's core does the deterministic work and hands a **prompt** to the host's subagent dispatch; the host owns model choice, credentials, and execution, and abcd consumes the structured result. No model plumbing is required by default (adr-25). Concrete oracle backends — native, CLI, API, MCP — are opt-in adapters behind the same seam, selected when an operator wants abcd to reach a model directly.

Examples:
- **Oracle seam** (lifeboat-reviewer, press-release-composer, intent-auditor): host-delegated by default; an operator may wire a native/CLI/API/MCP oracle adapter.
- **Code-rescuer**: spec-driven git-window file selection natively; a codemap adapter refines it when one is wired.
- **Scoped + broad reviewers** (adr-25 adapter guidance): when an operator wires two oracle adapters for a high-stakes review, a **scoped** reviewer (seeing only a selection) and a **broad** reviewer (reasoning over the whole repo) have complementary blind spots and are trusted **asymmetrically** — the scoped verdict gates, the broad reviewer is mined for findings, and the review-fix loop declares its stopping rule up front. Guidance the adapter layer offers, never a cascade the core imposes.
- Future agents follow the same pattern by default.

## 3. Native by default, peer interop over conventions and MCP

abcd runs fully on its own native code with no peer installed. When a peer tool is present — the companion harness above all — abcd interoperates with it as a **peer over shared conventions and MCP**, never a code dependency in either direction (adr-24). Interoperation is a capability, never a prerequisite.

```
native default (abcd's own core)
    └─ when a peer is present → interoperate over conventions (shared on-disk shapes) + MCP
```

- **Conventions** are the ground-level contract: shared on-disk markdown shapes — the `ccpm` spec/task layout (adr-26) — that either tool reads and writes without linking to the other.
- **MCP** is the runtime contract: when both are present, abcd and the companion harness interoperate as two independent MCP servers, each a front door over its own core (adr-23).
- **Spec/task creation**: `/abcd:intent plan <itd-N>` scaffolds the bidirectionally-linked spec in the native minimal spec store (adr-26); the canonical create `/abcd:intent "<text>"` only writes a draft. When the companion harness's `ccpm` is wired as the deeper backend, the same on-disk shape is read and written at the convention level.
- **Future**: any abcd capability with a strong peer counterpart interoperates at the convention/MCP boundary, never by importing or vendoring it.

## 4. JSON internal, MD render

All inter-agent data is JSON; markdown is a render step at the end of each pass.

- Each JSON artefact has a JSON Schema owned by the core (`internal/core/schema`)
- The render capability (`internal/core/render`) provides deterministic JSON → MD renderers
- Easier to validate, agents are unit-testable against schemas
- Re-rendering with different templates is cheap

## 5. Reports as JSON + MD pairs

**A design target: no shipped verb writes such a pair yet.** Every command emits `<command>-report.json` (full structured detail) and `<command>-report.md` (human skim summary, rendered from JSON). Both stored in `.abcd/.work.local/logs/<command>/<timestamp>/`.

What ships today is narrower. Most verbs write no report at all: `lint` and its targets, `capture`, `intent`, `guard`, `--version`, `changelog`, `rules`, `spec`, `identity` and `banlist` are read-only renders, and `abcd lint` in particular is documented as performing zero writes. Where run output exists it lands in the local ephemeral tier under `.abcd/.work.local/logs/<area>/<run>/`, which is the shape the pattern generalises.

## 6. `.abcd/.work.local/logs/` layout

**Also a design target, and one already relocated once.** The tree below is what the tier is designed to hold rather than what a checkout holds today. Its root was once `.abcd/logbook/`; a 2026-07-12 adjudication moved run output to the local ephemeral tier and retired that name, which must not be re-minted and is held retired by an armed detector (iss-73, [`02-constraints/04-naming.md`](../02-constraints/04-naming.md)). What the tier actually holds today is narrower and differently shaped: the memory lint's run output at `logs/memory/`, and the scan and audit-history directories the privacy scanner skips over.

```
.abcd/.work.local/logs/
├── ahoy/<timestamp>/
│   ├── ahoy-report.json
│   ├── ahoy-report.md
│   └── prompts.json              # what was asked, what was answered
├── disembark/<timestamp>/
│   ├── disembark-report.{json,md}
│   ├── _state.json               # forensic checkpoint state (no resume; for post-mortem only)
│   ├── progress.log              # streaming progress (tail -f friendly)
│   └── agents/<agent>/<run>.{json,md}
├── embark/<timestamp>/
│   └── embark-report.{json,md}
├── launch/<timestamp>/
│   ├── launch-report.{json,md}
│   └── preflight.{json,md}       # PII/secret scan output
├── intent/<timestamp>/
│   └── intent-report.{json,md}   # one per /abcd:intent invocation
├── capture/<timestamp>/
│   └── capture-report.{json,md}  # one per /abcd:capture invocation (per itd-4)
├── grill/<utc-ts>-<intent-id>/   # one per /abcd:intent grill session (per itd-27)
│   └── grill-report.{json,md}    # glossary terms are written inline to terminology/, not batched here
├── audit/<sub-tier>-<ts>/        # review/audit reports across six sub-tiers land here
│   └── report.{json,md}          # sub-tier ∈ {review, spec-mg, consistency, shape, chain, lifeboat}:
│                                 #   audit/review-<ts>/      (Role 1 itd-1 pass / /abcd:intent audit,        itd-1)
│                                 #   audit/spec-mg-<ts>/     (Role 1 MG004 pass / itd-37 boilerplate receipt, itd-37)
│                                 #   audit/consistency-<ts>/ (Role 2 / /abcd:intent consistency,   itd-48 — superseded itd-31; a later phase, spc-29 predecessor store, not yet a sub-verb)
│                                 #   audit/shape-<ts>/       (Role 3 / /abcd:intent shape,         itd-34, later phase)
│                                 #   audit/chain-<ts>/       (default app of /abcd:audit chain,    itd-16, later phase)
│                                 #   audit/lifeboat-<ts>/    (sibling app of /abcd:audit lifeboat, itd-35, later phase)
│                                 # Directory name (audit/) reflects "this is the on-disk audit trail"
│                                 # regardless of which verb produced it; sub-tier prefix names the verb.
│                                 # `audit` (with its `ingest` child) is the one registered sub-verb
│                                 # of /abcd:intent; `consistency` and `shape` are designed sub-verbs
│                                 # of it, and `chain` and `lifeboat` of the /abcd:audit umbrella,
│                                 # none of the four registered on the shipped surface.
│                                 # Bare /abcd:audit and bare /abcd:intent are status+help only.
├── sota-audits/<date>.{json,md}  # periodic prompt SOTA audit findings (option D)
└── phase/<phase-id>/             # validation cadence outputs per phase (Phase 0 study, Phase 1 acceptance, etc.)
    └── <test-name>.{json,md}
```

**Note: `.abcd/.work.local/logs/` is for reports only.** Coordination state (file locks like `shape.lock`, multi-agent claims per itd-33) lives at `.abcd/coordination/` — its own directory under `.abcd/`, not a subdirectory of the log tier. See `04-surfaces/05-intent.md § 7` for the canonical lock-path contract (`.abcd/coordination/shape.lock`).

**Later-phase additions to `.abcd/.work.local/logs/`** (appear when their parent intent ships):
- `dredge/<timestamp>/` — cross-corpus synthesis output (itd-25)
- `frontier/<timestamp>/` — per-run frontier-mapping events (Frontier Awareness; idea-4)
- `doc-fidelity/<input_fingerprint>/` — the doc-fidelity anti-drift pass planned under **draft intent itd-60** (`intents/planned/itd-60-doc-fidelity-anti-drift.md`; not yet built). **A planned EXCEPTION to the `<command>/<timestamp>/` convention above:** this tier is designed to be **content-addressed**, keyed by the run's `input_fingerprint` (a sha256 over the deterministic trust+reality inputs — receipts, target manifest, bundle manifest, prompts) rather than a timestamp, so an identical re-run reuses the same `report.json` + bound `decision.json` instead of accreting a fresh ts dir. A `decision.json` (approve/defer) binds to a fingerprint dir; `deferred.jsonl` sits directly under `doc-fidelity/` (not inside a fingerprint dir) so an open obligation stays discoverable after the gate clears. The **pre-fingerprint-failures/`<timestamp>/`** sibling is the one ts-keyed slice (a failure that occurs *before* a well-formed fingerprint can be computed — invalid config/manifest, intent-resolution conflict — has no reusable content-addressed report, so its diagnostics are ts-keyed and never reused). This layout contract is the design target for the tier's on-disk shape once itd-60 ships.

**Forward-source provenance marker (planned — the itd-61/spc-75 dedup contract).** Under **draft intents itd-60 and itd-61** (`intents/drafts/`; specs spc-74/spc-75, not yet in the live store), when the doc-fidelity pass drafts a brief delta to repair forward drift, the staged lines will be wrapped in a **paired** HTML-comment stamp so the covered region is unambiguous:

```
<!-- abcd:forward-doc-sync:begin origin=itd-60 spec=spc-N input_fingerprint=<hex> consumed_receipts_sha=<hex> -->
...the drafted delta lines...
<!-- abcd:forward-doc-sync:end -->
```

The pairing is load-bearing: a single self-closing comment cannot delimit a multi-line block, so itd-61/spc-75's derivation dedup needs a **matched** begin/end pair to exclude exactly the freshly-stamped lines and nothing else. `consumed_receipts_sha` is the sha256 over the sorted per-receipt **stable trust-and-reality digests** — the same `{spec_id, parse_error, rollup_agreement, criteria:[{criterion, verdict, detail_key}]}` digest the report's `input_fingerprint` uses (deterministic trust+reality fields only; no LLM-authored `detail` *value*, no timestamp). So the marker is reproducible across reviewer re-runs, does not churn on a forensic-prose rewording, but **does** change when a trust field changes. The grammar pins `origin=itd-60` and requires full lowercase sha256 widths (a short or foreign-origin marker is not valid coverage). **spc-75 will fail closed on any unmatched or legacy single-line marker.** The grammar will be owned by the planned doc-fidelity capability (`internal/core/docfidelity`), first created with the stamping under spc-74.3; the CI gate, the pre-commit advisory wrapper, and the spec-close preflight will all reference it rather than re-implement it.

**Later-phase sibling additions under `.abcd/`** (NOT under the log tier: operational state, not run reports):
- `.abcd/coordination/audit/<YYYY-MM-DD>.jsonl` — multi-agent coordination append-log (itd-33; JSONL, daily UTC rotation, committed). Sibling local-only state (gitignored): `.abcd/coordination/active-work.json` and `.abcd/coordination/*.lock`.

Tracked alongside the rest of `.abcd/` per the visibility rule ([`03-configuration.md § 1`](03-configuration.md#1-visibility-driven-gitignore-policy)) — committed in private repos, gitignored in public. No special exception. Sensitivity is handled at launch time: the launch payload manifest ([`../04-surfaces/04-launch.md § 2`](../04-surfaces/04-launch.md#2-curated-release-artefact-default-deny)) excludes the entire `.abcd/` namespace from what ships publicly.

**The log tier vs `voyage/`:** `.work.local/logs/<command>/<timestamp>/` holds *per-run* command output (reports, prompts asked, forensic checkpoint state — abcd ships no resume sub-verb, so checkpoints are post-mortem only) — ephemeral relative to a single invocation. `~/.abcd/voyage/<source-root-sha>/` (operator level, per adr-35; see [`../04-surfaces/03-embark.md § 7`](../04-surfaces/03-embark.md#7-voyage-layout--embarkdisembark-provenance-and-history)) holds *cross-run* embark/disembark provenance and history that the project carries forward. Both are tracked under the visibility rule; they answer different questions ("what happened in this run?" vs "what is the lifeboat history of this repo?").

## 7. Vendor-agnostic adapters with environment branching

This is the core internals story. **Every capability abcd could take from an external tool is instead a seam over the Go core: an interface, a native default, and an optional external plug-in.** No external tool is a hard dependency (adr-22); abcd runs fully with none installed, and a present tool is an upgrade the seam keeps cheap — never a floor abcd stands on. Each dropped hard dependency maps to exactly one seam:

| Seam (`internal/adapter/<seam>`) | Native default | Optional external plug-in(s) |
|---|---|---|
| **oracle** | host-delegated LLM (adr-25) | native / CLI / API / MCP oracle backends (RepoPrompt, codex, …) |
| **history** | native local redacted transcript store (adr-29) | specstory capture source |
| **spec** | native minimal spec/task store (adr-26) | the companion harness `ccpm` over conventions (adr-24) |
| **run** | thin native Go loop (adr-27) | Claude Workflows, the companion harness's agent loop |
| **scanner** | native secret/PII scan | gitleaks, Presidio, TruffleHog, … |

Each seam is a Go interface in `internal/adapter/<seam>` with a native implementation that ships in the binary; concrete external backends live behind the same interface, selected by config. Consumers in `internal/core` depend on the **interface**, never on a vendor — they consume "an oracle", "a transcript store", "a spec store", not "RepoPrompt" or "specstory". Adding a backend = implement the interface and register it in `internal/registry`; no edits to consumers.

**Backend resolution: the native default, config to override.**

- `.abcd/config.json` → `<seam>.backend` selects the backend; its **default is the seam's native path** — host-delegated for the `oracle` seam (adr-25), the native store/loop/scan for `history` / `spec` / `run` / `scanner`.
- The default requires no external tool: abcd works out of the box with no backend wired.
- An explicit value selects a wired external backend (richer capability, unusual setups, or testing).
- A missing or misbehaving external backend degrades to the native default rather than breaking abcd — each seam carries its own thin capability contract.

**Why the seam, not vendor coupling:** the vendor layer is implementation detail behind the interface, so every front door (adr-23) inherits whichever backend is configured, and no vendor's absence disables abcd. A second backend for a seam is additive — implement the interface, register it — and never touches the consumers (`review-collator`, `principle-distiller`, etc.).

## 8. Artefact-lifecycle taxonomy

abcd produces three classes of durable artefact, each with distinct curation rules. **Lifecycle class is declared in the parent README of each artefact namespace.** Lint blocks if a namespace's curation behaviour disagrees with its declared class (lint code reserved at `06-lint.md`).

**Three classes, three behaviours:**

| Class | Behaviour | Examples |
|---|---|---|
| **Regenerable** | Overwritten in place; regenerated from authoritative inputs on next run. Single canonical version at any time; history preserved separately if at all. | the lifeboat at `<dest>` (latest disembark snapshot only, out-of-tree per adr-35), `~/.abcd/voyage/<source-root-sha>/` cards, sota-audit findings, intent-fidelity audit reports |
| **Append-only** | Never modified after creation; new entries accrete; old entries preserved verbatim. | `.abcd/.work.local/logs/<command>/<timestamp>/` per-run reports, `~/.abcd/voyage/<source-root-sha>/disembark/history.jsonl`, capture issue-ledger entries (immutable post-create) |
| **Compounding-curated** | Accumulates across sessions/runs; pages added, modified, contradicted, deprecated by curator. Carries provenance per entry; lint surfaces drift between curated form and source-of-truth. | `.abcd/memory/` (multi-upstream knowledge substrate per itd-36), `.abcd/work/` (shared working record — CONTEXT.md, DECISIONS.md, the `.abcd/work/issues/` ledger), the brief itself |

**Why the taxonomy is load-bearing:** without it, "regenerable" and "compounding" get conflated, the curator agent (e.g., `principle-distiller` post-itd-36) loses its contract with consumers, and a pattern like itd-36's memory-unification looks like ceremony when it's actually a different lifecycle class than the lifeboat. Naming the three classes lets each artefact namespace declare its rules explicitly and lets cross-document fidelity audit (Role 2) catch drift.

**Recomputation discipline for regenerable artefacts.** Regenerable artefacts use **full-crawl-on-demand** recomputation, not incremental delta-application. Three failure modes that justify the discipline:

1. **Drift-without-detection.** Delta-application accumulates small bookkeeping errors silently; full crawl is stateless and idempotent.
2. **Schema fragility.** Schema bumps require re-interpreting every past delta; full crawl re-parses with the current schema each run, no museum of past schemas.
3. **False O(1).** Deltas that reference cross-cutting state ("this spec newly depends on spec-N's output") read back into the corpus, collapsing the O(1) claim where it matters most.

Cadence for regenerable artefacts: **on-demand + at phase-milestone boundaries**, NOT every state-change event. At current corpus sizes (10-30 specs × ~4 sub-bullets each), full crawl runs in seconds. Re-evaluate cadence if the corpus grows past ~50 entries; not before.

**Why compounding-curated is NOT regenerable.** A compounding artefact's value is the curated synthesis across upstream sources — it cannot be reconstructed from sources alone without the curator agent's accumulated decisions (which contradictions to surface, which sources to weight, which entries to deprecate). Regenerable artefacts are stateless functions of inputs; compounding-curated artefacts carry curator state.

