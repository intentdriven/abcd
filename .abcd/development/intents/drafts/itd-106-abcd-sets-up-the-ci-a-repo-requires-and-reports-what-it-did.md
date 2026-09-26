---
id: itd-106
slug: abcd-sets-up-the-ci-a-repo-requires-and-reports-what-it-did
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-93]
severity: minor
impact: additive
---

# abcd Sets Up the CI a Repo Requires, and Reports What It Did

## Press Release

> **abcd now sets up the continuous-integration gates a repository requires —
> and tells you exactly what it did and what it could not do.** Point it at a
> repo and it detects what a product thinker should never have to care about:
> the stack from its manifest, whether the repo is public or private, which
> workflows and branch rules already exist. It then scaffolds the house gate
> shape for that stack — one check job running the repo's own build, type,
> lint, and test gates, plus full-history secret scanning and a workflow
> self-audit — switches on auto-deletion of merged branches, and, where the
> platform allows, writes the branch ruleset that makes those checks required.
> Every outward write is confirmed first and lands as a pull request. The
> closing report names each gate included, each gate excluded and why, and
> each gap it could not close.
>
> "I think in products, not in workflow YAML," said Alice, a product thinker.
> "abcd set up the checks, told me my private repo cannot carry a branch
> ruleset on my plan, and told me my format gate would fail on 681 files if it
> were switched on. That last sentence is the one I could never have written
> myself."

## Why This Matters

Bringing eight repositories up to one safety bar — CI where none existed,
checks-gated branch rulesets, auto-deletion of merged branches — recently took
a full session of expert attention. Every step was deterministic: read the
manifest, pick the matching template, verify the gates actually pass before
requiring them, match required contexts to the exact job names, report the
private-plan repos where no ruleset is possible. Deterministic expert work
that repeats per repo is precisely the facilitation abcd exists to absorb;
leaving it manual means most repos simply never get the bar.

The seam already exists. [itd-93](../shipped/itd-93-abcd-scaffolds-a-hardened-changelog-driven-release-gate-into.md)
has abcd scaffolding the hardened release workflows into a managed repo,
parity-tested against the live workflows so the template cannot drift from
reality. This intent generalises that machinery to the standing CI gates and
repository settings rather than adding a second scaffolding path. The
repo-preparation front door then stops instructing a session to hand-write CI
and starts invoking a verb that does it — with the interview asking only the
questions a maintainer must answer.

Autonomous maintenance loops sharpen the need: their merge gates instruct
"wait for CI, merge only when green", which is vacuous in a repo with no CI
and unenforced in a repo with no required checks. The server-side backstop
should be something abcd establishes, not something a human remembers.

## What's In Scope

- Stack and state detection: manifest-based toolchain identification,
  repository visibility, existing workflows, existing branch rules, the
  merged-branch auto-deletion setting.
- Per-stack CI templates versioned in the binary and parity-tested in the
  itd-93 manner, all third-party actions and images pinned.
- Pre-flight verification: each candidate gate runs locally first; a gate that
  fails is excluded and reported, never silently required.
- Repository-settings reconciliation: auto-deletion of merged branches; a
  checks-gated branch ruleset whose required contexts exactly match the
  scaffolded job names, where the platform plan allows one.
- The report: every action taken, every exclusion with its reason, every gap
  that could not be closed and its consequence.
- Confirm-first on every outward write; workflow changes land as a pull
  request, never a direct push. Idempotent re-runs: a conformant repo yields
  zero writes and a drift report.

Out of scope: fixing a repo's failing gates (that is the repo's own work),
and changing platform plans or repository visibility.

## Acceptance Criteria

- Given a public repository with a lockfile and build, type-check, lint, and
  test scripts but no workflows, When Bob runs the facilitation verb and
  confirms, Then a CI workflow with one check job plus secret-scan and
  workflow-audit jobs is opened as a pull request, and the report names every
  gate included and every gate excluded with its reason.
- Given a repository whose format gate fails across the tree, When the verb
  runs, Then that gate is excluded from the scaffolded workflow and the
  exclusion is reported with the failing count — it is never required red.
- Given a private repository on a plan without branch rulesets, When the verb
  runs, Then no ruleset write is attempted and the report states the gap and
  its consequence plainly.
- Given a repository already conformant, When Carol re-runs the verb, Then it
  performs zero writes and reports conformance.
- Given scaffolded jobs, When the ruleset write is confirmed, Then the
  required status-check contexts equal the emitted job names exactly.

## Open Questions

- Verb home: the evolution of the repo-preparation front door, an extension of
  the itd-93 scaffold family, or a new verb of its own?
- How the split between audit (warn about missing gates) and facilitation
  (create them) is drawn — does the audit verb learn these checks first?
- Which repository settings beyond auto-deletion and the ruleset belong to the
  reconciliation set?
- Whether the interview also offers the platform-specific extras a stack
  implies (e.g. an end-to-end suite as a non-required job).

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
