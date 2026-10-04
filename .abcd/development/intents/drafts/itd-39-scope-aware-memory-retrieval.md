---
id: itd-39
slug: scope-aware-memory-retrieval
spec_id: null
kind: standalone
suggested_kind: standalone
reclassification_history: []
blocked_by: [itd-36, itd-3]
severity: minor
---

# abcd Surfaces the Right Memory at the Right Moment

## Press Release

> **abcd ships scope-aware memory retrieval so the project's accumulated knowledge reaches you exactly when it's relevant — and never floods your context when it isn't.** Memory pages carry `recall` keywords. The same `UserPromptSubmit` hook that injects modular rules (itd-3) also scans each prompt against memory pages across two scopes — the workspace and the user's personal store — and injects only the matching pages. Injection is budget-aware: with a fresh context window abcd surfaces full pages; as the window depletes it falls back to one-line index entries, then to nothing. Memory stops being an all-or-nothing wall and becomes a just-in-time lookup.
>
> "Our `.abcd/memory/` had grown to 140 pages across two years," said Carol, tech lead. "Loading all of it was impossible and loading none of it meant agents kept relearning the same pitfalls. Now a prompt about the auth module pulls the three auth pages and nothing else — and when I'm deep in a long session it quietly drops to titles-only so it never crowds out the actual work."

## Why This Matters

abcd's `.abcd/memory/` (itd-36) is a strong *storage* model — typed pages, a curator, quotation budgets, contradiction tracking. But it has no *retrieval* model beyond `/abcd:memory ask`, an explicit user query. A working agent gets memory all-or-nothing: either the whole corpus is injected (overflow) or none is (relearning). itd-36 bounds *ingest*; nothing bounds *retrieval* or *total surfaced size*.

This intent closes that gap by reusing infrastructure abcd already has. itd-3's modular rules loader is abcd's in-plugin adoption of [CARL][carl]'s recall engine — a `UserPromptSubmit` hook with domain-keyed keyword recall, signature dedup, and force-refresh-every-N. CARL's *same* engine drives two payloads: procedural rules and a decision-logging memory layer. abcd adopted the rules half and skipped the memory half. itd-39 adopts the half abcd skipped — and adds CARL's context brackets (adaptive injection by remaining window) as the overflow control itd-36 lacks.

## What's In Scope

- **Memory `recall` frontmatter** — every `.abcd/memory/` page's `source:` frontmatter gains a `recall:` keyword list (the domain-keying CARL applies to rules, applied to memory pages).
- **Hook extension** — `hooks/prompt_router_hook.py` (itd-3) is extended from rules-only to rules + memory. Same recall scan, same signature dedup, same force-refresh-every-N. The hook reads two indexes: `rules.json` domains and memory-page `recall:` frontmatter.
- **Context brackets** — injection is gated by remaining context budget, after CARL's model: `FRESH` (≥70% window free) injects matched pages in full; `MODERATE` (40–70%) injects only highest-relevance pages; `DEPLETED` (<40%) injects matching `index.md` lines only (titles, not bodies). This is the bounded-retrieval counterpart to itd-36's bounded-ingest quotation budgets.
- **Two-scope query** — the hook queries the `index.md` of both memory scopes (`<workspace>/.abcd/memory/` and `~/.abcd.noindex/memory/`). On a recall-keyword conflict, the narrower scope wins (workspace > user). The agent never loads the union of both scopes — it loads keyword-matched, budget-bracketed pages only.
- **`abcd memory recall [keyword]`** CLI subcommand — diagnostic: shows which pages a given prompt/keyword would surface, and in which bracket. Bare-command-as-render discipline (per `02-constraints/04-naming.md`).

## What's Out of Scope

- **Session-sticky recall** — itd-3 establishes per-prompt independent recall (no cross-prompt chaining). itd-39 inherits that model. A "pin this memory for the rest of the task" mode is a deferred candidate.
- **Writing to vendor memory** — the hook injects *from* `.abcd/memory/`. It never reads or writes `~/.claude/.../memory/`; that directory is a `dev-sync` harvest source only (see `02-adapters.md`).
- **Automatic scope routing of new pages** — *which* scope a curated page lands in is a `dev-sync` / curator decision (see `07-memory.md` scope-routing rule), not a retrieval-hook concern.
- **A separate memory MCP server** — consistent with itd-3, retrieval is hook + JSON/frontmatter only; no runtime memory-editing MCP in this intent.

## Reconciliation with itd-3's global-rules rejection

itd-3 explicitly rejects a global `~/.abcd.noindex/rules.json` — "a personal cross-repo rules file recreates the scaffolding-accumulates-outside-the-repo failure mode." itd-39 *does* place a memory scope at `~/.abcd.noindex/memory/`. This is deliberate, not a contradiction:

- **Rules are procedural and project-shaped.** A rule that applies everywhere belongs in the plugin defaults, not a personal file — hence itd-3's per-repo-only stance.
- **Memory is observational and inherently cross-project.** A personal working preference ("I prefer X phrasing in commit messages") or a principle that improves abcd development itself is *not* project-scoped knowledge — it has no per-repo home by nature. Forcing it into a repo would lose it on the next project.

The failure mode itd-3 guards against is *unbounded scaffolding outside the repo*. itd-39's scopes are bounded by the two-scope `.abcd/` namespace (see `05-internals/03-configuration.md`) and surfaced by bounded retrieval — they do not accumulate silently into every session. The economics differ; the scopes are justified.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline._

- **Given** an abcd-installed workspace with a memory page declaring `recall: [auth, login]`, **when** the user sends a prompt containing "auth", **then** that page is injected as system context within the turn and non-matching pages are not.
- **Given** a context window below 40% free, **when** a memory recall matches three pages, **then** only the matching `index.md` lines are injected (titles, not bodies), per the `DEPLETED` bracket.
- **Given** the same prompt repeated within a session, **when** dedup is enabled, **then** the matched memory injects once and skips on repeats until the force-refresh interval.
- **Given** a recall keyword that matches a page in both the workspace and user scope, **when** the hook resolves the match, **then** the workspace-scope page is injected and the user-scope page is suppressed (narrower scope wins).
- **Given** any prompt, **when** the hook runs, **then** `~/.claude/.../memory/` is never read or written by the retrieval path.

## Open Questions

- Should the three context-bracket thresholds be shared with itd-3's rule injection or tuned separately for memory? (Rules are smaller per-item than memory pages.)
- Does the `DEPLETED`-bracket "titles-only" fallback need a relevance ranking, or is recall-keyword match count sufficient?
- Release placement is deferred until itd-36's storage model has shipped and accumulated real corpus.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Deferred to by: `itd-42` (coherence-aware grill) — itd-42's Tier 3 is index-level only; full-body cross-intent comparison at scale is this intent's selective-retrieval problem.

[carl]: https://github.com/ChristopherKahler/carl "CARL — Context Augmentation & Reinforcement Layer (Kahler)"
