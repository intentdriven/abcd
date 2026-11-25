---
id: itd-2611251725203320
slug: one-abcd-command-starts-a-session-on
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
refines: [itd-2611040344170206]
related_issues: [iss-2611032326573729]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# One command starts a session on abcd alone

## Press Release

> One abcd command starts a session on abcd alone, on a separate login abcd made. Maya types it in a terminal in any repository abcd manages. The session holds the few tools the work needs, abcd's hooks and the repository's conventions, and nothing else of Maya's: no connected services, no other plugins, no skills, no memory, no personal settings or instructions. The first time, abcd makes the separate login's directory and hands Maya the tool's own login step; abcd drain runs on the same one. Before the session begins abcd checks what it holds and names anything that should not be there.

## Why This Matters

No abcd command starts a session on abcd alone. On 2026-11-25 one was started from a hand-written script that names an abcd version, switches off each other plugin by name, leaves out the personal settings and instructions, lists the tools, and passes the conventions in. The terminal drive already starts its agents behind such a wall, with settings abcd writes and proves (`internal/core/runner/profile.go`, `wall.go`), but it refuses on an account with no profile directory, and no abcd command makes one (iss-2611032326573729). The fuller model, one profile per way of paying, is itd-2611040344170206, which this refines: it takes the smallest part a person needs to start, and leaves the registry, the names, keys and usage windows there.

## Mechanism

We expect one command to be enough because the drive's wall is already built and tested: the same directory, settings writer and live proof serve a session as they serve a stage. Shown wrong if a session so started holds a tool the list does not name, a plugin other than abcd, a skill, the person's instructions or memory, or a connected service; or if abcd's guard or rules are missing from it.

## Scope Conditions

- Holds for Claude Code, the one host abcd can start this way today; another is refused by name.
- Holds on one operating-system account. The separate login is not a sandbox: a session's shell reads whatever the account can.
- Holds for one profile per account on the person's own subscription; several profiles, and keys, stay with itd-2611040344170206.
- Holds where a person starts it in a terminal. A repository's configuration never starts one or widens its tools.
- Holds for an ordinary session a person talks to and for one left unattended: the away mode decides which, as now.

## Acceptance Criteria

- **Given** an account with no profile, **when** the person runs the command that makes one, **then** abcd makes one directory of the account's own under `~/.abcd.noindex/profiles/`, writes abcd's settings there and nothing of the person's, prints the tool's own login step for them to run, and never performs the login or reads, stores or forwards its token.
- **Given** a profile already there, **when** the command is run again, **then** it changes nothing but abcd's own settings, says what it found, and never touches the login state.
- **Given** a profile that is logged in, **when** the person starts a session on it in a managed repository, **then** the session holds the named tools and no others, abcd as its only plugin, no skill, no connected service, no browser tool, no memory, and none of the person's settings, hooks or instructions; abcd's guard and rules act in it, and the conventions reach it whole.
- **Given** that session's tools as the host reports them, **when** something the list does not name is there, **then** the start refuses naming it, or, where abcd can only see it after the start, says so in a line that cannot be missed; a tool the host will not give up is named in abcd's own list rather than passed over.
- **Given** no profile, or one not logged in, **when** the session is started, **then** it refuses and prints the one next step.
- **Given** a session start by an agent, or from inside an agent's session, **when** it is tried, **then** it refuses: the person starts it in a terminal.
- **Given** a person's own first prompt, **when** they pass it, **then** the session opens with it; with none, it opens waiting for them.
- **Given** the same profile, **when** `abcd drain --drive` starts, **then** it runs on it with no further step.
- **Given** any page or line that names the profile, **when** it describes it, **then** it says a separate login and settings on one account, never a sandbox.

## Open Questions

_None open. The commands' names are a technical choice for the spec, recorded as a decision with how to reverse it. The question tool stays among the session's tools: the guard refuses a question while the person is away, and an attended session needs it; no question was put._

## Decisions

1. Ruled by the product thinker on 2026-11-25: this is on the main branch before the six-day unattended run on this account starts, beside its two companions, and it is planned and built overnight in as few pull requests as possible. What was put to them is the part's one sentence; the criteria are the writing session's reading of it, shown to them before the build was started.
2. The live proof needs the person's login, which they give once, in their own shell; it waits for them and is not a reason to hold the code back from the main branch. The spec stays open until it is done.

## Decomposition

Hand-run four-piece table, 2026-11-25. One capability, this intent (`refines` itd-2611040344170206); the trust rule on what an unattended agent may hold is already accepted in adr-2610222222326104 and is restated nowhere; the directory, flags and settings are plumbing for the spec. Verdict: FILE-AS-IS.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
