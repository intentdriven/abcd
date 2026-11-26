---
id: itd-2611251725203320
slug: one-abcd-command-starts-a-session-on
spec_id: spc-2611251757236507
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610222215485265]
refines: [itd-2611040344170206]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# One command starts a session on abcd alone

## Press Release

> One abcd command starts a session on abcd alone, on a separate login abcd made. Maya, an autonomous-development practitioner, types it in a terminal in any repository abcd manages. The session holds the few tools the work needs, abcd's hooks, and the repository's conventions, and nothing else of Maya's setup: no connected services, no plugins of Maya's, no memory, no personal settings or instructions. The first time, abcd makes the separate login's directory and hands Maya the tool's own login step; abcd drain runs on the same one. The separate login is a login and settings of its own on Maya's account, not a sandbox, so a command the session runs can still read what the account can. Before the session begins abcd proves the wall in a short launch with the same settings and tools, and names anything there that should not be; what the host adds after the start abcd cannot see, and says so.

## Why This Matters

No abcd command starts a session on abcd alone. On 2026-11-25 one was started from a hand-written script that names an abcd version, switches off each other plugin by name, leaves out the personal settings and instructions, lists the tools, and passes the conventions in. The terminal drive already starts its agents behind such a wall, with settings abcd writes and proves (`internal/core/runner/profile.go`, `wall.go`), but it refuses on an account with no profile directory, and no abcd command makes one (iss-2611032326573729). The fuller model, one profile per way of paying, is itd-2611040344170206, which this refines: It takes the smallest part a person needs to start, and leaves the registry, the names, the keys, and the usage windows there.

## Mechanism

We expect one command to be enough because the drive's wall is already built and tested: The same directory and the same hooks serve a session as they serve a stage, and the proof runs on a short launch with the session's own settings and tools. Shown wrong if that proof passes and the session, read afterwards from its transcript, held a tool the list did not name when it started, a plugin other than the host's own, the person's instructions or memory, or a connected service, or lacked abcd's guard or rules.

## Scope Conditions

- Holds for Claude Code, the one host abcd can start this way today. <!-- cond: cond-2611251757239976 -->
- Holds on one operating-system account. The separate login is not a sandbox: A session's shell reads whatever the account can. <!-- cond: cond-2611251757231162 -->
- Holds for the person's own subscription. An account may already hold profile directories the person made and named (`commands/drain.md`); the registry, the name list, keys, and usage windows stay with itd-2611040344170206. <!-- cond: cond-2611251757234410 -->
- Holds where a person starts it in a terminal. <!-- cond: cond-2611251757239987 -->
- Holds for a session a person starts; leaving it to work alone is the away mode's (itd-2610180912189125), which this does not change. <!-- cond: cond-2611251757234659 -->

## Acceptance Criteria

- **Given** an account with no profile, **when** the person runs the command that starts a session, **then** abcd makes one directory this account owns that no one else can write, under `~/.abcd.noindex/profiles/`, under the plain name the person gives, which `--profile` then takes, or a default one abcd ships that names no provider and no account, writes abcd's settings there and nothing of the person's, prints the host's own login command for it, which opens no session, starts no session, and never performs the login or reads, stores, or forwards its token.
- **Given** a profile already there, **when** the command is run again, **then** it changes no file in the profile, says what it found, and never touches the login state.
- **Given** a profile that is not logged in, **when** the session is started, **then** it refuses and prints the host's own login command.
- **Given** an account holding several profile directories, **when** a session is started with no name, **then** it refuses and lists them; with `--profile <name>` it starts on that one.
- **Given** a profile that is logged in, **when** the person starts a session on it in a managed repository, **then** the session is started with the tools on the list abcd prints at the start and no others, no plugin but the host's own, no server, no browser tool, the host's memory and instruction files switched off, no key in its environment, and none of the person's settings, hooks, skills, or instructions; the conventions file is handed to it whole as a launch argument; its settings are handed to it at the start, so no file a drive reads is written; and its shell runs under the person's home and takes none of a stage's files. The host's built-in skills and agents stay and are named. abcd's command pages and agents are not in it.
- **Given** the settings abcd hands the session, **when** they are compared with the plugin's hook manifest, **then** every event the manifest wires is wired there to the same verb, the one at a sub-agent's start among them (itd-2611251725190796).
- **Given** the arguments abcd will start the session with, **when** one short launch with those same arguments on the same profile, written to the launch ledger, reports a tool, a server, a plugin other than the host's own, or a key in use that the list does not name, or does not show the guard refusing a command it is meant to refuse, **then** the start refuses naming it and no session opens. A named tool that launch does not report is said, not refused. The start's line says what was read: a launch with the session's arguments, not the open session.
- **Given** a session start from an agent's shell, read as a host session's marker in the environment, or with no terminal on all three streams, **when** it is tried, **then** it refuses; abcd checks those two things and no more.
- **Given** a person's own first prompt, **when** they pass it, **then** the session opens with it in the person's terminal, in place of the abcd process, which says nothing after its start line; with none, it opens waiting for them. The ledger holds the short launch, not the session.
- **Given** a permission mode the person names on the command, **when** the session starts, **then** it starts in that mode, and in the host's default when they name none; a repository file never sets it.
- **Given** a repository file that names a tool or asks for a start, **when** a session is started, **then** the file starts nothing and adds no tool.
- **Given** the same profile, **when** `abcd drain --drive` starts, **then** it runs on it with no further step.
- **Given** the warning a session off a profile gets and the drive's no-profile refusal, **when** this ships, **then** each names the new command and no line says abcd has none.
- **Given** any page or line that names the profile, **when** it describes it, **then** it says a separate login and settings on one account, never a sandbox.

## Open Questions

_None open. Decisions 3 to 8 were settled with no question put; each says why._

## Decisions

1. The product thinker's, on 2026-11-25 (dcn-2611251726008978): this part and itd-2611251725190796 are on the main branch before their six-day unattended run starts; itd-2611251725209625 follows in that run. They chose the part from its one sentence and gave no wording for a criterion. The criteria, the Mechanism, and the scope conditions are the writing session's; they were listed to the product thinker in plain words, not walked one by one, and they then said to start building. That is the consent the plan stands on. They review each on return, and a change then is made through the record.
2. The writing session's, no question put, for the product thinker's review on return: The live proof needs the person's login, which they give once, in their own shell; it waits for them and is not a reason to hold the code back from the main branch. The spec stays open until it is done, so the intent stays planned and the landing change carries no `Delivers:` trailer.
3. The command is `abcd start`. It makes the profile where there is none and starts a session where there is one that is logged in, so one command carries both parts the product thinker chose. No question was put, since a name is a technical choice; to reverse, rename the verb before its release.
4. The question tool stays among the session's tools. No question was put: An attended session cannot be asked anything without it, and the guard refuses a question while the away mode holds.
5. The shipped list is the shell, reading, editing, and writing files, sub-agents and stopping one, the question tool, the monitor, and the session's own timers. Messages to other sessions are outside it, as decision 14 of itd-2611040344170206 rules; widening the list in the person's own machine configuration stays with that draft. No question was put, since both follow from rulings on the record.
6. The session's settings go to it at the start and no settings source is read, so a drive on the same profile is never disturbed; a test launch on 2026-11-25 (host 2.1.332, by hand) ran a hook handed over that way with every settings source switched off. No question was put: technical.
7. The name is the person's own, or the shipped default, until the name list of itd-2611040344170206's decision 15 is built; no question was put, and the product thinker reviews it on return.
8. The permission mode is the person's, named as they start it. No question was put: The person chooses it at their own terminal each time, and abcd grants nothing by itself.

## Decomposition

Hand-run four-piece table, 2026-11-25. One capability, this intent (`refines` itd-2611040344170206); the trust rule on what a walled agent may hold is adr-2610222222326104 as amended on 2026-11-04, and that a session coordinating unattended work starts walled is dcn-2611031722512037; the ADR names a no-host drive, so whether a session start needs a line there is flagged for the product thinker, never auto-classified. `builds_on` itd-2610222215485265. It takes from itd-2611040344170206 the criteria opening "Given a person with a subscription to a supported tool" (without the name list and the registry) and "Given any surface … that names a profile"; that record's Evidence says so in the change that ships this. The directory, flags, and settings are plumbing for the spec. Verdict: FILE-AS-IS.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the product thinker expects that a run with every other part of the host switched off will show whether unattended work on abcd alone is possible, and which parts abcd has leaned on; a run that needs one of them switched back on shows it is not yet.
