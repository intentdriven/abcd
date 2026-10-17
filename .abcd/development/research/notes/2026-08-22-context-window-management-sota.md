# SOTA survey — context-window management for agent sessions

Dated 2026-08-22. Compiled from two host-run web research passes and one
repo-survey pass. Method per
[`prefer-sota`](../../principles/prefer-sota.md): each technique is named as
the presumptive SOTA and then challenged for fit against this repo's
conventions and shipped state. The evidence-tier vocabulary used throughout
(published benchmark / vendor eval / vendor-recommended / production
practice / community consensus / research tier / position piece / anecdote)
is this note's own ladder, strongest to weakest — the principle prescribes
the challenge discipline, not a tier canon.

**Why this note.** The standing question is how a configuration layer on top
of an agent harness keeps a task from ever exhausting the model's context
window. The field's answer has consolidated enough to survey: the consensus,
in Anthropic's phrasing, is to *find the smallest set of high-signal tokens
that maximises the likelihood of the desired outcome* — overflow avoidance
falls out of that discipline rather than being pursued directly. This note
records the consensus stack, the evidence behind each layer, where this repo
already implements it, and where the gaps are. A second pass (§§9–11) extends
the survey to token estimation, capacity-based task sizing, and deterministic
overflow protocols.

## The organising result — context rot

Quality degrades long before the window fills. Chroma's context-rot study
measured non-uniform accuracy degradation as input grows, well before the
window limit, across 18 frontier models (evidence tier: published benchmark;
the study reports no single degradation percentage — the finding is the
non-uniformity). The NoLiMa benchmark puts 11 of 13 models below 50% of
their short-context performance at 32k tokens. Practitioner reversals point
the same way: accounts of shrinking a 1M-token window back to 200k report
fewer hallucinations, less goal drift, and better instruction retention
(anecdote tier). The corollary that frames everything below: **window size
raised the overflow ceiling, not the quality ceiling** — the target is staying
on the good side of the rot curve, not fitting under the limit.

## Ranked techniques

### 1. Files, not the transcript, carry state

The strongest consensus in the field. Anthropic's long-running-harness
guidance prescribes a progress log file, a structured feature list with
pass/fail state, git commits as recoverable state, and a session-start
checklist (read the progress file, read the git log, verify, then work)
(vendor-recommended). HumanLayer's ACE-FCA — "frequent intentional
compaction": research document → plan document → implement, holding context
utilisation in the 40–60% range — is the strongest practitioner report,
credited with shipping 35k lines to a 300k-line Rust codebase in one
seven-hour session (anecdote tier: self-reported, no controlled baseline). Manus independently converged on "the file system as ultimate
context" plus **recitation** — rewriting a todo file so goals stay in recent
attention (production practice).

*Fit:* the three-tier `.abcd/` layout is this pattern —
`.abcd/.work.local/NEXT.md` (handover), `.abcd/work/DECISIONS.md`
(append-only decision log), `.abcd/work/CONTEXT.md` (status-free
orientation). The decision record already adopts durable handoff +
fresh-context resume over compaction ([dcn-2607120000000004](../../../work/decisions/dcn-2607120000000004-abcd-auto-loop-design-recorded-plans.md)).
The gap is ownership, not design: no verb writes NEXT.md, no schema
validates it, no lint checks freshness, and it is lost on a worktree move
(finding F6 in `../../plans/2026-07-16-run-findings.md`). SOTA practice makes
the handoff write a *protocol trigger* — on stop signal, on budget approach,
continuously during runs — not a courtesy.

### 2. Compaction is a failure mode to pre-empt, not a strategy

Auto-compact fires deep inside the rot zone — practitioner reports place the
default trigger between 80% and ~95% of the window, overridable via the
harness's autocompact-percentage setting — and its documented failure modes
hit precisely when a task is longest and
most valuable: early constraints silently dropped, path-scoped rules gone
until a matching file is re-read, and thrashing when a chatty tool refills
context faster than summaries reclaim it (community consensus: DoltHub's
gotchas write-up, multiple practitioner posts, harness issue trackers).
Anthropic itself warns that aggressive compaction loses "subtle but critical
context whose importance only becomes apparent later". The practitioner
counter-moves split two ways: trigger fresh sessions before auto-compact
fires and rebuild from the state files of §1 (the ACE-FCA 40–60% band), or
keep auto-compact but fire it earlier — the same practitioner who reverted to
a 200k window lowers the auto-compact trigger to 70% and reports that working
well, so earlier-compaction remains a live alternative, not a straw man.
Clear-and-reload is lossless *provided* the state files are current — a
precondition the tooling has to enforce (gap 3), not assume; summarisation
carries no equivalent guarantee. Community analysis of the harness's
compaction internals reads its session-memory tier as pre-extracted notes
rather than a summarisation call at the cliff (anecdote tier: third-party
analysis, not vendor-confirmed).

*Fit:* the rules loader already treats compaction as the threat model — the
`SessionStart`/`PreCompact` dedup-ledger reset is recorded as "the single
most important correctness point in the design"
(`../../plans/2026-07-11-itd-3-rules-loader.md` §5). But the shipped
`PreCompact` hook only deletes the ledger file. It is the richest signal
available — the last moment before state is erased — and nothing durable is
written there: no orientation snapshot, no NEXT.md refresh, no memory page.
The itd-3 plan's "record what was live" instruction was superseded for rule
purposes — deleting the ledger makes every live domain re-inject on the next
prompt — but nothing broader took its place: there is no compaction-survival
contract beyond rule re-injection.

### 3. Read fan-out, single writer

The one genuinely contested area; the resolution below is a reading of what
shipped, not a measured comparison. Anthropic's multi-agent
research system reported +90.2% over single-agent on breadth-first research,
with token spend explaining 80% of variance — at roughly 15× the tokens
(vendor eval). Cognition's "Don't Build Multi-Agents" argued parallel
sub-agents make conflicting implicit decisions because context is not shared
(position piece). The working resolution visible in what shipped across the
field: **fan out reads, never writes.** Sub-agents burn their own context on
searching, reading, and reviewing, and return distilled conclusions;
synthesis and editing stay single-threaded so no implicit decision is made
outside the main context. Cognition's essay itself concedes the mechanism:
a sub-agent's value is that "its investigative work does not need to remain
in the history of the main agent".

*Fit:* already doctrine here. The auto-loop design
(`../../plans/2026-07-12-abcd-auto-loop-skill.md` §8) keeps the orchestrator
as the single writer and delegates read-heavy exploration and all review to
short-lived subagents returning terse structured verdicts — "this is what
bounds orchestrator context growth". The prompting research
(`../prompting/01-general-best-practices.md`) prescribes 1000–2000-token
distilled returns as "the defence against parent-context rot". Gap: the cap
is prescribed in research prose but no agent definition under `agents/`
carries an output *token budget* — verdict schemas do ship there (the
ruthless-reviewer's structured findings format, the intent-auditor's single
JSON verdict block); the missing half is the size contract.

### 4. Conditional injection and progressive disclosure over always-on context

Every always-on token is paid every turn and contributes to rot — and the
evidence says it also degrades behaviour, not just cost. Anthropic's deferred
tool loading measured an 85% token reduction on tool schemas with selection
accuracy *up* (Opus 4: 49%→74%; Opus 4.5: 79.5%→88.1% on their benchmark)
(vendor benchmark); code-execution-over-MCP reduced one scenario from 150k to
2k tokens; Agent Skills' three-level loading (name → SKILL.md → resources) is
the canonical progressive-disclosure design. The mirror image for rules:
keyword-triggered injection with per-session dedup, a small always-on core,
and an explicit override for the recall-miss case.

*Fit:* this is the rules loader (itd-3), shipped: zero model-facing tokens on
a no-match, per-domain FNV-1a signatures as dedup keys, event-driven refresh,
`*DOMAIN` explicit activation as the recall-miss mitigation. The external
evidence now validates what the intent argued from first principles. Gap: no
instrumented measurement — the loader reports bytes on stderr, but nothing
counts tokens and no eval records the actual saving (the intent's anecdote is
a 978-line CLAUDE.md reduced to a ~30-line marker block).

### 5. Observation pruning — bulky artefacts become pointers

Anthropic's context-editing API let agents complete 100-turn workflows that
otherwise died of context exhaustion (−84% tokens; +29% on an agentic eval
alone, +39% combined with the memory tool) (vendor eval). Independent
corroboration: JetBrains Research's observation masking measured −52% cost
with +2.6% solve rate on SWE-bench-style tasks for its strongest model (the
paper's general finding: simple masking is competitive with LLM-summarisation
approaches). Two production refinements
from Manus: prune the *content* but keep the *fact that the action happened*,
and deliberately keep errors in context — models repeat mistakes they cannot
see. Most of this is host-side; a configuration layer adopts it by not
fighting it: keep hook outputs small and idempotent, and route bulky
artefacts to files so the transcript holds paths, not payloads.

*Fit:* the `.abcd/.work.local/logs/`-and-`scratch/` rule is the file-routing
half. Hook outputs are already minimal and byte-stable. Nothing to build;
this layer's job is to preserve the discipline.

### 6. Persistent memory plus note-then-reread

The memory tool — files outside the context that persist across sessions —
is what pushed Anthropic's combined context-management improvement to +39%
(vendor eval). The broader trajectory (Anthropic's managed-agents
architecture stores context as an external event log the agent queries by
slice) confirms the direction: **context is becoming a database the agent
reads, not a buffer it fills**. The common gap in practice is write
discipline — when notes get taken — rather than storage.

*Fit:* storage is shipped (`internal/core/memory/`: typed pages, ingest,
ask, lint, single-writer discipline per adr-0013), and the explicit `ask`
verb ships a token-overlap ranking; what is not built is the recall-driven,
budget-bracketed injection path. The
`recall:` frontmatter field is parsed and round-tripped but nothing consumes
it, and the context-bracket design (FRESH/MODERATE/DEPLETED by remaining
window) exists only in the itd-39 draft. Memory does not currently
participate in compaction survival — nothing writes a page at session end or
on `PreCompact`. Ledger: iss-2608221254560250 records that
`docs/reference/terminology.md` overstates the shipped state.

### 7. Cache-aware layout — stable, append-only prefix

Production-proven at Manus, load-bearing for cost and latency rather than
overflow: agent workloads run near 100:1 input:output; cached tokens are
roughly 10× cheaper; a single early divergent token invalidates everything
after it. Corollaries: never mutate earlier turns, mask rather than remove
tools mid-session, append injections rather than editing the system prompt,
keep injected blocks byte-stable.

*Fit:* the prompt-router injects by appending at `UserPromptSubmit` and its
rendered blocks are byte-stable by construction (the dedup signature is the
hash of the rendered block). Compliant; nothing to build.

### 8. Retrieval on demand over pre-built indexes — contested, with a scale asterisk

The one place real data conflicts with real data. Anthropic's position and
the harness architecture it ships: agentic search (grep/glob just-in-time,
progressive disclosure of files) beat their RAG prototypes and is far simpler
to operate; paths and naming act as implicit metadata (vendor + production).
Cursor's counter-evidence: purpose-trained embeddings gave +12.5% average QA
accuracy over grep-only, with semantic + grep combined performing best
(vendor eval — on their custom-trained embedding model, not off-the-shelf).
The pragmatic reading (two vendor data points, no independent head-to-head):
the measured semantic gains required platform-scale investment in a trained
model and continuous index maintenance; at solo and
small-repo scale, agentic search wins on operational simplicity, while hybrid
is plausibly the endgame for large-monorepo products.

*Fit:* the repo already rejected RAG-over-ledger at single-milestone scale
([dcn-2607120000000004](../../../work/decisions/dcn-2607120000000004-abcd-auto-loop-design-recorded-plans.md)). The survey confirms the
rejection and adds the boundary condition under which it would be revisited:
monorepo scale plus a trained embedding model — neither applies here.

## Token estimation, capacity sizing, and overflow

The second research pass, scoped to what a configuration layer on a
coding-harness host can adopt. One finding cuts across all three sections:
the vendor's server-side `task_budget` beta is explicitly not supported on
the coding-harness surface, so a configuration layer there must implement
budgeting itself.

### 9. Token estimation — calibrate distributions, never point estimates

The strongest negative result in the field: on SWE-bench Verified across
eight frontier models, identical tasks vary up to 30× in total tokens across
runs; models' self-predictions of their own usage correlate at best 0.39 with
actuals and systematically underestimate (arXiv 2604.22750, published
benchmark). BAGEN corroborates: capability and budget-awareness correlate at
only r=0.35, and even after fine-tuning, budget-interval coverage peaks at
47% (arXiv 2606.00198). The practice that survives this evidence is
**distribution-based calibration**: record per-task spend from telemetry,
size budgets from percentiles — the vendor's own task-budgets guidance is to
measure first and start at the p99 of per-task spend (vendor-recommended).

What is solved: counting tokens in a *static* payload (prompt + files +
tools) via the free, model-specific `count_tokens` endpoint. What is
explicitly wrong for Claude: tiktoken (undercounts ~15–20% on typical text
and much more on code, per the vendor's own skills repo) and the chars/4
heuristic (no published error bound for Claude; a rough approximation at
best, worst on code and JSON). Three different numbers must not be
conflated: *window occupancy* (what overflow cares about — cache reads
occupy the window fully despite being billed at 0.1×), *billed tokens*, and
*budget-relevant spend* (the per-turn view the server's budgeting uses).

Live tracking: the host's statusline/hook JSON carries a `context_window`
object (window size, used percentage, remaining percentage, usage split into
input/output/cache categories) — production-proven with caveats: a
cumulative-session-versus-current-occupancy counting bug was reported and
has since been fixed (closed May 2026), while 1M-window sessions still
misreport against a 200k denominator (open at the time of writing). Durable
history for the calibration corpus comes from the harness's OpenTelemetry
export (`claude_code.token.usage`) or community usage tooling. A
pre-execution estimator for a planned task inside the harness does not ship
— the feature request was filed and closed as a duplicate of a standing
request, with no implementation (genuine gap in the field).

*Fit:* this resolves gap 4's adoption route — statusline/hook gauge for live
approximation (cross-checked, never sole truth), telemetry for the
calibration corpus. The run findings' measured worker costs (~78k–169k,
`../../plans/2026-07-16-run-findings.md`) are exactly the per-task-type
calibration data this prescribes; the repo already holds a first sample
set, uncollected as a corpus.

### 10. Sizing tasks to capacity — small by construction, not measured first

The dominant practice is **one bounded task per fresh context, state on
disk** — vendor best practice (clear between unrelated tasks; split
research/plan/implement into separate sessions) and,
independently, the community's stronger form: the Ralph Wiggum loop
(Geoffrey Huntley, mid-2025) — exactly one task per iteration, the agent
restarted with an empty context, all state (spec, plan, done-list) persisted
to files (community practice, widely replicated). Utilisation bands govern
when to act: community practice clusters around acting at ~60–70%
utilisation, well before default auto-compact — though the corpus agrees
only on "well before", not on a number: the ACE-FCA report (§1) works lower
still, in its 40–60% band. The threshold values are anecdote tier — no
controlled study picks one — but the direction is evidence-backed:
NoLiMa's degradation onset, and the Aider maintainer's remark that models
get confused above ~25–30k tokens as users' top complaint (secondhand: a
maintainer comment circulated through practitioner posts, no primary
write-up in the sources below).

Sub-agent budgeting: the production pattern is prompt-declared effort
scaling (the multi-agent research system embeds explicit rules — one agent
and 3–10 tool calls for simple fact-finding, scaling to 10+ agents for
complex research — because agents cannot judge effort themselves) plus
output contracts (workers burn tens of thousands of tokens but return
1,000–2,000-token distilled summaries). Mechanical enforcement exists only
as turn caps (`max_turns` → typed exception in the OpenAI Agents SDK;
`recursion_limit` → typed error in LangGraph); token-level per-worker caps
enforced mechanically exist only at enterprise gateway layer.

The rule this repo's gap 1 would want — "if estimated > X% of window, split
before starting" — is a **genuine gap in the field**: no vendor doc or
credible practitioner source publishes one, and §9 explains why (the
estimate it depends on is the least reliable number in the space). The
field's actual answer is to make tasks small by construction. A
percent-threshold preflight is therefore a design decision to own, not
adopted practice.

*Fit:* the split-the-intent doctrine sizes on verifiability; the survey's
verdict is that the capacity axis arrives not via per-task estimates but via
(a) small-by-construction decomposition — which the intent doctrine already
does, needing only the session-sized framing — and (b) calibration
percentiles per task type (§9). The per-plan "one item per burst" prose
lines are exactly the B-pattern, unenforced. The 1,000–2,000-token return
contract matches the repo's own prompting research; enforcement remains the
open piece (gap 5).

### 11. Deterministic overflow protocols — loud stop, no retry loop

Design constraints: overflow is never silently absorbed, and never produces
an endless retry of the same oversized task. The published floor, from the
vendor's API surface (vendor-documented): pre-4.5 models reject an oversized
request with a validation error; 4.5+ models accept it and stop
generation with a structured `stop_reason: "model_context_window_exceeded"`;
the server-side compaction beta can pause with `stop_reason: "compaction"`
as a deterministic, loud, resumable checkpoint; and the `task_budget` beta
is an advisory countdown — not enforced, remaining budget not exposed, and
unavailable on the coding-harness surface. These stop reasons are the only
published precedent for "budget stop as a structured status distinct from
error". The harness's default auto-compact, by contrast, is precisely the
quiet degradation the constraint forbids (§2).

Framework consensus is the typed hard stop (`MaxTurnsExceeded`,
`GraphRecursionError`): it satisfies both constraints trivially but
conflates "budget hit" with "failure" — the exception carries no partial
result or handoff state, a documented practitioner complaint. The measured
improvement over hard stops: BAGEN's early-stop alerts save 28–64% of tokens
on failed trajectories, but the model's budget self-report is poorly
calibrated (47% interval coverage), so it can only ever be a *signal into* a
deterministic harness check, never the check itself (research tier). From
the reliability literature, transferred to agents at blog tier: retries live
in exactly one layer (the orchestrator), a shared attempt budget rather than
per-step retries, and a task that exhausts its attempts goes to a
dead-letter state for human triage instead of requeueing.

The reporting taxonomy — a three-way terminal status of success / failure /
budget-stop-with-handoff, with an exit code and run-journal schema — is a
**genuine gap**: no vendor or framework ships it; the nearest working
practice is the Ralph community's file-based handoff, which is convention,
not schema.

The minimal deterministic protocol the evidence supports, for a
configuration layer: (1) calibrate per-task-type spend distributions from
telemetry and budget at a percentile (evidence-backed, vendor-recommended);
(2) declare the budget as a utilisation band plus a mechanical attempt cap
(band value anecdote tier; typed cap framework consensus); (3) monitor via
the host gauge, treated as approximate and cross-checked (production-proven
with known bugs); (4) on threshold, stop deterministically and hand off —
write state to disk, requeue the *remainder* as a new, smaller fresh-context
task, never the same task; after N budget-stops on one task, dead-letter it
to the human (handoff community-proven; requeue and dead-letter pattern
transfer); (5) report with the three-way terminal status modelled on the
API's structured stop reasons (ahead of published practice — the gap a layer
adopting it would be first-ish to fill).

*Fit:* the auto-loop design already specifies most of this, unbuilt:
`../../plans/2026-07-12-abcd-auto-loop-skill.md` §6 declares three bounds
with first-reached-wins, a soft orchestrator-token ceiling whose breach
means "write NEXT.md and stop", and the exact status distinction the field
lacks — "a budget stop is a *clean* stop, not a STOP condition". The
loud-staging principle (`../../principles/loud-staging.md`) is adjacent in
spirit — staged or degraded behaviour must announce itself, never a false
green — though it governs staged code, not run outcomes; the
terminal-status taxonomy has no in-repo articulation beyond that clean-stop
line. The auto-loop plan keeps the supporting binary verbs ("budget
preflight, rewind, ship, run reconcile") deliberately deferred; itd-29
(planned) puts a standalone preflight in its first cut. Adopting the five-step
protocol is therefore mostly wiring existing design, not new design — with
the attempt-cap/dead-letter piece the one genuinely new element, for which
the issue ledger's folder-as-status model is the natural home.

## Conflicts and their current resolutions

- **Multi-agent fan-out vs single thread** (Anthropic vs Cognition):
  resolved as read fan-out, write single-thread (§3). Neither original
  position survives intact.
- **Bigger windows vs disciplined small contexts:** 1M-token windows exist;
  context rot, NoLiMa, and practitioner reversals resolve this as: long
  context is for genuinely irreducible inputs, not a substitute for
  engineering.
- **Compact vs clear:** compaction is the vendor's safety net for the
  unplanned case; the deliberate mechanism is clearing plus file-based
  reload (§1, §2).

## Investigated and rejected for this repo's scale

- **A vector/embedding index of the codebase** — §8; the gains require
  platform-scale investment.
- **Relying on default auto-compact as the context strategy** — §2; it fires
  in the rot zone and its failure modes hit the longest tasks hardest.
- **Parallel writing sub-agents** — §3; the conflicting-implicit-decisions
  failure mode is confirmed from both camps' own reports.
- **Always-on mega-rulesets** — §4; directly contradicted by the
  tool-loading accuracy data.
- **Managed-agents-style external event-log context stores** — the right
  trajectory, but a platform product; nothing a configuration layer can or
  should replicate today.
- **tiktoken or chars/4 for Claude token budgets** — a wrong tokenizer
  (~15–20% undercount on text, worse on code) or an unbounded
  approximation; `count_tokens` or measured history instead (§9).
- **Story-point-style per-task token estimates** — 30× run variance and
  ≤0.39 self-prediction correlation make point estimates theatre; percentile
  bands per task type or nothing (§9).
- **Client-side mirroring of the server's `task_budget` countdown** — the
  vendor warns it double-counts resent history, makes the model wrap up
  early, and invalidates the prompt cache; moot on the coding-harness
  surface anyway, where the beta is unsupported (§11).
- **Enterprise gateway token enforcement at this scale** — the mechanically
  enforced ceiling is real but the operational cost is not justified; an
  attempt cap in the repo's own ledger gives most of it (§11).
- **Treating the statusline `used_percentage` as sole truth** — open
  correctness bugs (cumulative-vs-current, 1M misreporting); cross-check
  against telemetry or transcript-derived counts (§9).

## Gaps — ranked by leverage

1. **Task sizing is verifiability-based, never capacity-based.** The intent
   doctrine splits on "two separately verifiable promises = two intents";
   nothing asks whether an item fits one context window. "One item per
   burst" is a prose line in individual plan files, unenforced. The repo's
   own post-mortem named the failure exactly: no per-slice context budget,
   so one run packed ~12 checkpoints across 9 issues into a single window
   and overflowed (`2026-07-12-autonomous-loop-skill-learnings.md`). The
   run findings supply real calibration data: four measured worker contexts
   of ~78k–169k tokens, with ~169k recorded as "the upper end of what one
   worker context comfortably holds"
   (`../../plans/2026-07-16-run-findings.md`). Caveat:
   the host's statusline/hook payload does expose a live context gauge
   (`context_window`: window size, used percentage, per-category usage), but
   its correctness record warrants cross-checking (§9), so sizing combines
   the gauge with
   proxies — measured per-item costs, turn counts, one-item-per-session
   protocol; the itd-39 draft frames its brackets the same way.
2. **`PreCompact` writes nothing durable** (§2) — the compaction-survival
   contract is undefined beyond rule re-injection, and the hook seam already
   exists.
3. **NEXT.md has no owner** (§1) — no verb, no schema, no freshness lint,
   lost on worktree moves (F6).
4. **No token accounting anywhere** — bytes on stderr is the only
   observability; `disembark.maxAgentTokens` is documented in the brief with
   no code behind it (iss-2608221254566264); `evals/` holds only a smoke
   test, so the rules loader's saving is asserted, not measured.
5. **Designed but unbuilt retrieval and return caps** — itd-39's brackets
   (§6) and the 1000–2000-token distilled-return rule (§3) exist as design
   prose only.

## Review record

Round 1 (2026-08-22, one fresh-context adversarial reviewer): NEEDS_WORK,
11 findings (3 major) — every falsified claim was an external number; all
in-repo claims verified. All applied. Round 2 (2026-08-22, after the
§§9–11 extension; two reviewers, disjoint lenses): source fidelity
NEEDS_WORK, 7 findings; repo fidelity and reasoning NEEDS_WORK, 7
findings; no overlap between the lenses. All applied. (Session-reported
counts; the reviewer transcripts are local ephemera.)

## Sources

- [Effective context engineering for AI agents — Anthropic](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)
- [Effective harnesses for long-running agents — Anthropic](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)
- [Scaling Managed Agents — Anthropic](https://www.anthropic.com/engineering/managed-agents)
- [Managing context on the Claude Developer Platform — Anthropic](https://claude.com/blog/context-management)
- [Context engineering cookbook — Claude Cookbook](https://platform.claude.com/cookbook/tool-use-context-engineering-context-engineering-tools)
- [Don't Build Multi-Agents — Cognition](https://cognition.com/blog/dont-build-multi-agents)
- [Context Rot — Chroma research](https://www.trychroma.com/research/context-rot)
- [Context Engineering for AI Agents: Lessons from Building Manus](https://manus.im/blog/Context-Engineering-for-AI-Agents-Lessons-from-Building-Manus)
- [Advanced Context Engineering for Coding Agents — HumanLayer](https://github.com/humanlayer/advanced-context-engineering-for-coding-agents/blob/main/ace-fca.md)
- [Context engineering with Dex Horthy — Pragmatic Engineer](https://newsletter.pragmaticengineer.com/p/context-engineering-with-dex-horthy)
- [Improving agent with semantic search — Cursor](https://cursor.com/blog/semsearch)
- [Why I'm Against Claude Code's Grep-Only Retrieval — Milvus](https://milvus.io/blog/why-im-against-claude-codes-grep-only-retrieval-it-just-burns-too-many-tokens.md)
- [Why I Shrunk Claude Code's Context Window Back to 200k — Albert Sikkema](https://www.albertsikkema.com/ai/development/tools/2026/04/23/smaller-context-window-better-claude-code.html)
- [Claude Code Gotchas — DoltHub](https://www.dolthub.com/blog/2025-06-30-claude-code-gotchas/)
- [Stop Claude Code from Lobotomizing Itself Mid-Task — Ian Paterson](https://ianlpaterson.com/blog/stop-claude-code-from-lobotomizing-itself-mid-task/)
- [Inside Claude Code's Compaction System — Decode Claude](https://decodeclaude.com/compaction-deep-dive/)
- [Scaling MCP Tools with Anthropic's Defer Loading — Unified.to](https://unified.to/blog/scaling_mcp_tools_with_anthropic_defer_loading)
- [How and when to build multi-agent systems — LangChain](https://www.langchain.com/blog/how-and-when-to-build-multi-agent-systems)
- [How we built our multi-agent research system — Anthropic](https://www.anthropic.com/engineering/multi-agent-research-system)
- [How Anthropic built its multi-agent research system — ByteByteGo summary](https://blog.bytebytego.com/p/how-anthropic-built-a-multi-agent)
- [The Complexity Trap: simple observation masking — JetBrains Research (arXiv:2508.21433)](https://arxiv.org/abs/2508.21433)
- [NoLiMa: long-context evaluation beyond literal matching (arXiv:2502.05167)](https://arxiv.org/abs/2502.05167)
- [Claude Code auto-compact mechanics — howaiworks.ai](https://howaiworks.ai/blog/claude-code-auto-compact-context-management)
- [Claude Code Subagents: A 2026 Practical Guide — Tembo](https://www.tembo.io/blog/claude-code-subagents)
- [Task budgets — Claude Platform docs](https://platform.claude.com/docs/en/build-with-claude/task-budgets)
- [Compaction — Claude Platform docs](https://platform.claude.com/docs/en/build-with-claude/compaction)
- [Context windows — Claude Platform docs](https://platform.claude.com/docs/en/build-with-claude/context-windows)
- [Token counting — Claude Platform docs](https://platform.claude.com/docs/en/build-with-claude/token-counting)
- [Token counting for Claude — anthropics/skills](https://github.com/anthropics/skills/blob/main/skills/claude-api/shared/token-counting.md)
- [Claude Code best practices](https://code.claude.com/docs/en/best-practices)
- [Claude Code statusline docs](https://code.claude.com/docs/en/statusline)
- [Claude Code monitoring/OpenTelemetry docs](https://code.claude.com/docs/en/monitoring-usage)
- [ccusage statusline](https://ccusage.com/guide/statusline)
- [How Do AI Agents Spend Your Money? (arXiv 2604.22750, via alphaXiv)](https://www.alphaxiv.org/overview/2604.22750)
- [BAGEN: are LLM agents budget-aware? (arXiv 2606.00198)](https://arxiv.org/abs/2606.00198)
- [ContextBudget/BACM (arXiv 2604.01664)](https://arxiv.org/abs/2604.01664)
- [Claude Code context management — SitePoint](https://www.sitepoint.com/claude-code-context-management/)
- [The Ralph Wiggum loop — codecentric](https://www.codecentric.de/en/knowledge-hub/blog/the-ralph-wiggum-loop-autonomous-code-generation-with-a-fresh-context)
- [Ralph playbook — paddo.dev](https://paddo.dev/blog/ralph-wiggum-playbook/)
- [The Ralph Wiggum agent loop as engineering discipline — alteredcraft](https://writing.alteredcraft.com/p/the-ralph-wiggum-agent-loop-is-really)
- [OpenAI Agents SDK Runner reference](https://openai.github.io/openai-agents-python/ref/run/)
- [AgentCore behaviour and cost controls — AWS](https://aws.amazon.com/blogs/machine-learning/control-agent-behaviors-and-cost-beyond-a-single-action-new-capabilities-in-amazon-bedrock-agentcore/)
- [Circuit breakers in AI agent systems — meganova](https://blog.meganova.ai/circuit-breakers-in-ai-agent-systems-reliability-at-scale/)
- [Fault-tolerant AI agent pipelines — MightyBot](https://mightybot.ai/blog/fault-tolerant-ai-agent-pipelines/)
- [Tokenizer counting guide — Propel Code](https://www.propelcode.ai/blog/token-counting-tiktoken-anthropic-gemini-guide-2025)
- [Claude API errors](https://platform.claude.com/docs/en/api/errors)
- [Session management and 1M context — Claude blog](https://claude.com/blog/using-claude-code-session-management-and-1m-context)
- [Claude Code issue #55779 — pre-execution token estimate (closed as duplicate; no shipped implementation)](https://github.com/anthropics/claude-code/issues/55779)
- [Claude Code issue #13783 — statusline cumulative-vs-current (fixed May 2026)](https://github.com/anthropics/claude-code/issues/13783)
- [LangGraph documentation — recursion limit and GraphRecursionError](https://langchain-ai.github.io/langgraph/)
- [Claude Code issue #76751 — 1M-context statusline misreport](https://github.com/anthropics/claude-code/issues/76751)
