---
id: itd-2611041815429986
slug: a-person-who-starts-a-run-finds-out-at
spec_id: spc-2611042207343783
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-63]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A person tells abcd where a project's tools live, and abcd checks them before a run starts

## Press Release

> A person who starts a build or a drain finds out at once if a tool on the project's list is missing, not when the run dies halfway. abcd keeps a list of the outside tools each project depends on, and a person tells abcd where each one lives on their computer. Before any work starts, a quick test checks that each listed tool is still the exact file the person named. "My overnight drain used to spend an hour and then fail on a tool it could not find," said Bob, a staff engineer. "Now it has what the list says it needs, or it tells me before it spends anything." An agent can stop and say it needs a tool, and abcd does not take its word for where one lives. A project's files only ever name what it needs.

## Why This Matters

A run can reserve its work, ask a model to judge it, start agents and spend money,
and only then find that a tool the repository needs is not there. The process's
PATH decides, and it is often not the terminal's: a hook, a timer or a launched
agent never ran the person's shell setup. On 2026-11-04, on one machine,
`scripts/abcd.sh` stopped with `go: command not found`, because the `go` on that
account's PATH was a link into a home folder the account could not enter. Nothing
had removed Go, and nothing said so until a person's session traced the link
(iss-2611041825149649). An agent can also start relying on a new tool with nobody
deciding it should.

abcd checks its own tools and its runners' binaries, and nothing checks the
repository's. itd-63 and `abcd ahoy`'s `dependency` gap explain and install tools
abcd's own capabilities use, from a fixed registry, and this is not an extension of
them: a repository's tool is never in that registry. `abcd land` says only that make
is not installed, and no start check looks at all. The timer of
itd-2611041656012738 and the pick-up of itd-2611041656024409 make this worse, since
both start with a minimal PATH.

The product thinker set the design on 2026-11-04: a list of each project's outside
tools, added to as work is built, registering where a new tool lives at the moment
it is approved, and a smoke test before work starts. After reading what a
tricked agent could do, they then ruled that a person always says where a tool lives
and abcd never accepts a location by itself. The state of the art is in
[the research note](../../research/notes/2026-11-04-repository-tools-on-path-sota.md).
The trust rule is adr-2611041815445203. An agent's request and the wrapper for host
sessions are itd-2611042148411751, and other languages' version checks are
itd-2611042148424105.

## Mechanism

We expect fewer runs to fail on a missing tool after spending, because the list says
what a project relies on, the registration says where it is, and a test before
anything is reserved costs far less than a lane that fails. We expect a tricked agent
to get less far, because abcd takes no location from what it writes to the project. This would
be shown wrong if lanes kept failing on tools the test called present, if the test
refused runs that would have worked, or if the registering step proved so tiresome
that people waived every tool.

## Scope Conditions

- Holds for tools on the list. A tool nobody listed is not tested, so it can still fail mid-run, and an agent can use one without saying so; itd-2611042148424105 names one a stage failed to find. <!-- cond: cond-2611042207347221 -->
- Holds as a filter against mistakes, not a boundary (adr-42). An agent on the person's own account can edit their PATH, their home folder or anything else they can write, and being at a terminal refuses a pipe or a routine but not an agent that makes itself a terminal. The planned fence of itd-2610161621364075 would stop those writes for agents abcd launches, and agents a host session starts stay outside it. A forge approval by any login with write access counts as a person's, so an agent holding a second such login could approve a list change, though registering the tool still needs a person. <!-- cond: cond-2611042207340521 -->
- Holds for the file the person named, not what it loads or calls afterwards. A harmful file a person registers is protected as faithfully as a good one, and the fingerprint is checked before a command starts, so a swap during a long command, what Go's own child programs do, and a detached child that outlives its PATH directory are all out of reach. <!-- cond: cond-2611042207349594 -->
- Holds fully for commands abcd itself starts: the landing preflight and agents a runner launches. A host session's agents inherit the host's PATH, which abcd cannot set, so until itd-2611042148411751 adds the wrapper the start is refused when the PATH abcd was called with would not reach the registered file. abcd cannot see the host's own tool shells, so this is a filter too. <!-- cond: cond-2611042207341830 -->
- Holds for what abcd puts first on PATH, not for what the repository does next: a make file can reset PATH or set `GOTOOLCHAIN`, as abcd's own does. <!-- cond: cond-2611042207348403 -->
- Holds for Go's version only. Any other listed tool is checked for its file and fingerprint and is never run. <!-- cond: cond-2611042207344616 -->
- Holds on macOS and Linux. Windows is the concern of itd-2610070544145422. <!-- cond: cond-2611042207348171 -->
- Holds where a copy exists on the machine. Registering records a copy and fetches nothing; installing is itd-2611041815433220. A tool's own default download during a stage, such as Go switching toolchains, stays as the person's setup has it, and the report says it will happen. <!-- cond: cond-2611042207342301 -->
- Holds for updates only as far as a report: abcd says when a copy differs from what the project declares, from what is on disk, and a forge's update bot proposes new versions. The bot needs setting up, covers common tools only, and is not abcd's to run. <!-- cond: cond-2611042207349041 -->

## Acceptance Criteria

Adding and registering:

- **Given** a person at a terminal names a tool, why it is needed and where it lives, **when** they add it, **then** the committed list gains the name and the reason, and for Go the file that declares its version, with no path; and the path and the file's SHA-256 are recorded in the person's home for this computer. Registering Go records `go` and `gofmt`, each with its own fingerprint; any other tool records its one file.
- **Given** a tool already registered on this computer, **when** another repository lists it, **then** that copy serves, with no second registration.
- **Given** a caller off a terminal, such as a pipe, a routine, CI or an agent's shell, **when** it tries to add, register or waive a tool, **then** it is refused and nothing is written, naming that this is a person's act.
- **Given** an agent's shell command to add, register or waive a tool, **when** the shell guard reads it under any program name, **then** it is refused as other person acts are, where a host has hooks.
- **Given** a person is registering a copy, **when** abcd asks them to confirm, **then** the question shows the file's path, its owner, when its contents and when its metadata last changed, and what a link points at, read from the file itself, and no text from the list, an issue or an agent.
- **Given** a copy that is relative, inside any working tree, reached through a directory others can write, or a link to somewhere this account cannot enter, **when** someone tries to register it, **then** it is refused naming why, with both a link and what it points at held to this, and registering never downloads anything.
- **Given** a name outside the strict grammar, a reserved name, which means a shell, `git`, `abcd`, `node` and the command each runner starts, or a path whose own last part differs from the tool's name, **when** the list is read or a tool is added, **then** it is refused. `go` and `make` are not reserved, though abcd runs them, because a repository may need them.
- **Given** a reason over the length limit, or with a character outside its allowlist, which holds no path separator, `~`, `$`, backtick, quote or control character, **when** the list is read, a tool is added or an agent asks for a tool, **then** it is refused, so a repository's reason cannot carry a path to type, though it can still describe a place in words; a reason is always shown as data.
- **Given** the home file holding the locations is not a regular file this account owns that no one else can write, **when** the test reads it, **then** the test fails loudly, and no file in any repository can supply a location.

The test:

- **Given** a listed tool with no registration, **when** the test runs, **then** it fails naming the command that registers it, even if the PATH finds a copy. It may describe that copy, it never uses it, and it prints no ready-to-paste command holding that path, so the person types where.
- **Given** a registered file has changed since it was registered, an ordinary upgrade included, **when** the test runs, **then** it fails saying the file changed, cause unknown, and where a link pointed before and now; the changed file is not run, and only a person can register it again.
- **Given** every listed tool is present and unchanged, **when** a person starts a build or a drain, **then** the test passes and names each copy.
- **Given** a listed tool fails the test, **when** a person starts a build, **then** the start is refused before anything is reserved, created or spent, naming the tool, the version needed and how to register it, with an exit code of its own.
- **Given** a listed tool fails the test, **when** a drain moves, **then** the test runs once before the first issue, nothing is reserved or judged, no model runs, and the drain looks again at its next wait, carrying on once the tool is registered.
- **Given** a person at a terminal waives a tool for this repository on this computer, **when** the test runs, **then** the report calls it waived and says a command needing it may fail mid-run, the test passes, and it gets no link and no check; no repository configuration can set a waiver, and a shallow checkout gets none.
- **Given** a process whose PATH holds only the system directories, such as a timer, **when** it runs the test, **then** it finds every registered copy.
- **Given** a host session drives the lane, **when** the PATH abcd was called with resolves a listed tool's name to no file or to a file other than the registered one, **then** the start is refused before spending, naming the fix, which is to put the registered copy on the host's PATH.
- **Given** a run resumes or a drain opens another lane, **when** the lane's base commit changed the list or a version file, **then** the test runs again on what that commit holds.
- **Given** a repository with no list and no version files, **when** a run starts, **then** the test says there was nothing to check and refuses nothing.

Running a command:

- **Given** a command abcd starts, **when** it runs, **then** a directory made for that command under the person's abcd home comes first on PATH, holding only the registered programs the repository lists, each fingerprint is checked immediately before, and the directory is removed when the command ends.
- **Given** a fingerprint no longer matches at that check, **when** the command would start, **then** it does not start, the report names the file, and a lane's item waits as it would for a missing tool, with its work kept.
- **Given** an inherited `GOROOT` or a pinned `GOTOOLCHAIN`, **when** the test runs, **then** the report warns that it may change which copy runs.

An agent needing a tool:

- **Given** a lane's agent finds it needs a tool not on the list, **when** it runs the asking command its brief names, **then** that item waits with the tool and the reason recorded and its work kept, other items carry on, and the list is not changed.
- **Given** an item waiting on a tool, **when** the tool is on the list and registered, **then** the next build or drain takes the item up from the kept work.
- **Given** a lane's change touches the list, **when** it reaches landing, **then** landing refuses it naming the tool.
- **Given** a change touches the list and anything else, **when** it reaches landing or abcd's check in CI, **then** it is refused.
- **Given** a change to the list alone, **when** the merge gate decides, **then** it waits for a person's approval of that exact commit under any facilitator mode, and no review receipt stands in for it.

Go:

- **Given** a Go of 1.21 or later below `go.mod`'s `go` line and a `GOTOOLCHAIN` that lets it switch, **when** the test runs, **then** the report says it will fetch the newer toolchain and the start is not refused; with `GOTOOLCHAIN=local` the start is refused and names both versions.
- **Given** a `go.mod` or `go.work` whose `toolchain` line names a suffixed toolchain, which chooses what executes, **when** abcd reads it, **then** the start is refused and nothing it names is run.
- **Given** the list points Go at a version file other than `go.mod` or `go.work`, **when** the list is read, **then** it is refused; each tool has its own closed set.
- **Given** a `go` line in a form abcd does not recognise, **when** abcd reads it, **then** the report calls it unrecognised and it is never used to compose a path.
- **Given** a repository asking for a newer version than the registered copy, **when** abcd asks that copy its version, **then** nothing is downloaded, whatever the tool's own default would do; a probe that does not answer in time is reported as not verified.
- **Given** the registered copy is a script, or a link to a program of another name as a version manager's launcher is, **when** the test runs, **then** it is not run, and the report calls it present with its version not verified.
- **Given** a listed tool abcd has no version check for, **when** the test runs, **then** abcd checks the file and its fingerprint and does not run it.

The report:

- **Given** a person asks what the repository needs, **when** the read-only report runs, **then** it lists each tool, why it is needed, the copy, where, its version against the declared one and the tool's own upgrade command where they differ, in text and in JSON, with no network request and nothing written.
- **Given** a `go.mod` and no listed Go, **when** the test runs, **then** the run proceeds, and the report warns and proposes adding it.
- **Given** a repository with version files and no list yet, **when** a person asks what it needs, **then** the report proposes each tool and changes nothing until they add it.

## SOTA

Three reviews on 2026-11-04 are in [the research note](../../research/notes/2026-11-04-repository-tools-on-path-sota.md). Existing alternatives: version managers such as mise and asdf (mature, but they install by default, shims re-read each directory's configuration, and mise had two trust bypasses in 2026); hosted agents' setup scripts (mature, and they run on a machine the product manages, not a person's account); doctor commands such as Flutter's and mise's (mature, and they report without gating a run). Approval by file hash exists in Santa and in tool pinning (mature there), and no shipping coding agent was found doing it for tools. Weakness: most evidence on approving tools for agents is vendor statements and single studies, and none tests this exact chain.

Path taken: 3, bespoke, because no existing tool keeps a list of names and reasons with a person-given location, and depending on a version manager is what adr-2611041815445203 rejects.

## Decisions

The product thinker's rulings, 2026-11-04, rewritten:

1. What stops a run is what is on the list. A version file found outside the list warns and proposes adding it.
2. abcd reports and a forge's bot updates: abcd says when a computer's copy differs from what the project declares, and new versions reach the project as reviewed changes. abcd does not look online or update anything.
3. Registering where a new tool lives happens when it is approved, and that is how abcd knows. A smoke test before work starts confirms everything is there.
4. A person always says where a tool lives. abcd never accepts a location by itself, a copy already on the PATH included, even though every new tool on every computer then waits for someone.

Recorded with no question put, because the record leaves one defensible answer. Technical facilitator on delegation, 2026-11-04:

5. Invariant 2 stands, read in its context: it concerns adapters over abcd's own native defaults, and a repository's own build tool has no native path to degrade to. Its closing words, taken alone, say abcd never blocks on an absent tool, so it now points to invariant 26. The product thinker confirmed this narrower reading at sign-off, 2026-11-04.
6. An autonomous facilitator approves neither a location nor a name, withdrawing the first draft. Ruling 4 settles locations. For a name, review found a repository can turn the mode on for itself on its own default branch, and a session cannot prove it is not the lane that asked. It would also save no wait, since a person must register the tool anyway.
7. "A person" means at a terminal, the precedent the identity step of itd-131 and the drain rule's offer set: a scripted yes is not a person's. The verbs also join the shell guard's person acts, with `abcd land --approve`. Neither can prove who is typing: a pipe is refused and a pseudo-terminal is not. So the record says this is a filter (adr-42) and claims no boundary, as `enforcement-claims-are-facts` requires.
8. Approval covers the file as well as the path, with a SHA-256 fingerprint any change voids. A check at each start needs no daemon and no new dependency. The cost is one more registration after each upgrade. abcd cannot tell an upgrade from a swap, so it says the cause is unknown and shows the facts, and registering again can still become a habit.
9. Registration is per computer, not per repository, which fits the product thinker's description of it as part of the development environment and saves repeating it for each project. A repository can only choose among copies the person registered, and its own commands could already run those. What listing adds is moving a copy ahead on PATH before repository code runs, which could start a harness on an old registered interpreter, so the programs a harness runs on are reserved. Two repositories that need different copies of one tool are not served yet; itd-2611042148424105 carries that, because Go switches its own toolchains.
10. A waiver in the person's home, for one tool, one repository and one computer, replaces the override flag first drafted. Any agent can type a flag, and a flag cannot help a run a timer starts. Every lane takes the test whatever its issue touches, since a docs fix still passes `make preflight`, and the waiver is the answer for a tool this computer does not need.
11. The test runs after the claim and peers checks and before the budget check and any reservation, once per build start and once per drain move, never in the per-candidate check `abcd build next` calls.
12. A drain waits as a whole on a failed test, under the case in `waiting-holds-one-item-never-a-session` where every item waits: a missing listed tool is a fact about the computer, so every lane would stop on it alike, and the drain names the wait and when it looks again. An unlisted tool one lane needs holds that item alone. That wait is new because a hand-back discards work and holds the item until its text changes.
13. The verb is `abcd needs`, a read-only report when bare, with sub-verbs to add, register, waive and ask, and the list is `.abcd/needs.json`. `abcd tools` and `.abcd/tools.json` would read as `~/.abcd.noindex/tools.json` and `internal/core/tools`, which hold abcd's own installed tools. The list is its own file since no existing format holds a tool with no version and a reason, and a version stays in the file that already declares it, so bots can bump it. It is not an `abcd ahoy` gap, since nothing here is installed or adopted; `abcd ahoy doctor` names it.
14. This is the first of three stages, split on the security review, with installing filed apart. It delivers the list, registration and the test on their own. An agent can only say it needs a tool until itd-2611042148411751 turns that into a proposed change, and the version report covers Go until itd-2611042148424105. Go comes first because its forms are few and the failure that prompted this was Go.

## Open Questions

_None open. The product thinker's answers and the recorded decisions settle the ones raised._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect fewer runs to fail on a listed tool after spending, because a person says where each tool lives and a test runs before anything is reserved; this is wrong if lanes keep failing on tools the test called present, if the test refuses runs that would have worked, or if registering proves so tiresome that people waive every tool
