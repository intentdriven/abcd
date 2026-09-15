---
id: itd-131
slug: managed-repo-identity-gate
spec_id: spc-34
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
related_intents: [itd-91]
severity: major
promoted_from: iss-62
---

# The managed-repo identity gate: the human is the author of record on every commit, before the first one

## Press Release

> **abcd makes sure the right person is the author of record — before a
> single commit lands, not after 54 of them.** A managed repo can silently
> commit under the wrong git identity: a repo-local `Test User
> <test@example.com>` left over from a sandbox, a committer that differs from
> the author, or — for an autonomous routine — the harness's own default
> `Claude <noreply@anthropic.com>`. Each one puts an identity in the
> contributor graph that the human never chose, and abcd's own attribution
> gate rejects it downstream, where it surfaces as an unmergeable-looking PR
> long after the commits are written. With this gate, `ahoy` checks the
> *effective* commit identity — author **and committer**, whatever git config
> or environment produced it — against the identity the repo pins, and when
> they diverge it does not silently rewrite anything: it **proposes** the
> pinned (or global) identity and **asks** before writing repo-local config.
> For an autonomous routine — which has no human to ask and defaults to a tool
> identity — the gate detects the wrong identity and the routine's *runner*
> establishes the human one before the first commit, so the routine's work is
> the human's from the start.

> "A sandbox `Test User` override authored 54 commits before anyone noticed —
> we had to rewrite history and force-push to unpick it," said Alice, who
> maintains the repo. "Now `ahoy` catches a wrong identity before the first
> commit, and it asks me rather than guessing." Bob, who runs abcd's
> autonomous bug-hunt, hit the mirror image: "The routine committed as the
> tool, so every PR it opened failed the attribution check and read as
> unmergeable — but it wasn't a conflict, it was the author. The gate flags
> that tool identity, and the runner sets mine before the first commit, so the
> work is attributed to me."

## Why This Matters

Graduated from [[iss-62]]. abcd already has *most* of the check and none of
the establish. `ahoy`'s `detectGitIdentity` calls `identity.EffectiveIdentity`,
which already resolves the **author** identity from the environment
(`GIT_AUTHOR_NAME`/`GIT_AUTHOR_EMAIL`) first and falls back to git config — so
the sandbox/CI env-override case is caught *for the author* today, and this
intent must not re-claim it. What is actually missing is narrower and real:

1. **The committer is never checked.** `identity.Check` evaluates only the
   author; a committer that differs from the author, or a `GIT_COMMITTER_*`
   override, passes the disk-side gate. CI's `check-attribution` *does* check
   the committer, so the true gap is "the local gate checks author, not
   committer; CI checks both" — a divergence that lets a wrong committer reach
   CI before it is caught.
2. **The gate reports; it does not establish.** Every gap's fix hint tells the
   human to set `user.name`/`user.email` by hand. Nothing offers to do it —
   safely, with confirmation — at the moment the repo is prepared, so the
   wrong identity is a diagnostic the human must act on, not a state the gate
   corrects.
3. **The autonomous-routine case has no local establish path.** A cloud
   routine runs with the harness's default `Claude <noreply@anthropic.com>`
   identity and no human to confirm a write; the only guard today is a line
   in the routine's prompt (fragile — an agent can forget it), and the failure
   is caught only in CI, where it presents as an unmergeable PR (see the
   sibling captures on the CI-only attribution gate).

This is the **identity** half of a pair. [[itd-91]] owns the *acknowledgement*
half — how AI is credited (the `Assisted-by:` trailer style, footers, declared
once per repo). Identity answers "who is the git author/committer of record"
(always the human); attribution answers "how is the AI's help disclosed" (a
trailer, never the identity fields). Concretely, this splits the existing
`check-attribution` gate: its **`check_ident`** pass (the AI-name/vendor ban in
author/committer fields) is itd-131's concern; its **`check_text`** pass (the
trailer/footer convention) is itd-91's. The convention that the human is the
author of record already exists in `AGENTS.md` § Attribution; this intent
makes it *true by construction* when work starts, rather than *checked after
the fact* in CI.

## Decisions (grilled 2026-08-21)

The maintainer resolved these at the planning interview, after two adversarial
reviews reshaped the draft; they are commitments, not options.

- **Detect here, establish at launch.** This intent detects the wrong or
  tool identity (author≠committer, `GIT_COMMITTER_*`, an AI/tool default) and
  does propose-and-confirm at a TTY; *establishing* the human identity for a
  non-interactive routine is the runner's job — [[iss-2608210932052003]] for
  routines abcd launches. itd-131 defines the guarantee and owns detection; it
  does not implement the launcher. (Resolves the review's blocker.)
- **The proposal chain is disk-only: pinned → global.** No `gh`-authenticated
  fallback — a network lookup in `ahoy`'s implicit path violates [[adr-38]].
  If ever wanted, it is a separate explicit verb, not this gate.
- **The five acceptance criteria are accepted as revised.**

## What's In Scope

- `ahoy` (and `doctor`) extend the identity check to the **committer** —
  detecting author≠committer and `GIT_COMMITTER_*` overrides — reusing the
  env-first-then-config resolution `EffectiveIdentity` already applies to the
  author. (The author case is already covered and is not re-built here.)
- On divergence, **propose-and-confirm** at a TTY: offer the pinned identity
  (else the global git identity) and ask before writing repo-local
  `user.name`/`user.email`. Never silently rewrite git config, never escalate
  privileges (the `ahoy` install posture). The proposal chain is disk-only
  (pinned → global) — no network lookup, per [[adr-38]].
- **Non-interactive / routine context (no TTY):** the gate is fail-closed — it
  reports the divergence and does **not** write config unprompted. Establishing
  the human identity for a routine is the *runner's* job (see below), not a
  silent write by this gate.
- The un-pinned repo: adopting the gate (writing `identity.json`) stays the
  ConfigChange-gated, never-under-`--yes` step it already is.
- **Detecting the routine's tool identity:** the gate recognises an AI/tool
  identity (e.g. the harness default) as a divergence to report, so the
  condition is machine-visible wherever `ahoy` runs. *Establishing* the human
  identity before the first commit is done by whatever launches the routine —
  for routines abcd itself launches, [[iss-2608210932052003]] applies it at
  launch; for externally-run routines, the documented env/config step until
  then.

## What's Out of Scope

- The *acknowledgement* convention (trailer style, footers, the declared
  per-repo preference) and its `check_text` enforcement — [[itd-91]] owns
  them; this intent cites, never duplicates.
- **Launching autonomous routines and setting their identity at launch** —
  [[iss-2608210932052003]] owns that mechanism; this intent detects the wrong
  identity and defines the guarantee, but does not implement the launcher.
- A `gh`-authenticated-account fallback for the proposal: a network lookup in
  an implicit path violates [[adr-38]] (disk-only). If wanted, it is a
  separate explicit verb, not part of this gate.
- The CI-only-attribution-gate and working-tree-vs-commit gate gaps captured
  this session — enforcement plumbing, filed separately.
- Rewriting history a wrong identity already produced — the gate exists so that
  repair is not needed; when it is, repo policy forbids force-push, so the
  reauthor-and-new-branch path applies (not this intent's concern).

## Scope Conditions

None stated.

## Acceptance Criteria

> _Confirmed by the maintainer at the 2026-08-21 planning interview: all five
> accepted as revised (after two adversarial reviews the same day)._

- **Given** a repo whose pinned identity is the human's and a repo-local
  `user.name`/`user.email` override that differs, **when** `ahoy` runs at a
  TTY, **then** it reports the divergence and proposes the pinned identity, and
  writes repo-local config only after the human confirms.
- **Given** a commit whose committer would differ from its author (author≠
  committer, or a `GIT_COMMITTER_*` override), **when** the gate checks,
  **then** the committer divergence is detected — the delta over today's
  author-only check.
- **Given** a non-TTY / routine context, **when** the gate finds a divergence,
  **then** it reports and writes nothing unprompted (fail-closed) — it never
  silently rewrites config, and never blocks the run on a prompt that cannot
  be answered.
- **Given** an autonomous routine running under an AI/tool identity, **when**
  `ahoy` checks, **then** it *detects and reports* that identity as a
  divergence; *establishing* the human identity before the first commit is
  verified against the runner ([[iss-2608210932052003]]), not this gate.
- **Given** any divergence, **when** the gate acts, **then** it never silently
  rewrites git config and never escalates privileges.

## Open Questions

- **How is the committer identity resolved without a throwaway commit, and
  without regressing `StatusUnset`?** Extend the env-first-then-config
  resolution `EffectiveIdentity` uses to the committer fields
  (`GIT_COMMITTER_*` then committer config). NOT via `git var
  GIT_AUTHOR_IDENT`/`GIT_COMMITTER_IDENT`: `git var` fabricates an identity
  from gecos+hostname when none is set and exits 0, collapsing the distinct
  `StatusUnset` state the pre-commit hook blocks on — a regression the review
  demonstrated. Confirm the exact resolution at the grill.
- **Does splitting `check-attribution` (`check_ident` → itd-131, `check_text`
  → itd-91) need any change to the script, or only a recorded ownership note?**
  itd-91's out-of-scope also eyes a commit/PR-time lint; the two must not both
  claim the same pass.

## Prerequisites

- **iss-62 is the source** (promoted to this intent); its `Test User` incident
  is the motivating case.
- **The routine refinement** captured this session
  (`ahoy-setup-routine-git-identity`) folds in here as the detect-the-routine-
  identity scope.
- **[[itd-91]]** is the acknowledgement-half sibling (`related_intents`); this
  intent lands consistent with itd-91's declared-preference model.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
