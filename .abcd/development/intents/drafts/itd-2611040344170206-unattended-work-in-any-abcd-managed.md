---
id: itd-2611040344170206
slug: unattended-work-in-any-abcd-managed
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
refines: [itd-2610222215485265]
related_issues: [iss-2611032326573729, iss-2611032327111678]
severity: major
origin: researcher-authored
production_mode: hand-written
---

# Unattended work runs on a named profile, one per agent subscription

## Press Release

> **Unattended work in any abcd-managed repository runs on a profile: a separate login and settings for one agent subscription, kept beside the others under abcd's home.** Maya keeps three subscriptions of two agent tools. For each one they type one abcd command naming the tool; abcd makes a directory with a random name under `~/.abcd.noindex/profiles/`, writes that tool's settings there with abcd's own hooks wired in, and hands them the tool's own login step to run. abcd never sees the token. A registry they can read says which tool and which subscription each name holds, and `abcd ahoy` says whether each profile is ready: logged in, carrying abcd's guard, rules loader and session names, and admitting only the tools on abcd's allow-list. When Maya starts `abcd drain --drive` or any other unattended run, abcd picks a profile for each role from those whose usage window is below 95 per cent, proves that tool's wall on that profile before it runs a stage, and refuses a session that is not on a profile at all. Their own attended sessions are untouched. "I used to keep a hand-made directory per login and a start script nobody else could read," said Maya, an autonomous-development practitioner. "Now a profile is a thing abcd knows: it makes it, checks it, picks it, and says plainly what it is, a separate login on my account, not a wall around the account."

## Why This Matters

The subscription route of `abcd drain --drive` (itd-2610222215485265) rests on a dedicated configuration directory the person logs into once, and its spec (spc-2610230813348813) names no location for it, so the first live proof (dcn-2611031720003633, lab lab-261103171720-c0bdccf) chose a directory by hand and started it from a hand-written script that pinned abcd's hook to one versioned plugin path in one account (iss-2611032326573729). No abcd verb knows that directory: a session cannot find, create, pick or report it, and every account repeats the setup by hand.

The same evening's test (2026-11-03) found three things the directory alone does not give. A wall defined by denying connected services left built-in tools that reach the person's account in the coordinating session, so the wall has to name what a session may hold, not what it may not. A tool appeared in a session after its start, so a wall proved once at the start does not stay proved. And a profile that kept only the guard hook lost its session name and the rules loader, so none of the repository's rule domains reached the agents working there. The third attempt, on an allow-list of named tools, ran the implement and validate stages on a subscription with no API key and reached an open pull request, which is the evidence that the route works once the wall is an allow-list.

Nothing enforces the wall (iss-2611032327111678): the verbs that start unattended work accept a session holding the person's connected services and personal memory as readily as one on a profile, and the one exception made on 2026-11-03 (dcn-2611032348409091) had to be ruled by hand because no verb could tell the two apart. The product thinker ruled that watching is no control, since a person cannot interrupt a tool call as it runs (dcn-2611031722512037), so the wall is the control, and a control nothing checks is a request.

A person running abcd unattended on subscriptions, moving to a key when a window reaches 95 per cent and back when it resets (dcn-2611031615094092), needs several logins of several tools side by side, each one abcd can create, prove, pick and report. That is what a profile is.

## Mechanism

We expect a profile to keep an unattended run off the person's own setup, and to stay that way through a run, because each tool's login, settings and hooks live in a directory the tool itself treats as its whole home, and the tools a session on it may hold are named in one allow-list abcd re-reads before each stage, so what reaches the agent is what the list admits rather than whatever a deny list did not happen to name. The claim is shown wrong if a stage started on a profile, on a machine where the person's own instructions, memory, connected services, hooks and saved sessions are all present, holds a tool the list does not name or reaches any of those, in the live check abcd runs before the stage.

## Scope Conditions

- Holds for an agent tool whose command line can be pointed at a separate configuration directory and started with a named list of tools, and for which abcd has a runner: Claude Code and OpenCode today; Codex and Grok when their runners exist. Each tool is admitted by its own live proof, never by another's.
- Holds on one operating-system account. A profile is a separate login and settings on that account, not a sandbox and not isolation: a stage's shell reads whatever the account can, the person's own configuration and the other profiles' logins included. This matters most where several providers' logins sit side by side. What the account itself cannot reach is the sandbox intent's work (itd-2610161621364075), a later and separate step.
- Holds for the subscription route, where the tool's own login is the credential. What the key route does on a profile is the second open question below.
- Holds where the tool reports its own usage window; a profile whose tool reports none is chosen by the rule the third open question settles.
- Holds on macOS and Linux, where `~/.abcd.noindex/` is abcd's home; there is no Windows target.
- Holds for sessions a host reports enough about for abcd to tell a profile from the person's own session; what a host can report is the first open question.

## Acceptance Criteria

- **Given** a person with a subscription to a supported tool, **when** they run the profile-creating command naming the tool, **then** abcd makes a directory with a random name under `~/.abcd.noindex/profiles/`, writes that tool's settings there with abcd's guard, rules loader and session-name hooks wired to a path that survives a plugin upgrade, records the name, the tool and the date in a registry the person can read, and hands them the tool's own login step to run in their own shell. The name carries no provider and no account; abcd never performs the login, and never reads, stores or forwards the token.
- **Given** a tool abcd has no runner for, **when** the creating command names it, **then** the create is refused and names the runner it lacks; nothing is written.
- **Given** one or more profiles, **when** `abcd ahoy` reports them, **then** each shows its name, its tool, its creation date and its readiness: logged in (as the tool reports it, by state alone, never an account identifier), every abcd hook present and resolvable, the allow-list the profile starts with, the live proof's state for that tool version, and the usage window where the tool reports one. A gap is named with the step that closes it. Nothing it prints identifies the account behind a profile.
- **Given** the allow-list of named tools, kept in one place in the person's machine configuration and never in a repository file, **when** a stage is about to start on a profile, **then** abcd re-reads the list and the session's tool set as the host reports it, and a tool outside the list stops the stage before it starts, naming the tool. A session holding built-in tools that reach the person's account is not on a ready profile, whatever its connected-service settings say.
- **Given** a tool with no live proof recorded for the version a profile holds, **when** that profile would first run a stage, **then** abcd proves the wall on it first: the person's own instructions, memory, connected services, hooks other than abcd's, and saved sessions are out of the agent's reach, the tool set is the allow-list, the guard refuses a harmless command it is meant to refuse, and no API key is in the environment. It records the proof with the tool's version, refuses the stage naming any wall that failed, and a version change voids the proof.
- **Given** a tool that cannot carry abcd's guard (no hook surface, or a start mode that skips hooks), **when** a role would be handed to a profile of it, **then** no stage runs there and abcd names why; the role goes to another profile or is handed back.
- **Given** several ready profiles, **when** `abcd drain --drive` or any other unattended start hands out a role, **then** it picks a profile by the role's route and by window: one whose window is below 95 per cent, the next when one is at or above, and when none remains the key route per dcn-2611031615094092 or a wait, returning to a profile when its window resets. The run record names the profile by its name for every launch.
- **Given** a session that is not on a profile, **when** it starts unattended work (`abcd mode autonomous`, `abcd drain --drive`, a build or drain run in autonomous mode, or a coordinator's start), **then** abcd refuses, or warns so that the line cannot be missed (the first open question decides which, and for which verb), naming what the session holds and the command that starts one on a profile. A session in managed mode making one move at the person's direction is unaffected.
- **Given** any surface that names a profile (the registry, `ahoy`, a refusal, a command page, the docs), **when** it describes one, **then** it says it is a separate login and settings on one operating-system account, and that a stage's shell reads what the account can; it never calls a profile a sandbox or isolated.
- **Given** the hand-written setup this replaces (a directory per account and a start script outside abcd), **when** this ships, **then** the drive's precondition that reads "the walled configuration directory has no login" reads the profile registry instead, the live wall proof of spc-2610230813348813's first step becomes the per-profile proof above, and the command pages name the profile store as the one home.

## Open Questions

- **Refuse or warn, and on what evidence.** What a host reports about its own session is the limit of what abcd can check: the configuration directory in the environment, and the session id, working directory and permission mode in a hook's input, are visible to abcd; the session's tool list is shown to the model and to no hook. So: (a) refuse whenever the session's configuration directory is not a registered profile, and warn only where the host reports no directory at all; (b) warn always and refuse never, since a wrong refusal stops the person's own work; (c) refuse a drive and `mode autonomous`, which are unattended by their own declaration, and warn on a single build or drain step, which a person may be driving by hand. Each is a real choice; the planning interview puts it to the product thinker with what each host can report.
- **The key route.** adr-2610222222326104 runs the key route in the tool's bare mode, which reads no login and also runs no hooks, so it carries no guard; this draft's rule is that a tool that cannot carry the guard runs no stage. Either (a) a key-route stage also runs on a profile, a directory with no login and the key handed to that launch alone, so the guard holds and the bare mode is dropped for stages, or (b) the key route keeps the bare mode and the guard rule is narrowed to the subscription route, with the gap named in the run record.
- **Which window, and a tool that reports none.** A tool may report a short window and a long one; (a) the 95 per cent rule reads the shorter, since it is the one that stops a stage mid-run, or (b) both, passing a profile over when either is at the mark. A tool that reports no window is (a) chosen only when no reporting profile is free, or (b) never chosen unattended.
- **Tools that reach other sessions on the machine.** A session on a profile may hold built-in tools that reach other local sessions, which may be the person's own: (a) outside the allow-list by default, or (b) admitted, since the reach is to the machine and not the account's services.
- **What the registry records.** (a) Name, tool and date alone, so nothing in the registry identifies a subscription, and the person tells the profiles apart by the tool's own status; or (b) a label the person types at creation as well, never read from the tool, for a person with several subscriptions of one tool.
- **Who may change the allow-list.** (a) The person alone, in their machine configuration, an agent's session refused; or (b) a repository may narrow it for its own runs and never widen it.

## Decisions

The product thinker's, on 2026-11-03, restated:

1. Every session that coordinates unattended work starts on the wall itself; a stage run as a host sub-agent gets file and shell tools only; a stage that cannot run within that limit does not run (dcn-2611031722512037, refining dcn-2611031720012399).
2. Watching is no control: a person cannot interrupt a tool call as it runs, so a person's presence never makes an unwalled agent acceptable (dcn-2611031722512037).
3. The dedicated directory's home is `~/.abcd.noindex/profiles/`, one directory per agent subscription, named for what it is: a profile, a separate login and settings on one account, never a sandbox or an isolated environment (recorded in iss-2611032326573729's remedy).
4. Each profile gets a random name at creation, so the name carries no provider or account; a registry the person can read says what each holds (the same remedy).
5. Each tool's wall is proved on its own before that tool runs a stage; a tool that cannot carry abcd's guard does not run one (the same remedy).
6. Sessions use the person's subscriptions first and move to a key when a window reaches 95 per cent, returning once it resets (dcn-2611031615094092).
7. The first live proof runs on the account the drive will run in, after the person logs the directory in once (dcn-2611031720003633).
8. Real isolation follows as its own step (a separate operating-system user, a container, or the host's own sandbox per directory); it is not this intent (the same remedy; itd-2610161621364075 is the planned sandbox).

## Evidence

- 2026-11-03, the subscription test's first attempt: under a wall that denied connected services, the coordinating session still held built-in tools that reach the person's account, and stopped before any stage; the deny-list wall is why the allow-list is the rule above.
- 2026-11-03, the second attempt: all six wall checks passed at the start, then a tool outside the allow-list appeared in the session within two minutes, and the session stopped; the re-read before each stage is why.
- 2026-11-03 to 2026-11-04, the third attempt: on an allow-list of named tools, with no API key in the environment, the implement stage and both validate reviewers ran as file-and-shell sub-agents on a subscription, the lane's full check passed, and the branch reached an open, unarmed pull request; the intent audit and the arming did not run. The tool's usage recorder appended only on a change, so the per-stage cost went unmeasured; the sub-agents' token counts were the only per-stage figure.
- 2026-11-03, the setup's gaps: the hook was pinned to one versioned plugin path in one account; only the guard hook was wired, so the session had no name and the rules loader injected no domain; the guard's canary had to be a command the guard refuses, since a harmless one proves nothing.

## Decomposition

Hand-run four-piece table, 2026-11-04, from the product thinker's rulings of 2026-11-03 and the subscription test's three attempts. Routing proposed, for the product thinker to confirm; parts routed elsewhere are named here and not filed.

| Part | Kind | Home |
|---|---|---|
| profiles under `~/.abcd.noindex/profiles/`, random names, the readable registry, one creating command with the person's own login step, `ahoy` readiness, the per-tool live proof, the choice by role and window, abcd's hooks wired in, and the refusal or warning for a session not on one | capability | this intent |
| an agent abcd starts unattended holds only the tools a single allow-list names, re-read before each stage; a tool that cannot carry abcd's guard runs no stage; a profile is a login boundary, never an isolation boundary | trust rule | an ADR owed, extending the wall clause of adr-2610222222326104 from a dedicated configuration directory with deny switches to a profile with an allow-list of named tools, and a brief invariant when it is accepted; not filed here |
| a wall is defined by what it admits, never by what it denies | stance | a candidate principle, or a bound in an existing one, for the product thinker to confirm; not filed here |
| the directory layout, the registry's shape, each tool's start flags, where the allow-list lives, how the tool set is read back from the host | plumbing | the brief's runner and drain chapters, carried by this intent's spec |
| a separate operating-system user, a container, or the host's own sandbox per profile | capability | the planned sandbox intent itd-2610161621364075 and the later step the remedy names; not this intent |

Verdict: FILE-AS-IS, the trust rule and the stance held for confirmation rather than filed. Typed links: `refines` itd-2610222215485265, whose subscription-route wall becomes a profile among several and whose spec's first step becomes the per-profile proof; `related_issues` iss-2611032326573729 (no fixed place or command) and iss-2611032327111678 (nothing enforces the wall). Flagged for the product thinker, never auto-classified: this draft reverses in part the wall list of spc-2610230813348813, keeping abcd's own hooks where the spec disabled all hooks (already ruled for the guard by dcn-2611031722512037; the rules loader and session names are new) and replacing the connected-services deny with an allow-list of named tools. Neither sandbox draft is superseded or refined: itd-2610161710315026 (a declared list of hosts) fences from outside the launches a profile makes, and itd-2610161710316538 (a run reports how isolated it was) gains the profile a run is on as one of its facts, so each is proposed to `build on` this draft, an edge set on them with `abcd intent edge` once the product thinker confirms. `supersedes` nothing.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
