# `/abcd:auto-loop` — design plan for a generic autonomous burst-mode loop

**Status:** design plan recorded 2026-07-12 for **sign-off, not yet built**. This
doc is the design deliverable requested before any skill code is written. It
synthesises three inputs: the candidate prompt
(`Desktop/generic-start-prompt.md`, a strong base), the learnings note
([`../research/notes/2026-07-12-autonomous-loop-skill-learnings.md`](../research/notes/2026-07-12-autonomous-loop-skill-learnings.md)),
and its three-reviewer adversarial panel (verdicts: mechanics FLAWED, safety
UNSAFE, assessment OVERSTATED). The panel's MUST-FIX / CORRECTIONS /
Lower-severity sections are treated as **requirements**, resolved concretely
below. A `sota-researcher` pass backs the external-practice claims (§10).

## 1. What the capability is

A **generic** command that drives one repository through a run PLAN autonomously,
in timeboxed bursts, one PR per milestone, resuming from a durable handoff with no
memory of prior bursts. It is launched under the harness loop:

```
/loop 90m /abcd:auto-loop .abcd/development/plans/<dated-run-plan>.md
```

The command takes **exactly one argument: a path to a run PLAN** under
`.abcd/development/plans/`. Everything repo- or run-specific lives in the plan;
the command is the invariant protocol. This is the same generic/plan separation
the candidate prompt already nails — we keep it and push it further: the candidate
still had two placeholders baked into the prompt text (`{{PLAN_FILE}}`,
`{{TEST_COMMANDS}}`); here **the plan is the only placeholder** and the plan
itself carries the gate commands, DoD, ledger verb, reviewers, and STOP set.

## 2. Surface placement — where it lives (decided, with rationale)

**Decision: `/abcd:run`, the host-delegated realization of itd-29's surface** (see
§14 for the full sequencing decision — itd-29 already owns this surface and reserves
the `run` name; §13.0 for the ADR-27 reframe). When formalised (sequence step A), it
is a host-delegated command file `commands/abcd/run.md`, reconciled into itd-29. Until
then (step C) it lives as a run PLAN + protocol invoked under the harness loop.

- The repo's `/abcd:*` surface is `commands/abcd/<verb>.md`
  ([`brief/04-surfaces/README.md`](../brief/04-surfaces/README.md)). Most verbs are thin
  binary wrappers, but there is firm precedent for **prose-only agent-instruction
  commands** that call no binary: `prepare-this-repo.md`, `consult.md`,
  `ingest.md`. `/abcd:auto-loop` is that kind — an orchestration protocol, not a
  wrapper over an `abcd` sub-verb.
- It is *not* core behaviour (`internal/core` stays transport-agnostic and never
  orchestrates an LLM loop). The loop is host-delegated by design — consistent
  with the "host-delegated by default" boundary in AGENTS.md.

Rejected alternative — a `.claude/skills/` skill: that surface is harness-private
and not part of the published plugin marketplace; the whole point is that this
ships on the `/abcd:` surface like the others.

## 3. The two-part contract: what is generic vs what the plan carries

**Generic (in `auto-loop.md`, never parameterised):** the burst protocol (recover
state → pick next item → research-if-needed → TDD write+test+commit →
gate → risk-gated adversarial review → PR → checkpoint → stop); the STOP
conditions' *machinery*; the Attempts write-ahead journal; the delegation
discipline (lean orchestrator; workers return terse structured verdicts); the
chained-branch merge-commit-only mechanics and their fragility STOPs; the secret
scrub; the changelog-fragment convention; the NEXT.md handoff shape; the
per-burst time + token/fan-out budget.

**Plan-carried (a required template, §4):** Definition of Done / success criteria;
the exact gate commands; the ledger verb (`abcd capture …`); the review agents to
invoke; branch/PR/commit-trailer policy; the run-specific STOP conditions; the
backlog source; irreversible-step declarations; the milestone/item list with
**stable ids**.

## 4. Required run-PLAN shape (the template the command validates)

The command **refuses to start** if the plan lacks any mandatory block. The plan
is a normal dated design-record file; these headed sections are its contract:

```markdown
## Run contract          (MANDATORY — the command aborts if absent/malformed)
backlog: milestones | ledger        # §5 — where the work-list comes from
gate: <exact commands, newline-separated>   # the green bar, e.g. `make preflight`
definition_of_done: <the real acceptance bar; the gate is only the floor>
ledger_verb: abcd capture            # or "none"
reviewers:
  correctness: ruthless-reviewer     # always-on, every non-trivial diff
  security: security-reviewer         # trust-boundary diffs (default-on, §7.4)
branch_policy: <type>/<slug>; merge-commit-only; never force-push; never commit to default
commit_trailer: Assisted-by: Claude:<model>   # this repo's rule; "none" elsewhere
pr_policy: one PR per milestone; do not merge; do not enable auto-merge
budget: 30m wall | <N> worker-agents | <M> orchestrator-tokens   # §6
stop_conditions:                      # run-specific, ADDED to the generic set (§7)
  - <one line each>
irreversible:                         # §7.2 — always human-checkpointed
  - <migrations, history rewrites, data backfills, destructive cutovers>

## Milestones            (MANDATORY when backlog: milestones)
- id: M1  name: <...>  base: main            depends_on: []
- id: M2  name: <...>  base: M1              depends_on: [M1]
  # `id` is the STABLE strike-counter key (§7.1). `base` drives the chain (§7.5).
```

When `backlog: ledger`, the milestone list is replaced by a query the command runs
each burst (`abcd capture list --open --json`); each open `iss-N` is an item and
`iss-N` is its stable strike key. `depends_on` comes from the ledger's
`blocked_by`.

`GOAL.md` (candidate's authority doc) maps to the plan's `definition_of_done` +
this repo's durable record; there is no separate read-only GOAL file — the plan is
authoritative and revisable-but-not-in-premise (candidate's PLAN REVISION rule,
kept).

## 5. Backlog source — a real fork (recommendation inside)

Both real runs used a different source: the candidate prompt drives a
**plan-embedded milestone list** (greenfield feature work); the 2026-07-12
clean-slate run drained the **`abcd capture` ledger** (hardening/issue-burndown).
Rather than pick one, the plan declares `backlog:` and the command supports both:

- `backlog: milestones` — ordered milestone list in the plan; chain per `base`.
- `backlog: ledger` — `abcd capture list --open --json` each burst;
  derived-priority order; fold the `resolve` into the fix branch so the issue
  auto-resolves on merge (a hard-won learning — no trailing chore PR).

**Recommendation: support both, default to `milestones`.** Ledger mode reuses the
existing structured ledger (stable ids, dependency ordering, folder-as-status) we
already ship — no new machinery.

## 6. Budgets — bound context, not just time (resolves Lower-severity + note gap)

The candidate bounds only wall-clock (30 min). A burst that *implements several
milestones itself* accumulates context regardless of the clock — the exact
overflow the clean-slate run hit. So the plan declares three bounds and the burst
stops cleanly at the **first** one reached:

- **Time:** ≤30 min wall-clock (finish the current atomic unit, never truncate mid-change).
- **Fan-out:** ≤N concurrent/total worker agents per burst.
- **Tokens:** a soft orchestrator-context ceiling; on approach, write NEXT.md and stop.

Stopping on a budget is a *clean* stop (green, committed, handed off), never a STOP
condition (no human needed) — the next burst resumes.

## 7. Every MUST-FIX, resolved concretely

### 7.1 Dangling/died entry counts toward 3-strikes; strike key is a STABLE id

The candidate's 3-strikes counts only `→ FAILED` lines, so a step that reliably
**hangs/crashes** leaves an outcome-less line — treated as "investigate", never
counted — and loops forever. Fix, in the generic protocol:

- On burst start, read `## Attempts` **before** `## Next steps`. For the item about
  to be worked, count both `→ FAILED` lines **and** dangling (no-outcome) lines
  against the limit. **3 combined strikes → STOP.** A *second* consecutive dangling
  entry for the same id is itself a STOP (a reliable hang is not a flake).
- The counter is keyed on the item's **stable `id`** (`M3`, `iss-42`), never the
  free-text description (Lower-severity fix). The plan/ledger supplies the id;
  Attempts lines are `- [<ts>] <id> — trying: …`.

### 7.2 Irreversible changes ALWAYS get a human checkpoint + rehearsed rollback

The candidate exempts migrations "part of the plan" — but its own example M6 *is* a
backfill migration, so it would ship unattended. Fix: **naming a step in the plan
does not grant unattended execution.** Any step matching the plan's `irreversible:`
list — or the generic set (DB migration, data backfill, history rewrite,
destructive cutover, dropping/renaming persisted state) — triggers a **STOP with a
rollback rehearsal**: the burst prepares the change on a branch, writes the exact
forward + rollback commands into NEXT.md, and stops for a human. Never a
dual-write, never an untested revert (AGENTS.md risky-cutover rule).

### 7.3 Scrub secrets / local paths from pasted test output before it enters a PR

"Paste real test output" is kept (evidence beats assertion) but gated by a **scrub
pass** before any output reaches a PR body, commit, or issue: strip absolute local
paths (`/Users/…`, `/home/…`, `C:\…` → repo-relative), tokens/keys, real
hostnames/usernames/emails, and any `private-names.txt` match. This enforces the
non-negotiable privacy invariant. Mechanics that make it safe: always
`gh pr create --body-file` (inline `--body` shell-mangles backticks — a hard-won
learning), and the scrub runs on the file before it is passed.

### 7.4 Default-ON adversarial review gate before each PR

Green tests are not enough — the clean-slate run had **three** diffs pass TDD that
the security reviewer then BLOCKED. Before every PR:

- **Correctness reviewer (`ruthless-reviewer`) on every non-trivial diff** — always.
- **Adversarial security reviewer (`security-reviewer`) on trust-boundary diffs** —
  secrets, subprocess, network, input-parsing, file/DB, auth. The classifier is
  **conservative: default-to-review; only pure docs/comment/formatting diffs skip;
  when unsure, run it.** Mis-tagging must fail toward *more* review (the real
  same-line secret leak came from a diff nobody would pre-tag).
- Reviewers return a **terse structured verdict** (PROMOTE/HOLD + confirmed findings
  + `file:line`), never prose (§8), persisted as a **VSA receipt** under
  `.abcd/work/reviews/<sha>/` (pinned judge model + detector version), the shape abcd
  already uses. **The LLM verdict does not itself block** — per abcd's
  `verifier-selects-gates-decide`, admission stays deterministic: the release-gate
  (`record-lint --release-gate <sha> --require-gate <name>`) is **fail-closed on a
  PROMOTE receipt**, so a HOLD (or missing receipt) deterministically withholds the
  PR. Same effect as "BLOCK stops the PR", but the authority is the deterministic
  gate, never the probabilistic reviewer. Unresolved after the strike limit → STOP.
  See §13.2 — this corrects a conflict in the first draft.

### 7.5 Chain-fragility states → STOP-and-report, never a knowingly-broken chain

The chained-branch model is kept (merge-commit-only, GitHub auto-retarget — the
candidate's reasoning is sound), but three fragile states are added as STOPs
instead of "move on":

- **Upstream fix needed on an already-branched chain** (rebase forbidden): do not
  rewrite history to propagate it. Record in NEXT.md and STOP — a human decides.
- **Base PR closed-not-merged:** the downstream base vanished — STOP, do not retarget blindly.
- **Unpushed base** (its push failed twice): a chained child on a local-only base
  cannot open a valid PR. Record under `## Unpushed`, do not open the child PR, STOP if
  it blocks progress.
- **Squash/rebase-merge re-enabled on the repo:** the chain is no longer safe — STOP
  (candidate already had this; kept).

## 8. Corrected context principle — lean orchestrator, delegate reads/reviews (NOT implementation)

The note's first-draft headline "delegate ALL implementation" is **wrong** and the
panel was right: delegating the code-writing breaks TDD watched-fail (the
orchestrator would assert a red→green it never observed), the orchestrator-local
Attempts journal (workers don't share it, so 3-strikes dies), and commit
atomicity. The corrected principle, which the command encodes:

- **The orchestrator still writes, tests, and commits the code itself** — so it
  *observes* the watched-fail→pass, owns the Attempts journal, and keeps commits atomic.
- It **delegates read-heavy exploration and all REVIEW** to short-lived subagents
  that return a **terse structured verdict** (verdict + confirmed findings +
  `file:line`), never full prose. This is what bounds orchestrator context growth —
  not delegating implementation, and not "one issue per window" (a crude fallback,
  kept only as the budget-triggered fresh-context handoff of §6).
- Reviewers are subagents with **fresh context + external signal** (lint/SAST), never
  the implementer re-reading its own diff in the same context — Huang et al. (§10)
  show intrinsic self-review yields no gain or regresses; CriticGPT's benefit needs a
  *separate* lens. This is why review is delegated, not done in-orchestrator.
- **NEXT.md is written early and continuously**, not only at burst end — burst-end
  is the most cut-off-prone moment. Every attempt line gets an outcome before the
  burst yields.

## 9. Genuinely-good candidate parts kept intact

- **Attempts write-ahead journal** (record the approach *before* trying; dangling =
  died; never repeat a FAILED approach) — kept, with §7.1's dangling-counts fix.
  Lineage: Reflexion's episodic reflective memory (the don't-repeat-failures half) +
  database WAL semantics (the write-ahead-so-a-crash-is-recoverable half) — §10.
- **changelog.d/ fragments** (one file per change) — kept, but the false
  "structurally impossible [to conflict]" claim is corrected: **slug collisions are
  possible**, so the command guards them (suffix `-2`, `-3` on collision; the
  fragment README notes it).
- **Chained merge-commit-only branches + GitHub auto-retarget** — kept, with §7.5's
  fragility STOPs.
- **Two-placeholder → one-placeholder design, STOP conditions, research-first, TDD
  watched-fail, wired-or-dead reporting, paste-real-test-output** — all kept
  (research-first via `sota-researcher`; output scrubbed per §7.3).
- **Asymmetric marking** (delete completed items; keep FAILED attempts) — kept, but
  the Milestones `[PR #N]` ticks vs delete-completed two-regime drift
  (Lower-severity) is resolved: milestone status is **derived from git/PR state**,
  not hand-ticked; NEXT.md keeps only live items + the Attempts journal.

## 10. SOTA backing for adopted external practices

From a `sota-researcher` pass (primary sources, evidence-tiered). Verdicts recorded
in [dcn-2607120000000004](../../work/decisions/dcn-2607120000000004-abcd-auto-loop-design-recorded-plans.md); on build, the sources graduate to `ACKNOWLEDGEMENTS.md`
(inspirations/references) in the same change. The pass tempers, not overturns, the
design — three findings sharpen specific sections (flagged inline). Titles marked
[house convention] are ours, defensibly extrapolated, not directly measured.

- **Durable handoff + fresh-context resume — ADOPT (strongest-backed).** Anthropic
  *Effective harnesses for long-running agents* states compaction "isn't sufficient"
  for cross-window consistency and prescribes durable artifacts (progress log, git
  history, feature registry) that each burst re-reads — exactly our NEXT.md + git +
  ledger model. RAG-over-ledger is *rejected* below single-milestone-exceeds-context
  scale (adds a retrieval-miss failure mode, no source recommends it here).
  **Sharpening (new):** the same source finds models corrupt Markdown state more
  readily than JSON. Our handoff is Markdown by repo convention (NEXT.md) — see
  fork §11.4: keep Markdown, or store the *strike-critical* Attempts state as a JSON
  block the loop treats as append-only.
- **Delegate reads/reviews, keep implementation in ONE agent — ADOPT.** Both sides
  of the multi-agent debate converge: Cognition *Don't Build Multi-Agents* ("actions
  carry implicit decisions; conflicting decisions carry bad results"; Claude Code
  uses subagents for "answering a question, not writing any code") and Anthropic
  *Multi-agent research system* ("coding… fewer truly parallelizable tasks";
  many-dependency/shared-context tasks are a poor fit). The read/write boundary is
  SOTA-endorsed; the TDD-watched-fail / journal-ownership rationale (§8) is our house
  extension on top.
- **Orchestrator–worker with bounded structured output — ADOPT-WITH-CAVEATS.**
  Anthropic *Multi-agent research system*: subagents as "intelligent filters"
  returning results under an explicit "objective + output format + boundaries"
  contract. Caveat: ~15× token cost, ~80% of eval variance from token use — worth it
  only for genuinely read-heavy/parallel work, which is why our budget (§6) caps
  fan-out. "Return a verdict enum, not prose" is [house convention] atop their
  structured-output finding.
- **Adversarial/ensemble review before commit — ADOPT-WITH-CAVEATS.** OpenAI
  *CriticGPT* (arXiv 2407.00215): a critic lens catches more inserted bugs than paid
  human reviewers, and human+critic beats either alone. **Boundary condition
  (sharpens §7.4/§8):** Huang et al. *LLMs Cannot Self-Correct Reasoning Yet* (arXiv
  2310.01798, ICLR'24) — *intrinsic* self-correction (the implementer re-reading its
  own diff, same context) yields no gain or regresses. So reviewers **must** be
  separate-lens subagents with fresh context + external signal (lint/SAST), which
  §8 already mandates — now it's the cited reason, not just a preference. The
  *content-gating* of the security lens is [house convention] (cost control), not
  measured.
- **Bounded autonomy + HITL for irreversible actions — ADOPT (consensus, guidance-
  grade).** Anthropic guidance: pause at checkpoints/blockers; "unacceptable to
  remove or edit tests." **Sharpening (confirms §7.2):** gate on *action class* (DB
  migration, history rewrite, publish), never on the agent's self-rated confidence —
  RLHF models are miscalibrated (verbal confidence ↛ correctness). Our irreversible
  gate is action-class by construction. No controlled eval quantifies the incident
  reduction — adopt as consensus, claim no number.
- **Write-ahead attempt journal / N-strikes — ADOPT-WITH-CAVEATS (a composite).** The
  *reflect-so-you-don't-repeat* half has canonical prior art: Shinn et al.
  *Reflexion* (NeurIPS 2023), episodic reflective memory, ~8% absolute gain — cite as
  our lineage. The *write-ahead, dangling-entry-means-crashed* half is the database
  **WAL** pattern ported to an amnesiac agent — [house convention], well-grounded
  analogy, no agent-specific paper names it. N-strikes ↔ loop-detection is
  practitioner consensus, no strong eval.

**Rejected after investigation:** parallel multi-agent *implementation* (both
sources steer away; 15× cost buys nothing at one-PR-per-milestone); compaction as
the *primary* continuity mechanism ("isn't sufficient"); RAG-over-ledger at this
scale; confidence-threshold escalation (miscalibration).

## 13. Review against abcd (principles + current CLI) — tensions, must/should/could

Three read-heavy passes (CLI capability map, principles-vs-SOTA, review-surface
inventory) cross-checked this design against abcd itself. The headline: **abcd is
mostly ahead of the SOTA here and already owns the run contract** — but it forces
three corrections and adds a set of obligations. Everything is recorded below and
classified MUST / SHOULD / COULD for *this* build.

### 13.0 The load-bearing reframe (MUST)

**ADR-27 (`autonomous-run-pluggable-seam`) already defines what an autonomous run
IS:** iterate ready work → gate each step on a **receipt** → apply a **safety guard**
with shared stop conditions, and the loop is a pluggable **adapter** (host workflow /
companion harness / thin native Go fallback). Receipt-gating is **report-don't-block**
at each iteration boundary. So `/abcd:loop` is **the host-workflow adapter onto that
seam**, not a bespoke prompt. Consequence: the design must speak ADR-27's vocabulary
(run / ready-work / receipt / safety-guard / stop-conditions) and reuse its
artefacts, not invent parallel ones.

### 13.1 Tension table — every SOTA finding × abcd, resolved

| SOTA finding | abcd verdict | Resolution | Tier |
|---|---|---|---|
| (A) durable handoff + fresh-context resume | ALIGNED (receipts, directory-as-truth) | reuse the seam's durable artefacts | MUST |
| (A′) JSON > Markdown for corruptible state | **CONFLICT** with Markdown-handoff convention | **split**: prose handoff = Markdown (ADR-30/docs-lint); machine-parsed journal/strike state = JSON, matching abcd's own machine-state-is-JSON rule | MUST |
| (B) keep implementation in one orchestrator | ALIGNED (ADR-25 host-delegated; ADR-27 thin loop) | orchestrator = the run adapter; never embeds model access itself | MUST |
| (C) terse structured worker output | ALIGNED (VSA receipts, `verifier-selects-gates-decide`) | verdicts persist as receipts, advisory not authoritative | MUST |
| (D) separate-lens adversarial review before commit | ALIGNED in shape, **CONFLICT on authority** | LLM review may **not** be the blocking gate; it emits a PROMOTE/HOLD **receipt**, the *deterministic* release-gate is fail-closed on it (§13.2) | MUST |
| (E) action-class STOP + human checkpoint | STRONGLY ALIGNED (ADR-27 safety-guard; global rules) | keep; **add** sota-per-intent's two hard-stops: new dependency, bespoke-no-seam | MUST |
| (F) write-ahead journal + N-strikes | ALIGNED, constrained | journal = working-tier `.abcd/.work.local/` (gitignored, per-worktree), JSON strike state, append-only; N-strikes limit *is* a stop condition | MUST |

### 13.2 The three genuine conflicts (and how each is resolved)

1. **Review-as-authority (was §7.4 "a BLOCK stops the PR").** abcd's
   `verifier-selects-gates-decide` + `enforcement-claims-are-facts` forbid a
   probabilistic verdict from deciding admission. **Fixed:** the reviewer writes a
   PROMOTE/HOLD **VSA receipt** under `.abcd/work/reviews/<sha>/`; the *deterministic*
   `record-lint --release-gate <sha> --require-gate <name>` is fail-closed on a PROMOTE
   receipt. Same protective effect, abcd-correct authority. (§7.4 amended.)
2. **Markdown state corruption (the user's NEXT example).** Resolved as the A′ split
   above — not "Markdown vs JSON" but "prose vs machine-state", which is already how
   abcd draws the line. (§10, §11.4 amended.)
3. **Reviewers don't travel with the repo.** `ruthless-reviewer`, `security-reviewer`,
   and the `code-review` **Standards** axis are **USER-level**, not repo-committed;
   only `intent-fidelity-reviewer` + the deterministic gates ship in-repo. A loop that
   assumes them violates **wired-or-it-isn't-done**. **Fixed:** the plan *declares* its
   reviewers; the command resolves them via ADR-25 host-delegation and **degrades
   loudly** (`loud-staging`) to the deterministic gates + `intent-fidelity-reviewer` +
   security-review slash-command when a named reviewer is absent — never silently
   skips the lens.

### 13.3 Should the loop add a "review against repo conventions" lens? — YES, as a SHOULD

The `code-review` skill's **Standards axis** *is* the repo-conventions LLM lens (reads
`AGENTS.md`/`CONTRIBUTING.md` + a Fowler-smell baseline; explicitly skips what tooling
already enforces). Against abcd's deterministic gates (`gofmt`, `vet`, `record-lint`,
`docs-lint`) + a correctness reviewer it is **not redundant** — it catches naming that
reveals intent, module-boundary/altitude conventions, API-shape consistency, and
"documented convention X but code does Y" where X isn't machine-checkable.

Recommendation: **add a Standards/conventions lens as a third reviewer** — but (i)
advisory receipt, not a blocking authority (same as §13.2.1); (ii) gated to
non-trivial code diffs; (iii) told to skip anything the deterministic gates cover;
(iv) it partly overlaps the correctness reviewer, so it is a **SHOULD**, not a MUST —
worth it because abcd's whole value proposition is convention-conformance, but the
first cut can ship with correctness + security and add Standards next. Note the
repo-committed structured cousin already exists for the **Spec** half:
`intent-fidelity-reviewer` (per-criterion MET/NOT_MET, ingested by `abcd intent review
ingest`) — the loop should use *that* for spec-fidelity rather than a prose Spec pass.

### 13.4 `/abcd:loop` gated vs `--dangerously-skip-gates` — reshaped, not adopted as-worded

Your instinct is right that there are **two modes**, but the axis is **human-checkpoint
cadence, not gate-skipping**. In abcd "gates" = the *deterministic* gates (build / vet
/ test / record-lint / docs-lint); skipping those is exactly `--no-verify`, which
AGENTS.md forbids without exception, and would gut `guards-prove-themselves`. So a flag
literally named `--dangerously-skip-gates` is a **hard no** — it names the one thing
that must never be skipped.

The coherent two-mode design (recommended):

- **`/abcd:loop` (default = supervised).** Bounded autonomy: runs the full pipeline but
  **stops for a human at each PR / milestone boundary** (and at every STOP condition).
  This is ADR-27's report-**and-pause** posture — the safe default.
- **`/abcd:loop --dangerously-unattended` (the candidate's burst mode).** Keeps opening
  chained PRs across milestones without pausing at each boundary — "the human is not at
  the machine." What it skips is the **per-milestone human checkpoint**, never a
  deterministic gate and never a review receipt.

**Invariant across both modes (non-negotiable):** deterministic gates run every commit;
review receipts are emitted; and the **irreversible-action checkpoint (§7.2) always
fires regardless of the flag** — `--dangerously-unattended` cannot buy through a
migration/history-rewrite/publish. The `dangerously` prefix mirrors the ecosystem's
`--dangerously-skip-permissions` convention: it means "I accept unattended shipping",
not "I accept unverified code". (Fork §11.4 records the naming choice.)

### 13.5 abcd obligations the design must also honor

| Obligation (principle) | What it requires of this build | Tier |
|---|---|---|
| **wired-or-it-isn't-done** | reachable + demonstrated from CLI **and** plugin markdown surface; reviewer-absence degrades loudly (§13.2.3) | MUST |
| **spec-moves-with-the-surface** | land the brief row (`brief/04-surfaces/`) + `surface_coverage` in the same change | MUST |
| **prefer-sota / sota-per-intent** | the loop is itself an intent: carry a `## SOTA` block (the 6 findings + maturity + adopt/seam/bespoke), each having passed the adversarial fit-challenge (this review *is* that) | MUST |
| **guards-prove-themselves** | every STOP condition + the safety guard ships a test watching it refuse (must-pass/must-flag corpus) | MUST |
| **enforcement-claims-are-facts / loud-staging** | don't describe any gate/stop as running until it does; stage unwired parts loudly | MUST |
| **personas Alice/Bob/Carol + they/them** | any example/story in the command or plan template | SHOULD |
| **host-agnostic prose (docs-lint)** | command's user-facing text names no specific harness without the `allow` escape | SHOULD |
| **one-canonical-primitive** | reuse `internal/fsutil` durable-write/frontmatter for the JSON journal; don't fork atomic-write | SHOULD |
| **fix-the-detector** | stop-condition escapes arm a detector with the escaping case as fixture, not hand-patched | COULD |
| **capture→intent, in-progress/claim, priority field, changelog-fragment, NEXT verb** | CLI gaps the loop must work *around* today (derive ordering from `blocked_by_open`; no concurrent claim; fragment tooling is the loop's, not the binary's) | COULD (note as gaps) |

### 13.6 Net effect on the earlier sections

§7.4 (review authority) and §10/§11.4 (handoff format) are **amended above**. §2
(surface + naming), §5–§9 stand. The build surface is now: `commands/abcd/loop.md`
(the ADR-27 host adapter), a run-plan template, the brief-row + `surface_coverage`
registration, `commands/README.md` line, `ACKNOWLEDGEMENTS.md` entries, and STOP/guard
tests — no new dependency, no core change, no binary sub-verb (the seam is host-side).

## 11. Open decisions for sign-off

1. **Backlog source default** (§5): support both, default `milestones`? *(rec: yes)*
2. **Budget defaults** (§6): 30 min kept; fan-out cap *(rec ≤6 workers/burst)* + soft
   token ceiling *(tune on first run)* — your numbers.
3. **Reviewers** (§7.4, §13.2/§13.3): correctness + security *(rec: declared in plan,
   host-delegated, degrade loudly)*; **add the Standards/conventions lens now or
   next?** *(rec: next — ship correctness+security first)*; use
   `intent-fidelity-reviewer` for the Spec half *(rec: yes, it's repo-committed)*.
4. **Mode + naming** (§13.4): two modes `/abcd:loop` (supervised default) +
   `--dangerously-unattended` *(rec)*; reject a literal `--dangerously-skip-gates`.
5. **Command name** (§2): `/abcd:loop` vs `/abcd:run` (ADR-27 vocabulary; avoids
   loop-under-`/loop` confusion). *(rec: lean `/abcd:run`, your call)*
6. **Handoff format** (§13.1 A′): resolved — prose Markdown + JSON machine-state split.
   Confirm.

## 14. Blocking discovery — itd-29 already owns this surface (build paused)

Surfaced at build time, before any command code was written: the autonomous-run
surface is **already planned as itd-29 (`autonomous-run-resilience`, `intents/planned/`)**
and the brief registry **reserves the name `run`** for it (operator-internal;
`status/pause/resume/preflight` v1 cut; no `commands/abcd/run.md` shipped). itd-29:

- is a **binary** operator surface (`abcd spec start/status/pause/resume/rewind/ship`),
  not a host-delegated markdown command;
- is **deliberately deferred** — "failure modes of a system that doesn't yet exist…
  design against real evidence, not guesses"; implementation waits on substrate +
  revisit triggers;
- already scopes the **out-of-band-merge/chain reconciliation** (this plan's §7.5 —
  resolved there as host-owns-git MVP → a future read-only `abcd run reconcile --json`)
  and **429/quota** handling (delivered by spc-35);
- names as **revisit trigger #5**: "two consecutive autonomous runs succeed
  end-to-end, proving the run seam works" — i.e. exactly the evidence a working loop
  generates.

**Consequence:** this loop is best understood as the **ADR-27 host-workflow engine
over the seam** — the manual precedent itd-29 says must run first — *not* a competing
surface. **Naming decided (maintainer, 2026-07-12): `/abcd:run`** — consciously the
**host-delegated realization of itd-29's surface**, taking its reserved `run` name
rather than inventing a parallel `loop`. Because itd-29's binary operator surface is
deliberately deferred pending real evidence, the sequence is **C→A**:

- **C (now):** run the loop as a **plan + protocol under the harness loop** (no shipped
  command yet) to dogfood on abcd-cli and generate itd-29's evidence (revisit trigger
  #5). This ships nothing ahead of evidence — honouring itd-29's own discipline.
- **A (after 1–2 successful runs):** formalise `commands/abcd/run.md` as the
  host-delegated realization, reconciled into itd-29 (link the shipped host-workflow
  cut; keep the deferred binary verbs — budget preflight, rewind, ship,
  `abcd run reconcile --json` — deferred), with its brief row + `surface_coverage`.

The host-delegated `/abcd:run` v0 does **not** reimplement itd-29's deferred binary
bits; chain reconciliation uses host-side `gh`+`git` (the "host owns the git" MVP
itd-29 already blesses).

## 12. Scope guard — what this design does NOT do yet

No skill/command code is written until this design is signed off. Everything above is
design. When approved, the build is one command file plus template + registration +
acknowledgements + guard tests, on one branch/PR, following this repo's commit/PR/
attribution rules.
