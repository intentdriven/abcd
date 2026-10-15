# Run findings — planned-intent drain (manual test of the future `/abcd:run`)

**Status:** live findings log for the run driven by
[`2026-07-12-abcd-run-protocol.md`](2026-07-12-abcd-run-protocol.md) over
[`2026-07-16-planned-intent-drain-run.md`](2026-07-16-planned-intent-drain-run.md).
This run doubles as a manual test of the C-phase protocol before it becomes
`commands/abcd/run.md` (itd-29 reconciliation). Every burst appends here, as
findings occur, not at burst end.

Classification: **protocol gap** (the generic protocol needs a rule) ·
**plan gap** (the PLAN template needs a field) · **tooling gap** (the abcd
binary should do what the agent improvised) · **human stop** (a boundary
autonomy must not cross).

---

## Burst 1 (2026-07-16, M1 itd-89/spc-4 — shipped as PR #83, then STOP)

### F1 — Shared working tree with a concurrent agent · protocol gap + human stop

The plan's coordination rules were branch-scoped ("never touch a branch you
did not create"), but the concurrent bug-hunter edited the **same checkout's
working tree** while the run's branch was checked out — uncommitted foreign
edits accrued mid-burst across five packages. Branch discipline cannot protect
a shared checkout: any `git checkout`/`add`/`stash` risks sweeping in or
clobbering a second writer's work.

- The run survived because the item's files happened not to overlap
  (path-scoped staging), then stopped — correctly — rather than continue.
- **For `/abcd:run`:** (a) the protocol must require an exclusive checkout —
  own worktree or clone — as a *precondition*, not a coordination note;
  (b) cheap detector: snapshot `git status --porcelain` at burst start and
  re-check before every commit/checkout; any foreign delta is a STOP;
  (c) which agent relocates when two share a repo is a **human stop** — the
  agent cannot adjudicate ownership of a checkout it doesn't own.

### F2 — Fidelity-review provenance hashes are undocumented · tooling gap

`intent review ingest` hard-requires `policy.rubric_hash` + `prompt_hash`, but
nothing states what the host must hash. The agent reverse-engineered the
convention from itd-80's ingested provenance line plus git history (rubric =
reviewer agent file, prompt = emitted request file). A wrong-but-non-empty
hash would ingest fine — the requirement is currently un-checkable theatre.
- **For `/abcd:run`:** the emit path (`spec close` / `intent review`) should
  print or embed the expected hashes in the `.request.md` it writes; ingest
  could then verify instead of merely requiring non-empty.

### F3 — "delivered" input to the fidelity reviewer needs judgment · plan gap

The request file says "delivered: the diff/commit range that realised spc-4
(host supplies the range)". For record-catch-up items the implementation
merged long ago across many PRs, so "the range" is not mechanical; the agent
substituted "repo state at HEAD plus the load-bearing files". Worked, but a
PLAN driving record catch-up should state per item what "delivered" means.

### F4 — Named reviewers are prompt files, not harness agent types · protocol gap (resolved pattern)

`ruthless-reviewer`/`security-reviewer`/`intent-fidelity-reviewer` exist as
`agents/*.md` role definitions, not as dispatchable agent types in this
harness. The working pattern: spawn a general-purpose subagent whose first
instruction is to read and adopt the role file, and to return the terse
structured verdict only. The protocol should name this fallback explicitly so
"reviewer unavailable" is never silently equated with "skip the lens".
Corollary that worked well: the correctness reviewer independently chose a
detached worktree to escape the foreign uncommitted edits (F1) — reviewers
should always review the *committed* range, never the working tree.

### F5 — Run-level journal entries have no schema slot · protocol gap

`run-journal.json` is keyed on item stable-ids; the burst-1 STOP was run-level
(shared checkout), not item-level. The agent minted a `"RUN"` pseudo-id. Fine,
but the protocol should bless a reserved key for run-level outcomes so strike
counting never confuses an environmental STOP with an item failure.

### F6 — Worktree resume loses the handoff state · protocol gap

`.abcd/.work.local/` is per-worktree by design, so resuming in a fresh
worktree (the F1 remedy) starts blind: NEXT.md and run-journal.json stay in
the abandoned checkout. The restart prompt had to include a manual bootstrap
copy. `/abcd:run` needs a state-location rule for worktree moves (copy-once on
worktree creation, or a run-state directory addressed by repo, not checkout).

### F7 — Redundant review-emit step in the protocol flow · protocol gap (minor)

The plan said "close, then run `abcd intent review itd-89`" — but `spec close`
already emits the OWED receipt and request file. The extra `intent review`
call is harmless (idempotent re-emit) but the canonical flow should be stated
as: close → (auto-emit) → delegate reviewer → ingest.

### F8 — Interval semantics: 90m is not cron-expressible · tooling note (harness)

The harness loop rounded `/loop 90m` to 2h. Immaterial here (bursts are
self-bounded at 45m), but `/abcd:run` documentation should not promise
arbitrary intervals if the scheduling substrate quantises them.

### What ran fully autonomously in burst 1 (the positive finding)

State recovery → item pick → TDD gap-fill tests → gate → spec authoring →
lifecycle close → host-delegated fidelity review → deterministic ingest →
path-scoped commits → two PROMOTE reviews → push → PR, with zero human input.
The two places autonomy correctly ended: the shared-checkout collision (F1)
and the standing do-not-merge policy on PRs. Both are the right shape for
permanent human stops; nothing else in M1 needed one.
## Burst 2 (2026-07-17, manual test B — delegated implementation, M2 itd-40)

**Setup:** orchestrator (Fable 5) keeps state recovery, item picking,
journaling, gates, record commits, and PRs; implementation is delegated — a
deliberate, recorded deviation from the protocol's "never delegate
implementation" — to one Opus 4.8 max-effort worker per item in an isolated
worktree, with the orchestrator re-running `make preflight` on the output.
Reviews stay fresh-context Opus subagents. Structured via the harness
Workflow tool (one implementation workflow, one review workflow per item).

### F9 — Run-instruction premises go stale against a moving repo · protocol gap

The burst-2 instructions asserted two plan files were "currently untracked"
and must be committed on the first item's branch. By burst start they were
already committed — inside the concurrent hunter's *unpushed foreign commit*
on the old item branch, which the orchestrator must not touch. The letter of
the instruction was unsatisfiable; its intent (durable record on
`origin/main`) was satisfied by committing copies on the item branch and
flagging the future both-added merge. **For `/abcd:run`:** treat every factual
premise in run instructions (file X is untracked, branch Y is merged) as a
hypothesis to re-verify at burst start; on divergence, act on the recorded
*intent* and journal the divergence rather than STOPping or blindly obeying.

### F10 — Unpushed foreign commits are a coordination blind spot · protocol gap

Burst 1's F1 covered foreign *uncommitted* edits in a shared checkout. Burst 2
found the successor hazard: the hunter's finished work sits as unpushed
commits on a local branch, touching `internal/core/ahoy` (M2's package) and
`internal/core/capture` (M3's). Origin-based collision checks ("an open PR or
fresh commit by the other agent") never see these; branch-scoped discipline
protects the branch but not the eventual merge. Remedy used here: extract the
foreign commits' changed *functions* (`git diff` hunk headers) and hand the
worker an explicit no-overlap list, downgrading the plan's package-level
collision fear to a function-level guard — plus a PR-body note where the same
file is added on both sides. **For `/abcd:run`:** at burst start, enumerate
local branches ahead of origin, diff them, and feed the touched-function set
into every worker's guard; a *required* edit inside a foreign-touched function
is the skip trigger, not mere package co-location.

### F11 — The plan names a `spec create` verb that does not exist · plan gap + tooling gap

M2's instructions offered "`abcd intent plan itd-40` path or `spec create` +
`intent link`". Neither literal path works for an intent that is already in
`planned/` with `spec_id: null` (the record-catch-up shape this whole run
exists for): `intent plan` refuses non-drafts, and there is no `spec create`
CLI verb — `spec.Create` is core-internal, reachable only through `intent
plan`. Resolution: transient `git mv` planned→drafts, then `intent plan`
(mints spc-5 under the mint lock, relinks, moves back); net churn is the
`spec_id` write, record lint green throughout. **For `/abcd:run`:** plans must
name only verbs that exist (a dry-run of every named verb at plan-authoring
time would have caught this); tooling-wise, either `intent plan` should accept
a planned+unlinked intent, or `spec create` should exist for record catch-up.

### F12 — The protocol's changelog-fragment rule contradicts the repo · protocol gap (minor)

The protocol mandates `changelog.d/<slug>.md` fragments; this repo has no
`changelog.d/` and appends to `CHANGELOG.md` directly (AGENTS.md's rule). The
delegated worker noticed and followed the repo, which is correct — AGENTS.md
overrides — but the protocol text should say "the repo's changelog mechanism,
as the PLAN records it" instead of hard-coding one mechanism.

### F13 — Delegated commits need an identity check · protocol gap

The Opus worker's commit arrived authored "Alex Reppel <...>" although the
repo's `git config user.name` is `REPPL` — the worker (or its sub-shell)
resolved identity from somewhere other than the repo config, and nothing in
the flow would have caught it before push. The orchestrator caught it by
inspection and amended the unpushed commit's author. **For `/abcd:run`:** when
implementation is delegated, the orchestrator's post-worker gate must include
an identity check on every new commit (the repo even ships one — `abcd ahoy
identity-check`); author metadata is part of the record, not a cosmetic.

### Delegation observations (test B vs burst 1's write-it-yourself baseline; running log)

- **TDD evidence survives delegation.** The worker returned verbatim
  watched-fail excerpts for both genuine gaps, and — unprompted — split
  "gap-filled (watched-fail)" from "met-already (characterization test
  added)", exactly the honesty the protocol wants. Quality of evidence is
  indistinguishable from burst 1's first-person TDD.
- **The worker respected every guard:** function-overlap list (edit landed in
  `newAhoyCommand`'s render closure, outside all guarded bodies), iss-101
  untouched, hermetic tests (temp HOME), no new deps, no record files touched,
  single atomic commit with a why-shaped body.
- **Orchestrator re-verification caught real deltas:** the author-identity
  drift (F13) — nothing the worker reported; found only by inspecting the
  commit. Re-running the gate myself reproduced green (test cache warm from
  the worker's own run, so the re-run cost seconds, not minutes).
- **Cost:** implementation phase = 1 Opus/max worker, ~95k subagent tokens,
  ~9.5 min wall, 47 tool uses, for a 200-line diff (8 production lines, 188
  test lines) — heavier per line than burst 1's in-context work, but the
  orchestrator's context stayed small enough to run the whole record/lifecycle
  half without compaction risk.
- **Review outcomes under delegation:** three parallel Opus/max reviewers
  (correctness, security, fidelity) returned PROMOTE / PROMOTE / 5-of-5 MET
  with zero findings in ~2.6 min and ~134k subagent tokens. The correctness
  reviewer independently ran the mental-revert check (would the watched-fail
  tests fail without the 8 production lines?) and traced an unreachable
  duplicate of the hint string before promoting — review depth did not degrade
  because the code under review was itself agent-written. Burst-2 M2 totals:
  ~229k subagent tokens + orchestrator overhead, one gate re-run, zero HOLDs,
  zero human input.
### F14 — The local gate cannot see environment-dependent test failures · protocol gap

Mid-burst-3 maintainer detour: the bug-hunt branch (PR #85) failed CI on
ubuntu only — its new retention test created a git commit with `--author` but
no `-c user.name/user.email`, so it broke on any machine without a global git
identity. Every local gate run (the hunt's own, and this run's re-runs) passed
because the dev machine has one; macOS runners happen to carry a git identity
while ubuntu runners don't, making the failure platform-asymmetric and
invisible until push. `make preflight` is the run's sole admission authority,
but it inherits the workstation's environment. **For `/abcd:run`:** (a) worker
instructions must carry the repo's known env-hermeticity rules (this run's M3
worker prompt now includes the git-identity rule); (b) the protocol should
treat first-push CI as a second, asynchronous gate — a PR is not "done" until
its CI is green, and babysitting that belongs to the orchestrator's burst,
not the maintainer. Tooling-wise, a repo test-lint (grep for `git commit`
without `-c user.`) would catch the whole class deterministically.

### F15 — "Implement it at the skill surface" is not a universal fallback · plan gap

M3's pre-adjudication said: verify the plugin-surface promote path is wired,
"else that AC is a genuine gap to implement at the skill surface". The worker
verified the stronger fact: the skill surface *cannot* implement AC3, because
half the flow has no engine verb behind it (no intent-create until itd-46; no
back-link write verb at all), and markdown instructing hosts to hand-edit
frontmatter would violate the engine-backed convention (iss-86). The honest
verdict was BLOCKED, spec left open — the adjudication's fallback assumed
skill-surface sufficiency without checking verb coverage. **For `/abcd:run`:**
a plan adjudication that prescribes a fallback implementation path must name
the engine verbs that path depends on; "the skill layer will handle it" is
only true when every step is engine-backed.

### F16 — Record catch-up surfaces AC-letter vs live-design drift · protocol gap (minor)

itd-4's AC2 says the resolve note is "appended to the body"; the shipped,
in-daily-use design stores it as a queryable frontmatter scalar. For record
catch-up items this class — the AC letter written before implementation,
the implementation deliberately better — recurs (burst 1's SessionEnd-vs-Stop
was the same shape). The run handled it by adjudicating in the spec body +
[dcn-2607170000000002](../../work/decisions/dcn-2607170000000002-burst-3-m3-itd-4-spc-6-three-record.md). The protocol should name this verdict explicitly (met-with-
recorded-deviation) so catch-up runs neither force code back to the stale
letter nor silently mark MET.

### Delegation observations, burst 3 (running log)

- **The worker's honesty held under a blocked AC:** told to implement the
  promote surface "if genuinely missing", it instead proved the surface
  *couldn't* be implemented (verb-coverage analysis, engine-backed-convention
  citation) and returned BLOCKED with the precise missing links — the
  highest-value outcome available; forcing markdown would have shipped a lie.
- **Characterization-only milestone:** no production code changed; one test
  commit (60 lines), no CHANGELOG entry (correctly judged test-only), no
  watched-fail (correctly judged: no behaviour changed). Evidence discipline
  intact. ~78k subagent tokens, ~8.2 min.
- **Mid-burst maintainer detour handled inline:** PR #85's conflict + Linux
  CI failure (F14) were resolved by the orchestrator between the worker's
  launch and return — multi-agent structure kept the item flowing while the
  orchestrator context-switched.

### Delegation observations, burst 4 (running log)

- **Real-implementation delegation held the TDD bar:** itd-46 was the run's
  first genuinely new-behaviour item, and the worker produced verbatim
  watched-fail evidence for every gap-filled AC (build-failure red for the
  engine, `unknown command` red for both CLI routes, missing-rule red for the
  help line), plus a bare-invocation regression pin. Three atomic commits
  (engine / wiring / surface+docs) with why-shaped bodies.
- **Premise verification generalized:** told the intent's scope verbatim, the
  worker discovered two scope bullets referencing files that do not exist in
  the Go tree (old-system paths) and flagged them instead of creating them —
  the F9 lesson (stale premises) applied by a sub-agent unprompted. Both
  adjudicated in spc-7; the underlying surface gap ledgered (iss-105).
- **Design-delta honesty:** the worker flagged its own typo-guard asymmetry
  vs capture (mistyped sub-verb files a draft) as an explicit deliverable
  note rather than hiding it; ledgered (iss-104).
- **Cost:** ~140k subagent tokens, ~13 min, 3 commits, 6 files.

### F17 — `abcd audit` green is not `record-lint` green · tooling gap

The itd-46 lifecycle move (planned→shipped at spec close) broke two inbound
record links, and the orchestrator's mid-record checks missed it for a full
commit: `abcd audit` (run after every record step, green throughout) does not
resolve links, while preflight's `record-lint` (run only at gate time) does —
the red surfaced two commits after the breakage. Same-family gates with
different coverage invite exactly this. Fixed by pointing both links at the
shipped path; the deeper items: (a) `spec close` moves a file it KNOWS has
inbound links — it could rewrite or at least report them; (b) the orchestrator
rule is now "full `record-lint` after every record commit, not just `audit`".

### Delegation observations, burst 5 (running log)

- **Detector-first survives delegation fully:** the worker armed GL002 before
  touching prose, captured its 19 real corpus findings verbatim as the
  watched-fail record, swept exactly what it flagged, and pinned the corpus
  at zero with a real-glossary regression test — never hand-fixing ahead of
  the armed detector. The repo's fix-the-detector principle transferred into
  a sub-agent's discipline intact.
- **Memory-carried hazards transfer by prompt:** the orchestrator's standing
  note about Go's ASCII-only regexp \b went into the worker's instructions;
  the worker implemented explicit Unicode boundaries and wrote the boundary
  proof test (epics/epicenter/unicode-adjacent do not match). A known-hazard
  line in a worker prompt is cheap insurance.
- **Judgment calls surfaced, not buried:** the worker flagged its self-quote
  sweep decision (itd-48's internal quotes) as an explicit judgment call in
  its report; the orchestrator promoted it to the decision [dcn-2607170000000004](../../work/decisions/dcn-2607170000000004-burst-5-m5-itd-43-spc-8-gl002-enforces-a.md) and the spec body.
- **Heaviest worker of the run:** ~169k subagent tokens, ~23 min, 95 tool
  uses, 4 commits — the detector-build + corpus-sweep combination is the
  upper end of what one worker context comfortably holds.

### F18 — The changelog-driven release path verified live · closing finding

The run's final act was its own acceptance test: PR #89 (adr-37 + the
auto-release port + the `[0.2.0]` roll) merged, and the automation fired
first time — detect extracted 0.2.0 from the rolled heading, the annotated
`v0.2.0` tag landed on exactly the merge commit (36s after merge), the
called `release.yml` verified that commit, cross-compiled the four binaries,
generated checksums, and published the GitHub Release with all five assets;
the attestation steps skipped cleanly (private repo) and the
no-branch-push tripwire held. Zero human tag pushes, zero reruns. The
policy's whole claim — the CHANGELOG roll IS the release decision and
automation follows the record — held on its first firing.

### Run test B — closing summary

Five milestones (M2–M5 + release) shipped across six PRs (#84–#89, plus the
maintainer-directed bug-hunt landing #85), all merged same-day. Delegation
verdict: implementation quality survived one-worker-per-item delegation
intact (watched-fail TDD evidence, detector-first discipline, honest BLOCKED
verdicts), while the orchestrator's independent re-verification earned its
keep repeatedly (author-identity drift F13, the record off-by-one, the
broken-link gate red F17, the audit-vs-record-lint asymmetry). Cost: ~1.3M
subagent tokens across 17 Opus/max agents. The findings F1–F18 and the
per-burst delegation logs are the raw material for reconciling the protocol
into `commands/abcd/run.md` (itd-29).
