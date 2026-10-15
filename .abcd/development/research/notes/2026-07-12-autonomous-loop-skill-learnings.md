# Autonomous hardening loop — learnings + SOTA principles for `/abcd:auto-loop`

Source: the 2026-07-12 clean-slate hardening run (9 issues fixed/merged in one
session; run log in `.work.local/clean-slate-run-prompt.md`). Purpose: distil what
a *generic* autonomous-loop skill must encode, separated from repo-specific config,
and record where a run-prompt drifts from SOTA. Feeds the proposed `/abcd:auto-loop`.

## What a generic auto-loop skill owns (vs. what stays repo-specific)

**Generic (belongs in the skill):**
- The *loop protocol*: STEP 0 discovery/triage → per-item pipeline (fix → gate →
  adversarial-verify → PR) → checkpoint → fresh-context handoff → resume.
- Explicit **STOP conditions** (design decision, contract/API change, new dependency,
  red gate not self-caused, an "issue" that is really a new feature).
- The **review discipline** (test-first; two-lens adversarial verify; a BLOCK stops
  the change) and the **gates** (build/test/lint before every commit).
- The **context/handoff protocol** (the big gap this run hit — see below).
- Merge-pickup mechanics (fetch/prune, ff, delete branch, rebase-forward, re-gate).

**Repo-specific (must be PARAMETERS the skill takes, not hard-coded):**
- Success criteria / Definition of Done, the gate commands (`make preflight`,
  `make record-lint`), the ledger verb (`abcd capture`), branch/PR/commit-trailer
  policy, the reviewer agents available, the toolchain. The current run-prompt bakes
  these in — a generic skill must lift them into a small config block the caller fills.

## SOTA assessment (what the run-prompt got right, and gaps)

SOTA for autonomous coding agents (orchestrator–worker loops, agentic harnesses):
short-lived worker contexts + durable external memory; adversarial/ensemble
verification before commit; deterministic gates over prose; test/detector-first;
bounded autonomy with loud STOP conditions; structured cross-session handoff.

**Aligned (keep):**
- Detector/test-first with a *watched-fail-then-pass* rule — SOTA and it worked
  (every fix pinned; reverts verified).
- **Two-lens adversarial verification** (a correctness reviewer AND an adversarial
  security reviewer) — earned its keep repeatedly: the security lens caught THREE
  real BLOCKs the correctness pass *and* the tests missed (a same-line secret
  cross-leak, a FIFO-leaf hang in a hot hook path, a spec_id validator diverging
  from the lint contract). Single-reviewer or tests-only would have shipped them.
- Deterministic gates before every commit; STOP conditions; ledger-as-truth.

**Gaps vs SOTA (the run-prompt under-specifies these; they caused the failure mode):**
1. **No per-slice context budget → context-window overflow.** The run did ~12
   checkpoints/9 issues in ONE window. SOTA is short-lived contexts: the skill MUST
   mandate **one issue per context window, then a genuine fresh-context resume**, not
   a same-window continuation. The `ScheduleWakeup` pause must be a handoff boundary.
2. **Unbounded reviewer output.** Each of ~20 review subagents returned full multi-KB
   analyses into the main context. SOTA: workers return a **terse structured verdict**
   (verdict + only confirmed findings + file:line), not prose. Use a schema.
3. **Whole-file echoes on branch churn.** Switching main↔branch made the harness
   re-echo entire 1000+ line files. SOTA: **one branch per session**, minimise
   switches, and delegate file-heavy reading to subagents (conclusions only return).

## Session learnings (operational, hard-won)

- **Fold the ledger resolve into the fix branch** so an issue auto-resolves on merge
  (no trailing chore PR). One-time catch-up PR only for already-merged fixes.
- **CHANGELOG merge conflicts** from parallel fix-PRs: resolve by **merging main into
  the branch** (not rebase — avoids force-push, honours the no-force-push rule), keep
  both bullets. Root fix is **changelog fragments** (towncrier-style; already iss-24)
  so parallel PRs never touch one file — a generic loop over many small PRs needs it.
- `gh pr create` with backticks/`~` in an **inline `--body` gets shell-mangled** —
  always `--body-file`.
- A **branch-delete `git push` mid-TDD hits the pre-push gate hook** (runs the full
  test suite, which is red mid-TDD) — delete via the host API
  (`gh api -X DELETE repos/<owner>/<repo>/git/refs/heads/<branch>`).
- **Interactive `cp -i` silently declines** in the non-interactive shell — a
  restore-after-revert-verify can leave the file in the reverted state; re-apply with
  the editor, and re-run gates to confirm restoration.
- **Partial resolution is honest**: a multi-instance issue (iss-30) can land its
  clear subset and STAY OPEN with the remainder recorded in [dcn-2607120000000003](../../../work/decisions/dcn-2607120000000003-iss-30-memory-ingest-boundary-partially.md) — do not
  fold a false full-resolve.
- **Scope the security fix to the actual invariant**: an over-broad validator that
  *diverges from an existing gate* (spec-store anchored id-regex vs record-lint's
  prefix rule) is itself a BLOCK — it bricks a lint-green record. Make cooperating
  gates agree.

## Proposed `/abcd:auto-loop` shape (sketch)

The skill is GENERIC and takes **one argument: a path to a run PLAN doc** under
`.abcd/development/plans/`. That plan (the per-run contract, versioned in the design
record) carries the repo/plan-specific parts: the Definition of Done / success
criteria, the gate commands, the ledger verb, the review agents, the branch/PR/commit
policy, the STOP conditions, and the backlog source. The skill reads the plan and runs
the protocol above with a **hard per-slice budget**: STEP 0 once, then **exactly one
issue per invocation**, ending each turn by writing the handoff and stopping — the
caller (or a scheduled wakeup) re-invokes with a fresh context. Durability lives in
the plan + the ledger + the `NEXT.md` handoff, never in a single long context. (This
also means the run log / learnings should graduate back into the plan or a research
note, not only a gitignored `.work.local` file.)

## Adversary review — accepted corrections (the verdict above was OVERSTATED)

An adversary pass challenged the SOTA claims; these corrections stand:

- **The real root fix is a LEAN ORCHESTRATOR, not "one issue per window."** The
  overflow's causes were unbounded reviewer prose and whole-file echoes, not
  issue-count. Correct primary principle: **delegate ALL implementation + heavy
  reading to workers; the orchestrator holds only structured verdicts + ledger
  state** — then many issues per window scale fine. "One issue per window" is a
  crude fallback, not the design. (This directly indicts how *this* run was
  driven: the main loop did the implementation itself.)
- **Add context compaction/summarisation and RAG-over-the-ledger** — "fresh-context
  resume" is the crudest member of that family, not the SOTA one.
- **Two-lens review is not a universal default — RISK-GATE it.** Security lens only
  on trust-boundary diffs (secrets/subprocess/network/parsing/auth); docs and
  pure-refactor slices should not pay for an adversarial security pass. (This run
  already did this partially — iss-72 got ruthless only.) The "caught 3 BLOCKs"
  point is suggestive, not a proven base rate (n=1; no count of clean diffs paid
  for).
- **Detector-first / gates-over-prose / ledger-as-truth are abcd conventions**, not
  established universal SOTA — present them as *our opinionated defaults*, not law.
- **Missing dimensions a generic auto-loop needs:** a per-slice token/cost budget;
  an eval of the loop's OWN output quality (revert rate, regression rate, review
  turnaround); a kill-switch / rollback path; a human-in-the-loop checkpoint when a
  STOP fires; and non-determinism handling (flaky-gate policy, reviewer-disagreement
  arbitration).

**Revised verdict:** the operational learnings are solid, but the SOTA framing was
overstated. The skill's headline principle should be **lean-orchestrator delegation
with bounded worker output + risk-gated review**, with fresh-context handoff as a
safety net, not the main mechanism.

## Assessment of the candidate generic prompt (Desktop/generic-start-prompt.md)

**Strong — already ahead of our clean-slate run-prompt on:**
- Two placeholders only ({{PLAN_FILE}}, {{TEST_COMMANDS}}); acceptance criteria live
  in the plan. Exactly the generic/plan separation we want.
- **Attempts journal** (write-ahead, survives bursts; never repeat a FAILED approach;
  a dangling entry means the prior burst died) — solves cross-burst amnesia; SOTA-grade.
- **Asymmetric marking** (delete completed, keep failed attempts) — avoids NEXT.md drift.
- **changelog.d/ fragments** — structurally removes the CHANGELOG conflicts we hit (our iss-24).
- **Chained milestone branches, merge-commit-only, GitHub auto-retarget** — correctly
  reasoned (squash strands the chain + needs a forbidden force-push).
- STOP conditions, GOAL.md read-only, plan-revision-but-not-premise, research-first, TDD
  watched-fail, wired-or-dead reporting, paste-real-test-output.

**Two MATERIAL gaps — both confirmed by this session:**
1. **No lean-orchestrator / delegation mandate → it WILL overflow context on a
   multi-milestone burst, exactly as this run did.** The 30-min limit bounds TIME, not
   CONTEXT; a burst that implements several milestones itself accumulates the same way.
   ADD: "Delegate implementation and file-heavy reading to subagents; the orchestrating
   burst holds only conclusions + NEXT.md state. Subagents/reviewers return a TERSE
   structured verdict (verdict + confirmed findings + file:line), never full prose."
2. **No adversarial review before the PR — it trusts TDD alone.** This session's TDD
   PASSED on three diffs the security reviewer then BLOCKED (same-line secret leak, a
   FIFO-leaf hang, a spec_id validator diverging from the lint gate). ADD a RISK-GATED
   review step: a correctness (ruthless) reviewer on every non-trivial diff, PLUS an
   adversarial security reviewer on trust-boundary diffs only (secrets / subprocess /
   network / input-parsing / auth); a BLOCK stops the PR.

**Minor:** add a per-burst token/cost budget (complements the time bound); a
reviewer-disagreement/flaky-gate arbitration rule; and when it becomes /abcd:auto-loop,
map .work/ → abcd tiers (.abcd/.work.local/NEXT.md, .abcd/development/plans/,
.abcd/work/DECISIONS.md) and optionally source the backlog from the `abcd capture` ledger.

## Three-reviewer adversarial panel on the prompt (verdicts: mechanics FLAWED · safety UNSAFE · assessment OVERSTATED)

**MUST-FIX before it becomes /abcd:auto-loop:**
1. (mechanics, high) **Dangling-entry infinite loop**: 3-strikes counts only FAILED
   lines; a step that reliably hangs/crashes leaves an outcome-less line treated as
   "investigate", never counted → loops forever — the exact wedge the journal exists
   to stop. Fix: a died/dangling entry counts toward the strike limit (or 2nd dangling
   → STOP).
2. (safety, high) **Irreversible migration ships unattended**: STOP-3 exempts
   migrations "part of the plan"; the example M6 IS a backfill migration. Migrations
   need a human checkpoint / rehearsed rollback regardless of the plan.
3. (safety, high) **No secret scrub on "paste real test output"** → local paths/tokens
   leak into PR bodies (violates our privacy invariant). Redact before pasting.
4. (safety+mechanics+ours) **No adversarial diff review before PR** — green tests
   aren't enough (this run: 3 BLOCKs past TDD). Add review with a DEFAULT-ON gate.
5. (mechanics) **Upstream fix can't propagate to an already-branched chain** (rebase
   forbidden) → knowingly-broken downstream; and **base-PR-closed-not-merged / unpushed
   base** strand the chain. Handle these states (STOP-and-report, don't "move on").

**CORRECTIONS to this note's own two headline additions (the panel was right):**
- "Delegate ALL implementation" is WRONG — it breaks TDD watched-fail (orchestrator
  asserts red→green it never saw), the orchestrator-local Attempts journal (workers
  don't share it → 3-strikes dies), and commit atomicity. CORRECT: delegate read-heavy
  exploration and REVIEW to workers (bounded output); the orchestrator still
  writes+tests+commits, so it observes watched-fail and owns the journal. The context
  fix is bounded worker OUTPUT + delegated READS/REVIEWS, not delegated implementation.
- "Risk-gate the security reviewer" needs a CONSERVATIVE classifier — mis-tagging skips
  the lens that caught the real bugs (the same-line secret leak came from a diff nobody
  would pre-tag). CORRECT: default-to-review; only pure docs/comment diffs skip; when
  unsure, run it.

**Lower-severity:** free-text item name keys the strike counter (use stable ids);
changelog.d slug collisions are possible (the "structurally impossible" claim is
false); STOP-5 is a whitelist (silent on rm of generated dirs, the "didn't create THIS
burst" loophole, mass codemods); the plan itself is trusted (a harmful plan step runs
unchecked); no token/fan-out budget; NEXT.md written at burst-END is the most
cut-off-prone point; two marking regimes (delete-completed vs Milestones [PR#] ticks) drift.
