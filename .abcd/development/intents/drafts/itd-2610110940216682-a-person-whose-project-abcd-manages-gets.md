---
id: itd-2610110940216682
slug: a-person-whose-project-abcd-manages-gets
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Every change is checked in full once, on exactly what lands

## Press Release

> A person whose project abcd manages gets checks that are thorough and still fast. Each pull request gets the quick checks in minutes. The full suite, on every operating system the project supports, runs once per change, in the merge queue, on exactly what will land, so nothing reaches the default branch unchecked and nothing is checked three times. abcd sets this up when it prepares the repository: the workflow, a clean-up that stops runs the queue has abandoned, a queue sized to the runners the account actually has, a nightly full run in place of a run after every merge, and one test pass per platform. The person never tunes CI by hand, and a small project stops spending most of its CI time repeating itself.

## Why This Matters

abcd is built for people who know what they want and are not CI engineers, which argues for more checking, not less. abcd's own repository showed what more checking costs when it is laid out naively: in the six days to 2026-10-11 it ran its full CI 300 times for 71 merged changes, once on the pull request, once or more in the merge queue, and once again after every merge, each run holding one of the account's five macOS runners for about 40 minutes. On 2026-10-11 nothing merged for five hours while queue entries waited for those runners and timed out. A person whose project abcd manages would meet the same wall sooner, with no one to diagnose it. The same safety net costs a third as much when the full suite runs once, where it matters: on the merge result.

## Mechanism

We expect total CI time per merged change to fall by about half, with no change ever landing untested on any supported platform, because the merge queue already runs the full suite on the exact merge result: the pull-request run's full suite, the post-merge run and the second macOS test pass repeat work the queue run or the local preflight already did. Shown wrong if, after abcd's own CI takes this shape (iss-2610110938282654, iss-2610110938294826, iss-2610110938293273), runs per merged change do not fall, or platform defects reach the default branch that the old pull-request run would have caught. <!-- record-lint: forward-looking -->

## Scope Conditions

- Holds for a repository on GitHub whose default branch is gated by a merge queue; a repository without one keeps the full suite on its pull requests, since nothing else tests the merge result.
- Holds where some runner is scarce (macOS on GitHub's free and team plans: five concurrent jobs); with no scarce runner the queue sizing is moot and the saving is the duplicate runs alone.
- Holds where every push already passes a local preflight on the developer's platform, so the pull-request run's platform pass repeats local work.

## Acceptance Criteria

- **Given** a repository abcd prepares that gates its default branch through a merge queue, **when** its CI is set up, **then** a pull request runs the quick checks on one platform, every required check still reports a conclusion on the pull request, and the full suite runs on every supported platform in the merge queue on the merge result.
- **Given** the merge queue rebuilds its groups, **when** a superseded group's runs are still queued or running, **then** they are cancelled, and a run whose queue entry is still live is never cancelled.
- **Given** a merge to the default branch, **when** it lands, **then** no second full run starts for it, and a scheduled full run on the default branch checks it within a day and keeps the build cache warm.
- **Given** the account's concurrency for its scarcest runner, **when** the queue is configured, **then** it builds no more entries at once than that runner allows with room to spare, and its check timeout exceeds the full suite's measured duration.
- **Given** a platform whose suite runs a race-enabled pass, **when** the full suite runs there, **then** it runs one pass, not a plain pass and then a race pass.
- **Given** the checks abcd sets up, **when** the person reads them, **then** one page says in plain words what runs where and why, and what a queue ejection means.

## Open Questions

- **Write the workflow, or check the person's own.** The scaffold reads a managed repository's CI and never writes it (it derives check names and writes nothing hard-coded). This intent either lays down a CI workflow of its own, or checks the person's CI against the shape and says what to change.
- **The up-to-date rule.** Whether abcd turns off "require branches to be up to date" where a merge queue gates the default branch; the queue gives the same guarantee, and abcd's own repository kept the rule on 2026-09-01 (iss-2609012313465609 carries the measurement).
- **Where a platform flake lands.** Running a platform's suite only in the queue turns a flake on that platform into a queue ejection and a re-queue (iss-2610051434156079 is one); whether that cost is accepted, or the queue retries a platform job once.

## Decomposition

| Part | Kind | Home |
| --- | --- | --- |
| A managed repository's CI runs the full suite once, in the queue | capability | this intent |
| The full suite must run on the merge result of every change before it lands | trust rule | a decision record and a brief invariant, owed before planning |
| A remedy is filed whole, built where it can be, measured where it cannot | stance | `a-remedy-is-filed-whole` (principle) |
| abcd's own CI takes the shape first | plumbing | iss-2610110938282654, iss-2610110938294826, iss-2610110938293273 (built on `ci/full-suite-once-per-merge`), iss-2610110613372021 (superseded queue runs) <!-- record-lint: forward-looking --> |
| Parts that cannot be built yet | measured records | iss-2609012313465609 (up-to-date rule), iss-2610110939547495 (raise the macOS limit), iss-2610110939559446 (receipt as a status) |

Verdict: FILE-AS-IS, the routing awaiting the product thinker's confirmation.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
