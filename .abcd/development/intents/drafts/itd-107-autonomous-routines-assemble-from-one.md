---
id: itd-107
slug: autonomous-routines-assemble-from-one
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# Autonomous Routines Assemble From One Versioned Template

## Press Release

> **abcd now assembles the prompts that drive autonomous maintenance loops.** A
> repository's recurring bug-hunt or plan-drain routine is no longer a
> hand-written wall of text: `abcd routine render` detects the repo's toolchain,
> gate commands, and working-tier layout, fills a versioned archetype template,
> and emits the finished routine prompt plus a host-ready schedule manifest. The
> safety invariants every loop needs — state comments trusted only from the
> repo's own maintainers, resumption of a pushed branch that never got its pull
> request, a bounded wait on CI, one round per scheduled run — are baked into
> the template, not remembered per repo. When the template improves, the whole
> fleet re-renders in one pass.
>
> "I had six repos hunting bugs on a schedule, and every prompt was a slightly
> different vintage," said Alice, a maintainer. "One of them was told to commit
> into a directory the repo ignores. Now they all render from the same template,
> and when I fix the template I have fixed the fleet."

## Why This Matters

A fleet audit of eight live routines found three prompt generations with
divergent safety properties: different stop sentinels, merge gates with and
without a bounded CI wait, single- and multi-round pacing, and one prompt
instructing every round to commit a decision log into a path its repository
gitignores. None of these differences was a decision — each was drift from
hand-assembly. Re-hardening the fleet meant editing every routine individually,
a full session of expert attention for changes that were mechanically identical.

A live bug-hunt loop on this repository's own state issue (#197) independently
found three more invariants a hand-assembled prompt is prone to miss, none
caught by the original fleet audit because none had happened yet: an
orchestration-side cleanup racing a still-running reviewer subagent's own git
operations in a shared checkout (round 8, recovered only by chance from a
dangling stash); a stop-streak counter that could not tell three rounds of
convergent diagnosis — each one narrowing the same root cause — from three
rounds that learned nothing, and declared the loop wedged when it was actually
converging on a correct fix; and a then-blocked bug (iss-186/PR #203, since
resolved) that kept
resurfacing at STATE step 3's automatic PR-resumption every round until a human
happened to intervene by hand, with no way for the loop to set it aside and work
on something else in the meantime. A hand-assembled prompt fixes the instance in
front of it; only a shared template propagates the fix to every routine that
hasn't hit that failure yet.

The assembly is deterministic work sitting outside the binary. Detecting a
toolchain from its manifest, finding where the tracked decision log lives,
choosing gate commands, slotting a hunt focus into a template — all of it is
exactly the kind of behaviour the transport-agnostic core exists to own, and
today it lives in an unversioned, untestable prompt-assembly skill on one
person's machine. Under **wired or it isn't done**, the discipline exists but
nothing in abcd carries it.

The split follows the house boundary: the binary owns detection, the template,
and rendering; the plugin page owns the interview (focus, cadence, merge
authorisation) and hands the rendered artefact to whatever the host uses to
schedule it. The binary never contacts a scheduling service.

## What's In Scope

- A versioned routine template with archetypes — bug hunt (security /
  correctness / docs / all focuses) and plan drain — carrying the safety
  invariants: maintainer-only trust for loop-state comments, pushed-branch-
  without-PR resumption, bounded CI wait, one-round-per-run default with an
  explicit capped multi-round variant, refuter count, stop conditions, and
  public-artifact hygiene (iss-178): session URLs and harness attribution
  footers are banned from all public text, and because the harness can append
  them outside the model's own words, every created pull request, issue, and
  comment is re-read as stored and stripped by edit before the round continues.
- Working-tree isolation: every subagent that inspects, builds, tests, or
  diffs repository state — each hunt angle, each verify candidate, the fix
  itself, every pre-PR and merge-gate review — runs in its own dedicated git
  worktree, never the orchestrator's own checkout and never another
  subagent's. The orchestrator never mutates or discards state a subagent may
  still be reading; when several reviews of one diff are dispatched together,
  it waits for all of them to return before acting on any single verdict.
- Root-cause escalation in place of blunt strike-counting: a stop-eligible
  BLOCK that traces to the same underlying coupling as an earlier BLOCK — not
  merely the same bug, the same mechanism — is diagnostic progress, not
  failure, and does not by itself count toward the loop's stop streak. It
  instead queues a mandatory structural-fix attempt (removing the coupling,
  not special-casing around it, through the same unrelaxed review gates) for
  the next round; only a BLOCK on that structural attempt is a genuine stop.
- Ledger shelving: a bug (or its already-open, already-blocked PR) that a stop
  or an exhausted root-cause escalation rules out of autonomous reach is
  marked needs-human in the loop-state ledger once, not rediscovered every
  round — excluded from future picking and from automatic PR-resumption until
  a trusted comment or ledger edit clears it, so the round that follows works
  on the next fixable thing instead of re-litigating the same block.
- `abcd routine render`: repo detection (toolchain and gate commands from the
  manifest; tracked-versus-ignored working tiers, so a rendered prompt can
  never instruct a commit into an ignored path; platform caveats for
  cross-compiled targets) plus deterministic output — same repo, same template
  version, byte-identical render.
- A neutral schedule manifest alongside the prompt (name, cadence, repo,
  model routing), for the host front door to translate.
- A plugin page front door that interviews, renders through the binary, and
  hands off to the host's scheduler.
- An optional **release on completion** template switch, default off, offered
  explicitly by the interview: when the loop reaches its drained stop (the
  configured run of nitpick-only rounds) and substantive fixes have merged
  since the last release, the closing round initiates the repository's release
  flow instead of only posting the stop sentinel. The routine never bypasses a
  release gate — in an abcd-managed repo that means the changelog-driven
  release gate runs exactly as it would for a human-initiated cut.
- **Merge-lifecycle invariants for the loop's own pull requests.** Auto-merge
  eligibility splits on *additive versus editing*, never on "documentation
  versus code": a change qualifies only when every path it touches is a new
  record file or an append to the decision log, the diff carries no deletions,
  the record gate is green, and it is a single commit. Anything that rewrites
  an existing claim — published documentation, the brief, the changelog —
  stays behind the run's explicit merge authorisation, because that is where a
  repository's post-merge corrections cluster: they fix false claims about the
  system, which no deterministic gate catches. The eligible set is an
  allowlist of inert paths, never a denylist of source extensions — in an
  abcd-managed repo the plugin command pages, the rules file, the lint
  configuration and the agent-instruction router are all markdown or JSON that
  change behaviour, so a "no source touched" test fails open on every one of
  them. Intent drafts count as additive: the lifecycle already gates at
  promotion, so a second gate at merge is redundant.
- **The up-to-date requirement is load-bearing and is never relaxed.** A
  queued auto-merge whose base moves waits indefinitely, and the tempting fix
  — dropping the strict status-check policy — is the one change the template
  forbids: strict is what re-runs the record gate against the merged result,
  and that is the only place a record id minted concurrently on two branches
  becomes visible, since each branch passes in isolation and the mint lock is
  a filesystem lock that does not span checkouts. The loop clears the stall by
  updating the branch, which preserves the gate.
- **The loop's platform calls live in the rendered prompt, never in the
  binary.** Arming auto-merge, polling merge state, updating a stale branch,
  and re-reading a created artefact are host-executed steps carrying the
  host's own credentials, so the binary acquires no platform CLI dependency.
  A repository-scoped credential that a scaffolded workflow needs is consumed
  by that workflow, never held or read by abcd.

## Acceptance Criteria

- Given a Go repository with a `Makefile` preflight target, When Alice runs
  `abcd routine render` with the all-dimensions hunt archetype, Then the
  emitted prompt contains that repo's actual gate commands and every template
  safety invariant, and rendering twice produces byte-identical output.
- Given a repository whose candidate decision-log path is gitignored, When a
  routine is rendered for it, Then the prompt directs round records to the
  loop-state issue only and contains no instruction to commit into an ignored
  path.
- Given a fleet rendered at template version N, When Bob re-renders it at
  version N+1, Then each routine's prompt differs from its predecessor only by
  the template change.
- Given the plugin page interview, When Carol completes it, Then the rendered
  prompt and schedule manifest are produced by the binary alone, with no
  network call made by the binary.
- Given the interview's defaults accepted, When a routine is rendered, Then it
  contains no release step; Given release on completion switched on, When the
  loop's final stop follows rounds that merged substantive fixes, Then the
  rendered prompt directs the closing round to initiate the repository's
  release flow, subject to every release gate the repo declares.
- Given any archetype, When a routine is rendered, Then the prompt bans
  session URLs and harness attribution footers from every public artifact and
  directs a post-create re-read-and-strip of each pull request, issue, and
  comment the loop creates.
- Given a rendered bug-hunt archetype, When any round dispatches more than one
  subagent against the same ref (a multi-angle hunt, a paired pre-PR review, a
  merge-gate review pair), Then the prompt directs each to its own isolated
  git worktree and directs the orchestrator to wait for every dispatched
  subagent to return before mutating or discarding the state they inspected.
- Given a rendered bug-hunt archetype, When a second BLOCK traces to the same
  underlying coupling as an earlier BLOCK in the same round or a prior one,
  Then the prompt directs the next round to attempt a structural fix removing
  that coupling — through the same review gates, not a relaxed one — before
  counting the pair toward the loop's stop streak.
- Given a rendered bug-hunt archetype, When a bug or its open PR is ruled out
  of autonomous reach (a stop, or an exhausted root-cause escalation), Then
  the prompt directs it marked needs-human in the loop-state ledger and
  excluded from future picking and PR-resumption until a human clears it.

## Open Questions

- Verb name and family: `abcd routine` as its own verb, or a sub-verb of an
  existing family?
- Does the binary emit only the neutral manifest, or also host-specific
  schedule configuration behind an adapter seam?
- Where fleet membership is recorded (which repos carry which routines at
  which template version) — per-repo record, user home, or nowhere?
- Whether the plan-drain archetype's work-queue contract (plan file wins,
  strike rules) is part of this intent or its own.
- What "initiate the release flow" means per repo — invoke the release
  machinery directly, or open the release cut as a final gated pull request —
  and how the closing round interacts with receipt-gated releases that need
  host-run semantic reviews.
- Whether the merge-lifecycle protocol ships only as rendered prompt steps, or
  also as a scaffolded workflow — and if the latter, whether that workflow
  takes a repository-scoped credential or points at a platform merge queue
  where the plan allows one.
- Whether a repository's merge style — collapsing a single-commit record
  change, keeping the history of a multi-commit branch — is declared by the
  template or detected per repo.
- Whether two peer sessions sharing one checkout are prevented by policy or by
  giving every session its own worktree, given that the orchestrator/subagent
  isolation rule above does not cover independent peers.
- Whether attribution-footer defence is detection, post-create stripping, or
  both, and whether a detection blocks the round or only annotates it.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
