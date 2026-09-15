# Internals — Plumbing That Makes Surfaces Possible

This directory holds the *plumbing* design for abcd: the agents, adapters, configuration, universal patterns, and prompt-quality stack that the user-facing commands ([`04-surfaces/`](../04-surfaces)) depend on. None of this is user-facing in itself; it exists to *enable* user-facing capability (see [`01-product/03-mental-model.md`](../01-product/03-mental-model.md) for the why-plumbing-is-not-an-intent argument).

| # | File | Purpose |
|---|---|---|
| 1 | [`01-agents.md`](01-agents.md) | The fifteen shipped agent prompts and the design roster still to be built, each with declared JSON inputs/outputs; the host-delegated oracle seam (adr-25) |
| 2 | [`02-adapters.md`](02-adapters.md) | The central adapter model (adr-22): the five capability seams (oracle \| history \| spec \| run \| scanner) — interface + native default + optional external plug-in — plus the lifeboat source readers |
| 3 | [`03-configuration.md`](03-configuration.md) | `config.json` schema, including its `meta` setup block (seam-backend + adapter-registry config); visibility-driven gitignore policy; `dev-sync` namespace |
| 4 | [`04-universal-patterns.md`](04-universal-patterns.md) | Cross-cutting patterns shared by all commands (host-delegated oracle, native-with-peer-interop, the adapter seam model, artefact-lifecycle taxonomy, …) |
| 5 | [`05-prompt-quality.md`](05-prompt-quality.md) | Prompt-quality infrastructure (B+C+D + itd-5 prompt-quality additions) |
| 6 | [`06-lint.md`](06-lint.md) | Lint contract: the intent/prompt/terminology checkers in `internal/core/lint`, severity model, CI integration |
| 7 | [`07-memory.md`](07-memory.md) | `.abcd/memory/` component — multi-upstream curated knowledge substrate per itd-36; page-class enum; curator role on `principle-distiller`; quotation/licence lints. Karpathy LLM Wiki pattern as prior art. |
| 8 | [`08-skills.md`](08-skills.md) | Skills-vs-commands boundary: codifies decision criteria for later skill additions. abcd ships zero skills — `/abcd:grill` was originally proposed as one; its **design target** is promotion to `/abcd:intent grill` (mid-session glossary writes are command-shaped), not a shipped sub-verb. |
| 9 | [`09-provenance-substrate.md`](09-provenance-substrate.md) | Provenance/licence subsystem (used by both itd-36 memory ingest and itd-26 loot OSS-vendor). Licence detection (SPDX), citation generation, source-hash registry, restrictive-licence publish gate (lifeboat consumer, future/inert at launch — adr-28). Separable spec; pulled forward alongside itd-36. |
| 10 | [`10-site.md`](10-site.md) | The website as a rendered surface of the record (adr-47/adr-48): `abcd site build` and the `record.json` export, the single-source rule, the README migration, the generic/specific boundary (itd-140). The `check` gates and the explorer pages ship and the deploy workflow rides the release chain (adr-48); the one **design target** marked on the page is the abcdev.app root, which still serves the MkDocs rendering until the first tagged deploy moves it. |

## Policy: no skeleton enforcement (deferred)

The brief skeleton (numbered-folder layout under `.abcd/development/brief/`) is **not enforced** by `intent-auditor` or `documentation-auditor`. This is a deliberate choice — rigidity on a young template kills iteration; better to let the shape settle through use, then enforce.

**Future enforcement candidates** (deferred, not yet tracked in the ledger):

- Require a populated `01-product/01-press-release.md` (no placeholder content)
- Require at least one entry in `04-surfaces/`
- Require `02-constraints/01-platform.md` to exist (even if minimal)
- Warn if `05-internals/` is populated on a fresh `ahoy`-bootstrapped brief (signals over-specification — fresh briefs should let agents design plumbing, not pre-specify it)

When `intent-auditor` matures, brief-skeleton presence checks would land alongside its existing intent-vs-implementation checks.

## Plumbing has no user moment

Per the three-layer mental model: plumbing infrastructure (adapters, agents, hooks, scaffolding) lives in the brief, not in intents. Intents require a user moment — a customer quote, a "Bob staff engineer says…" — and plumbing has no such moment. It exists to make user-facing work possible. The brief is the right home for plumbing because it speaks to designers and contributors, not customers.
