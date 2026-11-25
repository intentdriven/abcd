---
id: itd-2611251725190796
slug: every-sub-agent-in-an-abcd-managed
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

# Every sub-agent starts under abcd's rules and the conventions

## Press Release

> Every sub-agent in an abcd-managed repository starts under abcd's rules and the repository's conventions, whoever starts it. Maya leaves an agent working overnight; it hands pieces to helpers of its own, and each helper knows how this repository commits, tests and lands before it touches a file, because abcd tells it so as it starts. Nothing in Maya's own setup has to be switched on for that, and a helper that skipped its instructions is named in the record Maya reads in the morning.

## Why This Matters

On 2026-11-25 a session was started with every plugin but abcd switched off. The repository's conventions file then reached no session, because a plugin built into the host loads it, not abcd. Passed to the main session by hand, it still reached neither of the two sub-agents tested, and neither named a rules domain: the rules loader runs when a person sends a prompt, and a sub-agent is sent none. The build loop's own implementer brief carries the conventions (`internal/core/implement/loop/brief.go`), so the gap is every sub-agent a session starts by hand, reviewers included.

## Mechanism

We expect a hook at a sub-agent's start to close the gap because the host delivers what that hook returns to the sub-agent itself: a test hook on 2026-11-25 gave a general-purpose sub-agent a check word it then repeated. The hook's output is capped (10,000 characters, against a 32,762-character conventions file in this repository), so it names the file to read and the rules to fetch, and carries the rules that fit. Shown wrong if a sub-agent started in a managed repository, with no other plugin on, works without having read the conventions file and nothing says so.

## Scope Conditions

- Holds for a host that runs a hook as a sub-agent starts and hands its output to the sub-agent. On a host that does not, the build loop's brief stays the only route, and abcd says so where it reports its hooks.
- Holds in a repository abcd manages, whichever worktree of it the sub-agent works in.
- Holds whether or not the host loads the conventions file itself: where it does, the sub-agent is told to read it once, not twice.

## Acceptance Criteria

- **Given** a managed repository and a session with no plugin but abcd, **when** it starts a sub-agent of any kind, **then** the sub-agent is told, before its first act, to read the repository's conventions file in the worktree it works in, and how to fetch the rules its task recalls, and can say the conventions file's check word where the file carries one.
- **Given** rules that fit the host's limit on a hook's output, **when** the sub-agent starts, **then** the rules its task recalls are in what it is told; what does not fit is named with the command that prints it, never cut silently.
- **Given** a sub-agent that ends without having read the conventions file, **when** it stops, **then** one line naming it is recorded where the run's own log is read, and nothing is refused.
- **Given** a repository abcd does not manage, or one whose rules are switched off, **when** a sub-agent starts, **then** abcd adds nothing.
- **Given** the hook fails or its time runs out, **when** a sub-agent starts, **then** the sub-agent starts all the same and the failure is one line on the hook's stderr.
- **Given** an abcd whose hooks lack the one at a sub-agent's start, **when** `abcd ahoy` reports the install, **then** the gap is named.

## Open Questions

_None open. A skipped reading is reported and never refused, on the stance that abcd warns where it cannot prove (dcn-2611041550129278); no question was put._

## Decisions

1. Ruled by the product thinker on 2026-11-25: this is on the main branch before the six-day unattended run on this account starts, beside its two companions, and it is planned and built overnight in as few pull requests as possible. What was put to them is the part's one sentence; the criteria are the writing session's reading of it, shown to them before the build was started.
2. Reported, never refused: see Open Questions.

## Decomposition

Hand-run four-piece table, 2026-11-25. One capability, this intent; the hook's wiring and the cap's handling are plumbing, carried by its spec and the brief's hooks chapter; no trust rule and no stance. Verdict: FILE-AS-IS.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
