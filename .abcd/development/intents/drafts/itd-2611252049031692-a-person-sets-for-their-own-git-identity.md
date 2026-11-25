---
id: itd-2611252049031692
slug: a-person-sets-for-their-own-git-identity
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610150602547271, itd-2611252117557299, itd-2610240918547728]
severity: critical
refines: [itd-97, itd-2610241015120503, itd-2610301850454860, itd-2610301750162657, iss-2611241109321287]
impact: breaking
related_intents: [itd-2610300955026531, itd-2610300955031405]
related_adrs: [adr-2610150603484802, adr-2609151528057260, adr-2611011332427073, adr-2611251808065076, adr-2610241022541063]
origin: researcher-authored
production_mode: hand-written
---

# A project enforces auto, prevents it, or leaves the facilitator's mode to each person

## Press Release

> Who plays the technical facilitator, a person or abcd, is now decided in one clear order. A project enforces auto for everyone who works in it, prevents it for everyone, or leaves it to each contributor. Jack, a consultant, works in two clients' projects that say nothing. In the first they type one command at their own terminal and abcd is their facilitator there, and every row reads `facilitator: auto (yours, here)`; in the second they set nothing, and their work waits for a person. Their colleague Carol, a tech lead in the first, set nothing either, so their own work still waits, and nothing Jack typed changed that. When the client's maintainers later commit `auto` to the main branch, it is auto for both of them, and the row says that the project decides. abcd never merges a change to that line by itself: another account confirms it, or a maintainer makes it by hand on the forge's page.

## Why This Matters

The product thinker ruled on 2026-11-25 that the technical facilitator's mode is a personal setting, held for each repository for a person's git identity, and that the status line must say which it is (dcn-2611252040395990), and then how it is decided between a project and a person (dcn-2611252132144530). Today one setting speaks for everyone who works in a project: the committed line on its default branch, beside the computer's file. A project that says nothing has no way to say who decides, and a person has no setting for one project alone. The same ruling says what auto means: everything the facilitator is there to do is automated, which is technical decisions, merging reviewed work, and assessing what is delivered against the specs. This gives both, and says which of the three parts auto does today. That the row names the mode is itd-2611252117557299 and comes first. Installing a newer release unasked (dcn-2611252038565989, iss-2611252038564722) goes on reading the person's own mode and never a project's, with adr-2611251808065076, whose third point keeps every repository file out of it.

## Mechanism

We expect a person always to be able to say who decides and why, because there are three values a project can have, one order between a project and a person, and one resolver every reader asks; and we expect the project's value to stay the maintainers', because abcd merges no change to it in any mode. Shown wrong if a contributor who set nothing, in a project that leaves it to each, has a pull request of theirs armed with nobody's approval; if one person's setting decides another's pull request; if a project's value moves by a merge abcd armed; or if two readers give two answers for one reading of the default branch.

## Scope Conditions

- Holds for what a project commits on its default branch and what a person sets on one computer and one operating-system account. On a second computer a person has set nothing until they do it there.
- Holds for the person an identity record names (itd-2610240918547728). The record selects which entry of the account's is read; it is not evidence of who is typing, and two people who share an account share its entries.
- Holds for a repository with a root commit and a name on the forge.
- Holds for what abcd decides and starts: its own merges, the shell commands its guard judges where its hooks run, and its questions. A person merging on the forge's page, and a program that calls the forge itself, are outside it.
- Holds for a person's setting as a check against mistakes, as [adr-42](../../decisions/adrs/0042-guard-parse-layer-is-a-mistake-filter.md) has it for the guard. Rules on the forge are the place for a limit that abcd is not asked to keep.
- Holds for telling an agent from a maintainer only where the two have different forge accounts (itd-2610300955026531, planned later). Where an agent works under the maintainer's own account, the forge sees one account.

## The trust this rests on

Auto lets abcd merge a pull request whose head a recorded review pair passed, with no person asked. So who may set it, for a project and for a person, is stated here, with what holds each and what does not.

**The project's value.** An agent must not be able to change it (dcn-2611252132144530): another account confirms a change an agent makes, or a maintainer makes it by hand on the forge's page. Three things bear on that, and they are not alike.

- *The forge's rules* hold it whatever runs: where a change to `.abcd/config.json` needs the approval of a code owner whose account is not the author's, no pull request changes the line without one. The committed mirror of this project's rules asks for a code owner's review, and names no owner of this file yet (`.abcd/work/rulesets/README.md`). Naming one is work to do, is a person's act on the forge, and is checked by a criterion here; abcd proposes it and never applies it.
- *abcd's own refusals* hold it for what abcd does: abcd arms no pull request that changes the line, in any mode, and its guard refuses the merge command for one where its hooks run. They speak for abcd alone.
- *The plain limit*: where an agent works under the maintainer's own forge account, the forge cannot tell the two apart, so there a rule about accounts asks for a second account and does not tell an agent from its maintainer. Agents with a forge account of their own are planned (itd-2610300955026531; the key touch of itd-2610300955031405 comes after it, for an approval).

**A person's own setting.** Where the project leaves it to each, the terminal and host-session checks on the command, the guard's refusals, and the checks on the file are filters for mistakes. They serve against the ordinary failure, an agent that sets `auto` because a task seemed to need it, and are presented as that and no more. Another operating-system account holds more: an agent run under one has no write to this one's home; abcd does not arrange that. A project that wants a limit its contributors' own settings do not pass commits `human`.

**What binds abcd's own merge paths in every mode.** The review pair's receipt, which every merge abcd arms in `auto` waits for, and the approval a contributor's pull request waits for in either mode. No setting changes them.

**What selects and does not prove.** The identity record, and the time written beside an entry. A word in a chat that an agent relays sets nothing.

**What shows it afterwards.** The mode and whose it is, on every row; the time a person set theirs, in the command's answer; the project's line, in its history; and each merge abcd armed with nobody asked, in this checkout's record of the gate's decisions (itd-2610301750162657).

## What's In Scope

**Three values for a project**, in the key and the spellings there are, on the default branch:

| `facilitator.mode` in the committed file | What it means |
|---|---|
| `auto` | auto is enforced: it is auto for everyone who works in the project |
| `human` | auto is prevented: it is human for everyone |
| absent | each contributor decides for themselves |

**One order.** The project's value decides where it has one. Where it has none: the person's entry for this project; else the computer's file, which becomes the person's default for every project; else human. A person's entry is kept in one file under abcd's home, never in a checkout, for each identity and each project they chose, under the project's root commit and its name on the forge. The computer's file keeps its key and its meaning where a project says nothing. One command sets and shows a person's own, and shows the project's.

**What auto covers.** The three parts, as they stand when this ships:

| The facilitator's work | In auto today | Here | Not here |
|---|---|---|---|
| Merging reviewed work | done: abcd arms a maintainer's pull request whose head a review pair passed | unchanged, but for whose mode it reads and for a change to the project's value | a contributor's pull request, which waits for a maintainer in every mode |
| Technical decisions | in part: each prompt tells the session to decide and record; a drain's judgement is a model's in either mode | a technical question put with the host's question tool is refused and decided | a technical question put in prose (iss-2611251828588773); what a run hands back to the facilitator, which still waits for a person |
| Assessing what is delivered against the specs | in part, in either mode: `abcd build` has an auditor judge every criterion on the lane that ships an intent, and sends it back while one is not met | said plainly on the pages | work shipped outside a build: its audit is owed and listed, and is run when someone asks for it; an audit that failed or was undecided becomes an issue and waits |

The last column's hand-backs and assessments are a second intent, named in the Decomposition.

| Reader | Asks for | The default branch it reads |
|---|---|---|
| The merge gate, when `abcd land` or the build loop would arm | the resolved mode of the person running it, when the gate is asked | asked of origin |
| A withdrawn merge re-armed, or admitted after a release's hold | the same | as last fetched, as it is today |
| The guard's judgement of a merge command | the same | asked of origin |
| The line each prompt carries, and the check on a technical question | the same | as last fetched |
| The drain and the build, on whether a records commit takes a review round | the same, for the person whose account runs them, asked again at each decision | as last fetched; the arming after it asks origin |
| The bare board | the same, with whose it is | as last fetched |
| The row | the same, with whose it is, the person's and the computer's files read at each refresh | kept (itd-2611252117557299) |
| The listing of past landings | the mode the gate recorded for that merge | the line the first parent committed |

## What's Out of Scope

- Finer grants (pushing, opening a pull request, deciding as a role, until a time) are itd-2610240850336695's. That grant and this mode stand side by side, and neither is read as the other.
- Naming the mode on the row and the board, in both modes, is itd-2611252117557299's. The table, the widths and the badges of the row are itd-2611031342000215's, which is a draft.
- Agents with a forge account of their own (itd-2610300955026531), and an approval that needs a key touch (itd-2610300955031405).
- Installing a release: the installing intent's (iss-2611252038564722), which reads the person's own mode and is put to the product thinker with the second and third points of adr-2611251808065076.
- Any change abcd makes to a forge's rules, rulesets or queue, and proving who a person is.

## Acceptance Criteria

- **Given** a default branch whose committed file says `auto`, `human`, or nothing, **when** any person's mode resolves, **then** it is auto for everyone under the first, human for everyone under the second, and the person's own under the third; a person's entry and the computer's file change nothing under the first two, an entry of `human` under an enforced `auto` included. Table-tested over every combination of the project's value, the computer's file, and the person's entry (each `human`, `auto`, or absent), each naming the mode and whose it is.
- **Given** a project that leaves it to each, **when** the mode resolves, **then** the person's entry for this project decides; with none, the computer's file; with neither, it is human, and each landing that arms nothing for this reason says so and names the command. An entry for one project is none for another.
- **Given** a person with no entry, **when** any reader asks over each pairing of the project's value and the computer's file, **then** the answer is the one it was before this intent in all but one: where the project commits `auto` and the computer's file says `human`, it was human and is auto. There the row and the board name the project, and each landing that arms says that the project enforces auto over the person's own human. Where the project says nothing, the computer's file does what it did.
- **Given** a person at their own terminal in a checkout whose identity record names them, **when** they run the command with `auto` or `human`, **then** it writes their entry for this project, or with one flag their default for every project in the computer's file, says the mode now in force and whose it is, has the kept reading the row shows taken again, and writes nothing in the checkout, nothing committed, and nothing on the network. In a project that enforces or prevents auto the entry is written and said to be held, naming the project's value.
- **Given** anyone, an agent included, **when** they run the command bare, **then** it shows the resolved mode, the project's value in the words enforced, prevented, or left to each contributor, the computer's file, the person's entry and when it was set, and writes nothing.
- **Given** a pull request whose head gives `facilitator.mode` another value than the default branch's tip (added, removed, changed, or made unreadable), **when** `abcd land`, the build loop, the drain, or a re-arm reaches it, **then** abcd arms nothing in any mode, whatever approvals, recorded approvals, or receipts there are, takes an arming of its own back off it, and says that the project's value is confirmed by another account, or changed by a maintainer by hand, on the forge's page. A test covers enforced, prevented, and left to each, with and without an approval, and the re-arm.
- **Given** the same pull request, **when** an agent on a host with abcd's hooks runs a command the guard judges as merging, queueing, or arming it, **then** it is refused in any mode, naming why; a pull request whose change to the value cannot be read is refused as a mode that cannot be read is today.
- **Given** a project with a forge, **when** abcd's check of the remote settings runs, **then** it says whether a change to `.abcd/config.json` on the default branch needs an approval from an account other than its author's, by which rule, and whether one account holds every maintainer's access here, so that the rule tells no agent from its maintainer; it names the forge page that sets it, and changes nothing. The person sets it up, and the check is run and shown in this project. **Changes with question 1.**
- **Given** an identity record, for this repository or else for the account (itd-2610240918547728, adr-2610241022541063), that names a person with an entry, in a project that leaves it to each, **when** git would author and commit in the checkout as that person, **then** the entry is read; **when** git would author or commit as another, **then** the mode is human and the reader names the difference. An account that holds an entry for anyone and has no identity record, or one that is refused, reads human there and is told how to record one; an account that holds none reads as the criterion on a person with no entry says. An entry is selected by the record alone, never by git's configuration or a command's environment. Table-tested over the record, the entries, and git's identity.
- **Given** any flag or environment variable on any command of abcd's, **when** the mode resolves, **then** none sets `auto` or selects an entry. The person's file and the computer's file are read from one home, the one the account's own record names; with a `HOME` that is not the account's, every reader in the table reads human where the project leaves it to each, and says that that is why.
- **Given** the command asked to set `auto`, **when** it runs with no terminal, under a host session, or where that could not be told, **then** it is refused with exit 2, nothing is written, and the line says to type it in a terminal of one's own; **when** it is asked to set `human`, **then** only the first two are refused, since a refusal there would hold a person to auto.
- **Given** a host with abcd's hooks, **when** an agent runs the command to set either mode under any program name, or writes, moves, or removes either file that holds a person's own with the shell or the host's own file tools, **then** the guard refuses and says it is the person's to type. The page calls this and the terminal check filters for mistakes, in the words of The trust this rests on. An agent's edit of the project's line on a branch is a proposal and is not refused; the criteria on its pull request hold it.
- **Given** the person's file, **when** it is missing, **then** the person has no entry and nothing is said; **when** it is a link, is owned by or writable by another account, is over its size, is not valid, or holds a value that is neither mode, **then** it is refused naming the file in its tilde form, the mode counts as human, and nothing is armed, as a value that is refused in any layer is today. A file of its name inside a checkout is never read.
- **Given** an entry, **when** the gate resolves it, **then** it is matched on the root commit of the default branch the gate has just fetched and on the repository's name as the forge answers it; a reader that asks no network matches the root of the default branch it reads and the name origin's address spells. A fork, which carries the same root under another name, takes an entry of `auto` from no other repository; an entry of `human` is matched on the root alone and so holds in the fork too. A repository that is renamed keeps its `human` and is asked for its `auto` again.
- **Given** one reading of the default branch, **when** every reader in the table that asks for the mode is asked over it, across the cases of the criteria on the three values and on identity, **then** they give one answer for that reading, through one resolver, and a test that adds a reader of `facilitator.mode` beside that resolver fails. Two readers that hold different readings of the default branch may differ until each has read the same one, and the pages say so.
- **Given** a branch that commits either mode, or a working tree that holds one, **when** the mode resolves in that checkout, **then** it changes nothing: the project's value is the default branch's.
- **Given** a person whose own entry gives `auto`, in a project that leaves it to each, and a ready pull request that another maintainer's account opened, or one whose opener or whose own account cannot be read, **when** the gate is asked, **then** it answers as for human and says whose pull request it is; one their own account opened is armed on the terms `auto` has. Under an enforced `auto`, and where the computer's file gives `auto` as it does today, the ruling of 2026-10-29 stands as it is: a maintainer's pull request goes through in `auto`.
- **Given** a pull request whose author has Read or Triage permission, or none, **when** the gate reaches it under any of the three values and any person's own, **then** it waits for a maintainer's approval of its head, as shipped.
- **Given** the resolved mode, **when** a session is prompted, **then** the line telling it that a technical question is decided and not asked is sent under `auto` and not under `human`, and says whose setting it is; two people's sessions in one project that leaves it to each, one on each mode, each get their own.
- **Given** `auto` in force, **when** a session puts a question marked technical with the host's question tool, **then** the question check refuses it and says to decide it and record it, unless it names an act a command page reserves to the person, or is marked critical; a product question, and every question under `human`, is judged as it is today. Carries the first part of iss-2611241109321287; its other parts stay that issue's.
- **Given** the pages that describe the mode, **when** this ships, **then** they say of each of the three parts of the facilitator's work what auto does and what still waits for a person, in the terms of the table What auto covers, and claim no more.
- **Given** a drain or a build begun under a person's own `auto`, **when** they set `human` while it runs, **then** the next decision it takes reads human and arms nothing more; a merge already armed or queued is not taken back by the command, and the command says so and names the act that does. A drive that is running on the binary it began with, from before this intent, goes on reading the settings that binary knows, and the command names each such drive it finds and how to stop it.
- **Given** a hosted session, or any session with no home of the person's, **when** it reads the mode, **then** it is auto under a project that enforces auto and human otherwise, whatever file its container holds, and the row says which.
- **Given** a merge the gate armed, **when** the past landings are listed, **then** a first parent that committed `auto` or `human` is read as that value; where it committed neither, the mode is the one this checkout's record of that decision holds, and with no such record the row reads `unread`, is counted neither as automatic nor as a person's, and is listed among the unattended, as every row the evidence cannot place with a person is. The spot-check's groups take such a row as their own rule says. Refines itd-2610301850454860 and itd-2610301750162657.
- **Given** any act above, **when** what it writes is read back, **then** nothing this intent writes adds to a commit, branch, pull request, announcement, spot-check record, or receipt the person's identity, the file's entries, or whether it was theirs or another's: a record that names a mode names `human` or `auto` and no more.
- **Given** any case above, **when** the row or the board renders, **then** whose it is stands where itd-2611252117557299 puts it, with one answer more than that intent has: the person's own for this project (`yours, here`). Nothing that intent shipped is withdrawn.

## SOTA

Not searched again for this draft: no source was opened on 2026-11-25, and the independent fit-challenge is owed before the build. What follows is from searches already on record, and from what the writing session knows and did not re-fetch.

- **A project file cannot raise a person's autonomy**: mature, and on record from 2026-10-23 (itd-2610150602547271 and itd-97). One host ignores `auto` in a project's settings and another enables its all-approve mode from the command line only. The ruling departs from it for a project that enforces auto, and that is a difference to weigh in the fit-challenge; the shape is kept where a project leaves it to each.
- **A person's value for a project, kept outside it**: mature, and on record from 2026-11-01 (adr-2611011332427073, which read git, Cargo, pip, mise, VS Code and others).
- **A policy above the person's own setting** (managed or organisation settings that a user cannot override): known to the writing session, not read for this. It is the nearest case to an enforced value.
- **Approval by a named owner of one path** (code owners, required reviewers of a path): known from the forge's documentation read for earlier records, not read again for this. Whether a sole owner's own change passes such a rule is unconfirmed and is checked when it is set up.
- **A person's act told from a program's** (hardware keys, an operating system's own consent dialog, a signed act): not searched here; itd-2610300955031405 holds the key.

Path 2: a native file, resolver and command, and no dependency. The seams are the host's own personal-settings file, the one resolver, and the forge's path rules.

## Open Questions

1. The project's TF==auto is one line in a file on the main branch (`facilitator.mode`), and you ruled that an agent's change to it is confirmed by another account, or that you make it by hand on GitHub's page. Where you and your agents share one GitHub account, GitHub cannot tell which of you made a change, so what holds the line in such a project? **First reading, wait for agents to have their own account.** Until then abcd never auto-merges a change to that line, whoever asks, and you merge it yourself on GitHub's page. What GitHub itself asks for stays as it is. **Second reading, GitHub asks for a second account now.** No change to that line merges without an approval from another account, whatever merges it. You need a second account or another maintainer for every change to it, and your own change by hand may wait for that approval too. It brings back a tracked file that names who approves, which you removed on 2026-10-29 along with naming a person or a team. **Third reading, by the number of maintainers.** Two or more accounts: the second reading. One account: the first. It asks nothing new of you today and changes when somebody joins. What is written is the first with the check, so that no reading is closed.

## Decisions

1. The product thinker's, on 2026-11-25 (dcn-2611252040395990): the mode is a personal setting, held for each repository for the person's git identity; the status line says which it is; it is critical and built without delay. To reverse: keep the committed declaration alone.
2. The product thinker's, on 2026-11-25 (dcn-2611252132144530): auto means everything the facilitator does is automated; a project enforces auto, prevents it, or leaves it to each contributor; what the project sets is above what the person set, an enforced auto over a person's own human included, which reverses in part adr-2610150603484802 (human from any layer wins); an agent must not be able to change the project's value. The session read from it, open to correction: the committed `auto` is enforced and the committed `human` prevented; absence leaves it to each; a contributor who chose nothing is human; a hosted session follows an enforcing project and is human otherwise; a contributor with no write access waits for a maintainer, as shipped. They gave no wording for a criterion. The criteria, the Mechanism, the scope conditions, and The trust this rests on are the writing session's, for their review. To reverse: rule again.
3. The session's (dcn-2611252114265462), whose split stands: the row and the board are itd-2611252117557299, first. Its readings about both sides permitting and about a fallback are replaced by decision 2, and with them the later cut that would have ended the fallback: there is none to end. To reverse: one intent.
4. The session's, for review: no explicit third value is added beside absence. A copy of abcd from before this refuses a value it does not know and then arms nothing, so a third spelling would stop the work of everyone who has not updated; absence means the same to every copy. To reverse: add one once older copies are gone.
5. The session's, for review: the impact is breaking. One pairing changes: a project that commits `auto`, with `human` in a person's computer file, which a shipped criterion of itd-2610150602547271 reads as human and the ruling reads as auto. Everyone else is read as today. To reverse: call it additive.
6. The session's, for review: the computer's file is a person's default for every project and names no identity, so it is the default of everyone on the account, as it is now. Where a project says nothing it does what it did. To reverse: a default for each identity.
7. The session's, for review: a person's own `auto` never arms another maintainer's pull request, since each contributor decides for themselves and the gate cannot read another person's setting. It narrows the ruling of 2026-10-29 in itd-2610150602547271 ("a maintainer's goes through in `auto`") for a person's entry alone, and a person confirms it; an enforced `auto`, and the computer's file where a project says nothing, keep it. To reverse: let the setting of whoever runs the landing decide.
8. The session's, for review: abcd never arms a change to the project's value, a maintainer's with an approval included. It cannot tell an agent's change from a maintainer's, so it leaves every one to the forge's page. To reverse: arm one that a review from another account approves.
9. The session's, for review: an agent sets neither of a person's modes, `human` included, where the command or the guard can tell. To reverse: let an agent set `human`.
10. The session's, for review: nothing public says whose setting a person's was, only which mode. To reverse: name it in the landing's record.
11. The session's, for review: a fork shares a person's entry of `human` and not of `auto`. An entry of `auto` holds under the root commit and the forge's name together, a `human` under the root alone, so a mistake of matching ends at human. To reverse: match both on the root.
12. The session's, for review: one flag writes the default for every project, beside the entries. To reverse: entries only, the default written by hand as today.
13. The session's, for review: an entry is selected by the identity record and not by git's configuration, and a difference between the two reads human. To reverse: select by what git would author as.
14. The session's, for review: with a `HOME` that is not the account's, every reader reads human where a project leaves it to each; a project's value stands above it. To reverse: limit that to an account that holds an entry.
15. The session's, for review: of what auto does not yet do, the check on a technical question is built here, since it is one reader of the mode with a remedy on record, and the rest is a second intent, since it starts agents, spends, and closes work nobody asked it to. To reverse: build all of it here.
16. The session's, no question put: the command's name, the file's name and shape, and the short words in the row are the spec's. It is not `abcd mode facilitator`, which already sets whose answer is owed.

## Decomposition

Hand-run four-piece table, 2026-11-25, run a third time after the ruling on a project and a person (dcn-2611252132144530); a person confirms the routing. One capability here: a project and a person decide the facilitator's mode in one order. One capability beside it, itd-2611252117557299: a person always sees the mode. One capability later, named and not filed: **in auto, abcd does the rest of the facilitator's work** (it assesses delivered work against its specs where no build did, by taking up each audit that is owed and each verdict that failed or was undecided, and it answers what a run hands back to the technical facilitator). It is kept apart because it starts agents and spends with nobody asking, needs a rule on when and how much, and would close work; it refines itd-97, whose composed "when" it is a part of, and takes up itd-83 and itd-165. The cut that was to end a fallback is withdrawn with the fallback. One trust rule, for a new ADR, described here and not minted: a project's value on its default branch is above a person's own and no agent changes it; a person's own, read from their account's home, decides where the project leaves it to each; nothing set is human. It reverses in part adr-2610150603484802, ruled by the product thinker and to be recorded by that ADR: the clause that human from any layer wins, for a project that enforces auto; and the clause that puts the declaration in `.abcd/config.json` alone. What stands of that record: a person is the facilitator until told otherwise. It refines adr-2609151528057260, whose facilitator is a mode a role can be in, by saying who decides the mode and that auto is the whole role. It departs from adr-2611011332427073 (proposed), which lists `facilitator.mode` among the keys a restriction only tightens and says a repository never grants capability: that record takes the exception, or the order, before it is accepted. The brief's invariants gain the rule with the ADR. No new stance. The file, the resolver, the command, and the checks are plumbing, for the spec and the brief's configuration, status line, land, guard, question, drain, and landings chapters. Typed links: `builds_on` itd-2610150602547271, whose gate, guard and element this reads through, itd-2611252117557299, whose row says whose it is, and itd-2610240918547728 with adr-2610241022541063, whose identity record selects the entry; `refines` itd-97, which holds the per-repository declaration as a promise of its own, itd-2610241015120503, whose first and third open questions the ruling answers for this mode, itd-2610301850454860 and itd-2610301750162657, whose listing and groups gain a row that reads `unread`, and iss-2611241109321287, whose first part this carries; `related_intents` itd-2610300955026531 and itd-2610300955031405, which would let the forge and an approval tell an agent from its maintainer. itd-2610240850336695 stands beside it and is neither built on nor changed. Flagged for a person to confirm, never auto-classified: the ruling of 2026-10-29 that decision 7 narrows; the rulings of that day in itd-2610290858115394 and itd-2610150602547271 that no code-owners file and no approval rule is kept, which question 1's second reading would reverse for one path; the rulings of 2026-11-03 in itd-2611031342000215, a draft, whose `auto-merge` badge takes its word from the committed line; and cond-2610161609350397 of itd-2610150602547271 ("One facilitator mode per repository"), which this falsifies where a project leaves it to each. Verdict: SPLIT, two intents filed, one later and unfiled, and the trust rule to its ADR.

## Audit Notes

Review of 2026-11-25, applied to this draft and to itd-2611252117557299 but for these:

- Not applied as worded: the Decision that a fork shares the entry. Matching on the forge's name as well, which the same review asks for, leaves a fork sharing `human` alone (decision 11).
- Overtaken by the ruling (dcn-2611252132144530): the readings of the first question, the cut-over and its migration, the permitting line, and a landing that names it. The committed `auto` is enforced, so none of them is left to say.
- Overtaken by the ruling: a merge with no record reading `unread` whatever the first parent committed. A committed value now decides alone, so only a first parent that committed none leaves anything unread.
- Overtaken by the ruling: a `HOME` that is not the account's reading human for every reader. A project's value is above the person's home, and the rest of it stands (decision 14).
- Corrected: the review's link to the draft that already holds two of the questions is itd-2610241015120503; the id it gave names no record.
- Corrected: the withdrawn merge re-armed and the drain's records round read the default branch as last fetched, as they do today, so the table gives them rows of their own.
