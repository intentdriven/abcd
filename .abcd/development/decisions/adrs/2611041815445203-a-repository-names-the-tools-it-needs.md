---
id: adr-2611041815445203
slug: a-repository-names-the-tools-it-needs
status: proposed
date: 2026-11-04
supersedes: null
superseded_by: null
related_intents: [itd-2611041815429986, itd-2611042148411751, itd-2611042148424105, itd-2611041815433220, itd-2611041656024409]
related_rfcs: []
related_adrs: [adr-22, adr-38, adr-42, adr-2610150603484802, adr-2610222222326104, adr-2610241057124151]
---

# ADR-2611041815445203: A repository names the tools it needs and never where abcd finds them

## Context

Repositories say what they need in files such as `go.mod` and `.tool-versions`, and
a run needs those tools on a machine where PATH often misses them. Finding a copy
from a repository's own files is a trust problem. Asking `go` its version can run or
fetch the toolchain a repository chose, some version files can name a binary to run,
and mise had two trust bypasses in 2026 through settings read from a repository.

Approval is a second trust problem. An issue can trick an agent into asking for a
harmful program, and no filter of that agent's words holds against a determined
attacker. A location accepted automatically is also weak where an agent can write to
a folder on the path and then ask for what it put there.
[The research note](../../research/notes/2026-11-04-repository-tools-on-path-sota.md)
has the sources for both.

adr-2610222222326104 already says a repository's configuration can never start or
widen something on a person's account, and adr-2610241057124151 says what abcd
downloads is never composed from a repository file. The runner's admit rule
(`internal/core/runner/proc.go`) refuses a binary that is relative, inside the
checkout or writable by others. This record applies those to the repository's own
tools. itd-2611041815429986 carries the capability.

Brief invariant 2 says abcd never blocks on an absent tool. That invariant, with
adr-22, is about adapters over abcd's own native defaults, where a missing tool
degrades to the native path. A tool a repository needs for its own build has no
native path to degrade to, so refusing a run on one does not reverse it in its context.
Its closing words read alone do say otherwise, so they now point to invariant 26, a reading the product thinker confirmed at sign-off.

## Decision

We will treat a repository's tool list and version files as data about what it needs,
and take where a copy lives from a person alone.

1. **Read, never run.** abcd reads the list and version files as plain text, through
   the guarded read, from the commit a lane is based on. It does not execute, source or
   pass to a shell any repository file to learn what is needed.
2. **A name and a version, never a binary or a path.** The committed list holds a tool's
   name, why it is needed and which file declares its version, and no path, checksum or
   version of its own. A name follows a strict grammar. Reserved names are refused: a
   shell, `git`, `abcd`, `node` and the command each runner starts. `go` and `make` are not reserved, though abcd runs them, because a repository may need them. A reason has a length limit and a character allowlist with no path separator, `~`, `$`, backtick, quote or control character, so it cannot hand a person a location to type, and it is shown as data. A version
   file comes from a closed set for that tool. One naming a binary or a remote (`path:`,
   `ref:`, a rustup path, a `packageManager` URL, a suffixed Go toolchain name) is refused,
   not followed. A version is held to a strict allowlist, anything else is reported as
   unrecognised, and no version text ever composes a path.
3. **A person says where.** The product thinker ruled on 2026-11-04 that abcd never accepts
   a location by itself. A copy's path is recorded in the person's home, per computer, by a
   person at a terminal, and a scripted yes is not a person's. A copy on the PATH never
   counts by being there, may only be described, and is never offered as a command to paste. There is no table of
   usual places and no capture of a login shell, and no repository file can add a place to
   search. Every candidate passes one judge, extracted from the runner's admit rule and
   given ancestor, owner and file-name checks, and it holds a link and what the link points at
   alike. The home file is read only under the
   ownership test `rules.json` uses.
4. **Approval covers the file.** A registration holds each program's path and SHA-256, and
   any change voids it until a person registers it again. abcd cannot tell an upgrade from a
   swap, so it says the cause is unknown and never runs the changed file. Each is checked at
   the test and again immediately before a command abcd starts.
5. **Asking is not approving.** A lane's agent can stop and say it needs a tool, and its
   change cannot touch the list. A change to the list touches nothing else and needs a
   person's approval of that commit under any facilitator mode, in the merge gate. Approving
   a name runs nothing, since rule 3 still needs a person. No AI reviewer grants a name or a
   location.
6. **Probes are sealed.** Only a tool abcd knows is asked its version, by a fixed
   argument, by absolute path, from a directory outside the repository, with a minimal
   environment, a timeout and each tool's no-download switch set. A listed tool abcd has
   no check for is checked and never run, so a repository cannot make abcd execute a
   program by naming it. A script, or a link to a program of another name as a version
   manager's launcher is, is never probed, because it may re-read the working directory's
   configuration.
7. **Only the chosen programs move, and the claim stops at PATH.** Commands abcd starts
   get a directory made for that command, under the person's abcd home and removed when it
   ends, first on PATH, holding links to the registered programs and nothing else,
   so the person's other tools resolve as before. No other part of a stage's environment
   is changed: abcd does not force no-download switches on a repository's own commands,
   because that would change how the repository works. The report warns about an inherited
   override, and a repository's own commands can still change PATH.
8. **Nothing is downloaded and nothing looks online.** Finding and registering never
   fetch. adr-38 stands: abcd reports a copy against what the project declares from what
   is on disk, and gives the tool's own upgrade command. New versions reach a repository
   as a reviewed change from a forge's update bot. Installing is itd-2611041815433220,
   behind a per-tool yes.
9. **No version manager is a dependency.** abcd runs one only where the person asked it
   to.
10. **A refusal is passed per tool, and only by the person.** A waiver for one tool, one
    repository and one computer is recorded in the person's home, at a terminal. A waived tool
    gets no link and no check. No repository configuration can set it, and no flag replaces it.
11. **This is a filter, and abcd says so.** Under adr-42 none of this is a boundary on the
    person's own account: an agent there can edit PATH or the home file, a pseudo-terminal
    passes for a terminal, the guard's person acts need a host with hooks, and a forge approval by any login with write access counts as a person's, so an agent holding a second such login could approve a list change, though rule 3 still needs a person. The planned fence of itd-2610161621364075 would
    deny those writes for agents abcd launches, and not for agents a host session starts.

## Alternatives Considered

- **A person says where, with the list reviewed as a change (chosen; the product thinker's design and ruling of 2026-11-04).** Least to trust, and it holds with no version manager installed. Every new tool on every computer waits for a person, and each upgrade needs registering again.
- **Accept a copy the person's own recorded path already finds, predating the run, by fixed checks.** Overnight runs keep going. Offered and rejected by the product thinker: a harmful copy already on the path is trusted.
- **An autonomous facilitator approves any tool and records why (the first draft).** No wait for a name. Rejected on review: a repository could turn the mode on for itself, a session cannot prove it is not the lane, a probabilistic check must not make a permanent grant, and a person has to register the tool anyway.
- **An AI reviewer that judges requests from verified facts, with grants that end with the run.** Reduces a person's prompts. Not needed once a person holds every location; the state of the art treats it as no boundary either.
- **A table of usual places.** Nothing for anyone to do, and it would have found Go on the machine that prompted this. Rejected by the product thinker's design: the table needs upkeep, and no one decides that the tool belongs.
- **One command that saves the terminal's folders.** Covers unusual setups with no table. Not taken: it runs all of the person's startup code once, and goes stale.
- **Run the version manager (`mise exec` and its kin).** Least code, and right on machines that have one. Rejected: it records trust implicitly and installs by default, and mise had two trust bypasses in 2026.
- **Run the repository's setup script, as hosted agent products do.** Handles anything. Rejected: those products run it in a machine they manage, while here it would run on the person's account.
- **abcd looks online and updates tools itself.** One place, no setup. Rejected by the product thinker: it reverses adr-38 and adr-2610241057124151, and no product found updates developer tools silently.

## Consequences

- Runs can refuse before spending, on a host session as well as a plain terminal.
- A merged list entry stops a drain on every computer that has not registered the tool, and that drain waits, spending nothing, until a person registers or waives it. An unattended run makes no progress on that repository in the meantime, which the product thinker accepted in choosing this rule. A drive keeps the deadline condition 3 of adr-2610222222326104 sets on every wait, so a drive that reaches it ends, and a later one tests again.
- Every upgrade of a registered tool needs registering again, and a registered path a package manager cleans up stops existing, so registering again can become a habit.
- A host session's agents stay outside rule 7, since abcd cannot set their environment. itd-2611041815429986 says what a start does about that.
- The fingerprint pins bad bytes as faithfully as good ones, and covers only the named file, not what it loads. It is checked before a command starts, and abcd has no daemon, so a swap at any point during that command is not caught, however long it runs. `go` puts its own directory first for its children, and a detached child outlives the PATH directory and falls through to the inherited PATH.
- abcd holds a grammar for each tool's version files, needing upkeep. A declaration it does not recognise is reported, not enforced.
- A stage may still fetch what the person's own setup would fetch, as the note records, and the report says so.
- Updates depend on a forge's bot that someone has to set up.
- Brief invariant 26 carries this. It is a new invariant, not a clause on invariant 2, because invariant 2 is about abcd's own adapters and this is about whose word decides what runs.
