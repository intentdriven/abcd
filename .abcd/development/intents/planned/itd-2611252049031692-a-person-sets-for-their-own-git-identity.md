---
id: itd-2611252049031692
slug: a-person-sets-for-their-own-git-identity
spec_id: spc-2611262159212693
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610150602547271, itd-2611252117557299]
severity: critical
refines: [itd-97, itd-2610241015120503, itd-2610301850454860, itd-2610301750162657, iss-2611241109321287]
impact: breaking
related_intents: [itd-2610300955026531, itd-2610300955031405]
related_adrs: [adr-2611262044127943, adr-2610150603484802, adr-2609151528057260, adr-2611011332427073, adr-2611251808065076]
origin: researcher-authored
production_mode: hand-written
---

# A project enforces auto, prevents it, or leaves the facilitator's mode to each person

## Press Release

> Who plays the technical facilitator, a person or abcd, is now decided in two places and one clear order. A project enforces auto for everyone who works in it, prevents it for everyone, or says nothing and leaves it to each contributor's own setting. Jack, a consultant, works in two clients' projects that say nothing. They type one command at their own terminal, once, and abcd is their facilitator in both, and every row reads `facilitator: auto (computer)`. Their colleague Carol, a tech lead in the first, set nothing, so her own work still waits for a person, and nothing Jack typed changed that. When the second client's maintainers commit `human` to their main branch, Jack's work there waits for a person again and the row says the project decides; when the first client's commit `auto`, it is auto for Carol too. abcd never arms or merges a change to that line, in any mode; a maintainer merges it on the forge's page.

## Why This Matters

The product thinker ruled on 2026-11-25 that the technical facilitator's mode is a personal setting, then read as held for each repository for a person's git identity, and that the status line must say which it is (dcn-2611252040395990), and then how it is decided between a project and a person (dcn-2611252132144530). Today one setting speaks for everyone who works in a project: the committed line on its default branch, beside the computer's file. A project that says nothing has no way to say who decides, and a person has no setting for one project alone. The same ruling says what auto means: everything the facilitator is there to do is automated, which is technical decisions, merging reviewed work, and assessing what is delivered against the specs. On 2026-11-26 the product thinker made it simpler: two places decide, the project's committed value and otherwise the person's own global setting, and there is no setting of a person's for one repository ("simpler; if we ever need a third option, we can revisit"). This gives both, and says which of the three parts auto does today. That the row names the mode is itd-2611252117557299 and comes first. Installing a newer release unasked (dcn-2611252038565989, iss-2611252038564722) will read the person's own mode and never a project's, with adr-2611251808065076, whose third point keeps every repository file out of it.

## Mechanism

We expect a person always to be able to say who decides and why, because there are three values a project can have, one order between a project and a person, and one resolver every reader asks; and we expect no change to the project's value to be one that abcd merged, because it arms no change to it in any mode. Shown wrong if a contributor who set nothing, in a project that leaves it to each, has a pull request of theirs armed with nobody's approval; if one person's setting decides another's pull request; if a project's value moves by a merge abcd armed; or if two readers give two answers for one reading of the default branch.

## Scope Conditions

- Holds for what a project commits on its default branch and what a person sets on one computer and one operating-system account. On a second computer a person has set nothing until they do it there, and two people who share an account share its setting. <!-- cond: cond-2611262159211564 -->
- Holds for what abcd decides and starts: its own merges, the shell commands its guard judges where its hooks run, and its questions. A person merging on the forge's page, and a program that calls the forge itself, are outside it. <!-- cond: cond-2611262159212071 -->
- Holds for a person's setting as a check against mistakes, as [adr-42](../../decisions/adrs/0042-guard-parse-layer-is-a-mistake-filter.md) has it for the guard. Rules on the forge are the place for a limit that abcd is not asked to keep. <!-- cond: cond-2611262159219957 -->
- Holds for telling an agent from a maintainer only where the two have different forge accounts (itd-2610300955026531, a draft). Where an agent works under the maintainer's own account, the forge sees one account. <!-- cond: cond-2611262159210266 -->

## The trust this rests on

Auto lets abcd merge a pull request whose head a recorded review pair passed, with no person asked. So who may set it, for a project and for a person, is stated here, with what holds each and what does not.

**The project's value.** An agent must not be able to change it (dcn-2611252132144530): another account confirms a change an agent makes, or a maintainer makes it by hand on the forge's page. Where the two share one forge account, abcd arms none and the person merges it on the forge's page (dcn-2611252138203673). Three things bear on that, and they are not alike.

- *The forge's rules* hold it whatever runs: where a change to `.abcd/config.json` needs the approval of a code owner whose account is not the author's, no pull request changes the line without one. What the forge asks is left as it is (dcn-2611252138203673); setting it up is the person's act, and abcd's check of the remote settings says what it asks.
- *abcd's own refusals* hold it for what abcd does: abcd arms no pull request that changes the line, in any mode, and its guard refuses the merge command for one where its hooks run. They speak for abcd alone.
- *The plain limit*: where an agent works under the maintainer's own forge account, the forge cannot tell the two apart, so there a rule about accounts asks for a second account and does not tell an agent from its maintainer. Agents with a forge account of their own are a draft (itd-2610300955026531), and so is a key touch for an approval, which comes after it (itd-2610300955031405).

**A person's own setting.** Where the project leaves it to each, the person's global setting decides. The terminal and host-session checks on the command, the guard's refusals, and the checks on the file are filters for mistakes. They serve against the ordinary failure, an agent that sets `auto` because a task seemed to need it, and are presented as that and no more. Another operating-system account holds more: an agent run under one has no write to this one's home; abcd does not arrange that. A project that wants a limit its contributors' own settings do not pass commits `human`.

**What binds abcd's own merge paths in every mode.** The review pair's receipt, which every merge abcd arms in `auto` waits for, and the approval a contributor's pull request waits for in either mode. No setting changes them.

**What does not prove.** A word in a chat that an agent relays sets nothing.

**What shows it afterwards.** The mode and whose it is, on every row; the project's line, in its history; and each merge abcd armed with nobody asked, in this checkout's record of the gate's decisions (itd-2610301750162657).

## What's In Scope

**Three values for a project**, in the key and the spellings there are, on the default branch:

| `facilitator.mode` in the committed file | What it means |
|---|---|
| `auto` | auto is enforced: it is auto for everyone who works in the project |
| `human` | auto is prevented: it is human for everyone |
| absent | each contributor decides for themselves |

**One order, two places** (ruled 2026-11-26, [adr-2611262044127943](../../decisions/adrs/2611262044127943-the-project-s-committed-facilitator-mode.md)). The project's value decides where it has one. Where it has none, the person's global setting, `facilitator.mode` in `~/.abcd.noindex/config.json`, which the row names `computer` as it does today; else human. There is no setting of a person's for one project. The global file keeps its key and its meaning where a project says nothing. One command sets and shows a person's own, and shows the project's; it writes only the global file and takes no flag that chooses where.

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
| A withdrawn merge re-armed, or admitted after a release's hold | the same | asked of origin |
| The guard's judgement of a merge command | the same | asked of origin |
| The line each prompt carries, and the check on a technical question | the same | as last fetched |
| The drain and the build, on whether a records commit takes a review round | the same, for the person whose account runs them, asked again at each decision | as last fetched; the arming after it asks origin |
| The bare board | the same, with whose it is | as last fetched |
| The row | the same, with whose it is, the person's global file read at each refresh | kept (itd-2611252117557299) |
| The listing of past landings | the mode the gate recorded for that merge | the line the first parent committed |

## What's Out of Scope

- Finer grants (pushing, opening a pull request, deciding as a role, until a time) are itd-2610240850336695's. That grant and this mode stand side by side, and neither is read as the other.
- Naming the mode on the row and the board, in both modes, is itd-2611252117557299's. The table, the widths and the badges of the row are itd-2611031342000215's, which is a draft.
- Agents with a forge account of their own (itd-2610300955026531), and an approval that needs a key touch (itd-2610300955031405).
- Installing a release: the installing intent's (iss-2611252038564722), which reads the person's own mode and is put to the product thinker with the second and third points of adr-2611251808065076.
- Any change abcd makes to a forge's rules, rulesets or queue, and proving who a person is.

## Acceptance Criteria

- **Given** a default branch whose committed file says `auto`, `human`, or nothing, **when** any person's mode resolves, **then** it is auto for everyone under the first, human for everyone under the second, and the person's own global setting under the third; the global setting changes nothing under the first two, a `human` under an enforced `auto` included. Table-tested over every combination of the project's value and the global setting (each `human`, `auto`, or absent), each naming the mode and whose it is.
- **Given** a project that leaves it to each, **when** the mode resolves, **then** the person's global setting decides; with none, it is human, and each landing that arms nothing for this reason says so and names the command. A file of entries for one project in abcd's home, as an earlier build of this intent kept, decides nothing for any reader.
- **Given** any person, **when** any reader asks over each pairing of the project's value and the global setting, **then** the answer is the one it was before this intent in all but these: a project that commits `auto`, with `human` in the computer's file or with a home abcd refuses, was human and is auto; a project that says nothing, with `auto` in the computer's file, is human under a `HOME` other than the account's and in a hosted session; and a past landing whose first parent committed no mode is listed as the criterion on past landings says. In the first, the row and the board name the project, and each landing that arms says that the project enforces auto over the person's own human. Table-tested, the exceptions by name.
- **Given** a person at their own terminal, **when** they run the command with `auto` or `human`, **then** it writes `facilitator.mode` in their global file, `~/.abcd.noindex/config.json`, keeping every other setting there, says the mode now in force and whose it is, has the kept reading the row shows taken again, and writes nothing in the checkout, nothing committed, and nothing on the network. It takes no flag that chooses where. In a project that enforces or prevents auto the setting is written and said to be held, naming the project's value.
- **Given** anyone, an agent included, **when** they run the command bare, **then** it shows the resolved mode, the project's value in the words enforced, prevented, or left to each contributor, and the global setting, and writes nothing.
- **Given** a pull request whose merge into the default branch's tip would change `facilitator.mode` there (add, remove, or alter it, or make it unreadable), **when** `abcd land`, the build loop, the drain, or a re-arm reaches it, **then** abcd arms nothing in any mode, whatever approvals, recorded approvals, or receipts there are, takes an arming of its own back off it, and says that a maintainer merges it on the forge's page. A test covers enforced, prevented, and left to each, with and without an approval, a branch behind the tip, and the re-arm.
- **Given** the same pull request, **when** an agent on a host with abcd's hooks runs a command the guard judges as merging, queueing, or arming it, **then** it is refused in any mode, naming why; a pull request of which that cannot be read is refused in the same way.
- **Given** a project with a forge, **when** abcd's check of the remote settings runs, **then** it says whether a change to `.abcd/config.json` on the default branch needs an approval from an account other than its author's, and by which rule, names the forge page that sets it, and changes nothing. Setting it up is the person's act.
- **Given** any flag or environment variable on any command of abcd's, **when** the mode resolves, **then** none sets `auto`. The global setting is the one in the home the account database names for the account, whatever `HOME` holds; under a `HOME` other than that one a person has no setting, which is human where the project leaves it to each. Tested for every reader in the table.
- **Given** the command asked to set `auto`, **when** it runs with no terminal, under a host session, or where that could not be told, **then** it is refused with exit 2, nothing is written, and the line says to type it in a terminal of one's own; **when** it is asked to set `human`, **then** only the first two are refused, since a refusal there would hold a person to auto.
- **Given** a host with abcd's hooks, **when** an agent runs the command to set either mode under the names the guard knows for abcd, or writes, moves, or removes the global settings file with the shell or the host's own file tools, **then** the guard refuses and says it is the person's to type. The page calls this and the terminal check filters for mistakes, in the words of The trust this rests on. An agent's edit of the project's line on a branch is a proposal and is not refused; the criteria on its pull request hold it.
- **Given** the global file, **when** it is missing, **then** the person has no setting and nothing is said; **when** the home that holds it is refused, **then** where the project leaves it to each the mode counts as human and nothing is armed, and where the project has a value, that value stands. A value the global file holds that is neither mode is refused in every project, as today. A file of its name inside a checkout is never read as it.
- **Given** one reading of the default branch, **when** every reader in the table that asks for the mode is asked over it, across the cases of the criteria on the three values, **then** they give one answer for that reading, through one resolver, and a test that adds a reader of `facilitator.mode` beside that resolver fails. Two readers that hold different readings may differ until each has read the same one, and the pages say so.
- **Given** a branch that commits either mode, or a working tree that holds one, **when** the mode resolves in that checkout, **then** it changes nothing: the project's value is the default branch's.
- **Given** a person whose global setting gives `auto`, in a project that leaves it to each, and a ready pull request that another maintainer's account opened, or one whose opener or whose own account cannot be read, **when** the gate is asked, **then** it answers as for human and says whose pull request it is; one their own account opened is armed on the terms `auto` has. Under an enforced `auto` the ruling of 2026-10-29 stands as it is: a maintainer's pull request goes through in `auto`. **Narrows the ruling of 2026-10-29 for a person's own setting; question 2, answered "only your own" on 2026-11-26.** Built in the gate (decision 21); the guard's half is spc-2611262159212693 step 1.
- **Given** a pull request whose author has Read or Triage permission, or none, **when** the gate reaches it under any of the three values and any person's own, **then** it waits for a maintainer's approval of its head, as shipped.
- **Given** the resolved mode, **when** a session is prompted, **then** the line telling it that a technical question is decided and not asked is sent under `auto` and not under `human`, and says whose setting it is; two people's sessions in one project that leaves it to each, one on each mode, each get their own.
- **Given** `auto` in force, **when** a session puts a question marked technical with the host's question tool, **then** the question check refuses it and says to decide it and record it, unless it names an act a command page reserves to the person; a product question, and every question under `human`, is judged as it is today. The mark by which the most critical decisions still reach a person (dcn-2611241109230165) is iss-2611241109321287's to name, and this check is not built before it is. Carries the first part of that issue; its other parts stay its own.
- **Given** the pages that describe the mode, **when** this ships, **then** they say of each of the three parts of the facilitator's work what auto does and what still waits for a person, in the terms of the table What auto covers, and claim no more.
- **Given** a drain or a build begun under a person's own `auto`, **when** they set `human` while it runs, **then** the next decision it takes reads human and arms nothing more; a merge already armed or queued is not taken back by the command, and the command says so and names the act that does. A drive that is running on the binary it began with, from before this intent, goes on reading the settings that binary knows, and the command names each such drive it finds and how to stop it.
- **Given** a hosted session, or any session with no home of the person's, **when** it reads the mode, **then** it is auto under a project that enforces auto and human otherwise, whatever file its container holds, and the row says which.
- **Given** a merge the gate armed, **when** the past landings are listed, **then** a first parent that committed `auto` or `human` is read as that value; where it committed neither, the mode is the one this checkout's record of that decision holds, and with no such record the row reads `unrecorded` (not `forge unread`, which is a forge that did not answer, nor `unreadable`, which is a committed file that cannot be read), is counted neither as automatic nor as a person's, and is listed among the unattended, as every row the evidence cannot place with a person is. The spot-check's groups take such a row as their own rule says. Refines itd-2610301850454860 and itd-2610301750162657.
- **Given** any act above, **when** what it writes is read back, **then** nothing this intent writes adds to a commit, branch, pull request, announcement, spot-check record, or receipt the person's identity, their settings file, or whether it was theirs or another's: a record that names a mode names `human` or `auto` and no more. The gate's reason may name whose setting it was where it is printed; the reason the spot-check keeps is cut to the mode.
- **Given** any case above, **when** the row or the board renders, **then** whose it is stands where itd-2611252117557299 puts it, the person's global setting under the word that intent gave the computer's file (`computer`), with one answer more for a `HOME` that is not the account's own (`not your home`). Nothing that intent shipped is withdrawn: under a refused home the row still names the mode the resolver gives and says that the computer's file could not be read.

## SOTA

Not searched again for this draft: no source was opened on 2026-11-25, and the independent fit-challenge is owed before the build. What follows is from searches already on record, and from what the writing session knows and did not re-fetch.

- **A project file cannot raise a person's autonomy**: mature, and on record from 2026-10-23 (itd-2610150602547271 and itd-97). One host ignores `auto` in a project's settings and another enables its all-approve mode from the command line only. The ruling departs from it for a project that enforces auto, and that is a difference to weigh in the fit-challenge; the shape is kept where a project leaves it to each.
- **A person's value for a project, kept outside it**: mature, and on record from 2026-11-01 (adr-2611011332427073, which read git, Cargo, pip, mise, VS Code and others).
- **A policy above the person's own setting** (managed or organisation settings that a user cannot override): known to the writing session, not read for this. It is the nearest case to an enforced value.
- **Approval by a named owner of one path** (code owners, required reviewers of a path): known from the forge's documentation read for earlier records, not read again for this. Whether a sole owner's own change passes such a rule is unconfirmed and is checked when it is set up.
- **A person's act told from a program's** (hardware keys, an operating system's own consent dialog, a signed act): not searched here; itd-2610300955031405 holds the key.

Path 2: a native file, resolver and command, and no dependency. The seams are the host's own personal-settings file, the one resolver, and the forge's path rules.

## Open Questions

1. resolved: What holds the line on the project's TF==auto (`facilitator.mode` in a file on the main branch) where you and your agents share one GitHub account? Answered by the session while the product thinker is away (dcn-2611252138203673): abcd never auto-merges a change to that line, you merge it yourself on GitHub's page, and what GitHub asks is left as it is. The readings it was chosen over: a second account's approval at GitHub, now or wherever two accounts maintain the project. It is theirs to rule otherwise.
2. resolved: In a project that leaves it to each contributor, does your own TF==auto auto-merge only pull requests you opened, or also a colleague's that you land? Answered by the product thinker on 2026-11-26: **only your own**. Your setting never decides a colleague's work, and a colleague's waits for an approval or for them to land it. The reading it was chosen over: a colleague's too, whoever lands deciding. The gate holds it (decision 21).

## Decisions

1. The product thinker's, on 2026-11-25 (dcn-2611252040395990): the mode is a personal setting; the status line says which it is; it is critical and built without delay. Its reading as held for each repository for the person's git identity is overtaken by decision 17. To reverse: keep the committed declaration alone.
2. The product thinker's, on 2026-11-25 (dcn-2611252132144530): auto means everything the facilitator does is automated; a project enforces auto, prevents it, or leaves it to each contributor; what the project sets is above what the person set, an enforced auto over a person's own human included, which reverses in part adr-2610150603484802 (human from any layer wins); an agent must not be able to change the project's value. The session read from it, open to correction: the committed `auto` is enforced and the committed `human` prevented; absence leaves it to each; a contributor who chose nothing is human; a hosted session follows an enforcing project and is human otherwise; a contributor with no write access waits for a maintainer, as shipped. They gave no wording for a criterion. The criteria, the Mechanism, the scope conditions, and The trust this rests on are the writing session's, for their review. To reverse: rule again.
3. The session's (dcn-2611252114265462), whose split stands: the row and the board are itd-2611252117557299, first. Its readings about both sides permitting and about a fallback are replaced by decision 2, and with them the later cut that would have ended the fallback: there is none to end. To reverse: one intent.
4. The session's, for review: no explicit third value is added beside absence. A copy of abcd from before this refuses a value it does not know and then arms nothing, so a third spelling would stop the work of everyone who has not updated; absence means the same to every copy. To reverse: add one once older copies are gone.
5. The session's, for review: the impact is breaking. What changes for a person: a project that commits `auto` is auto over `human` in the computer's file, which a shipped criterion of itd-2610150602547271 reads as human, and over a home abcd refuses; a project that says nothing is human, whatever the computer's file says, under a `HOME` other than the account's and in a hosted session; a past landing whose first parent committed no mode may read `unrecorded`; and a pull request that would change the project's value is armed no more. Everyone else is read as today. To reverse: call it additive.
6. The session's, for review: the computer's file is the person's global setting and names no identity, so it is the setting of everyone on the account, as it is now. Where a project says nothing it does what it did. To reverse: a setting for each identity.
7. Ruled by the product thinker on 2026-11-26 (question 2): a person's own `auto` never arms another maintainer's pull request, since each contributor decides for themselves and the gate cannot read another person's setting. It narrows the ruling of 2026-10-29 in itd-2610150602547271 ("a maintainer's goes through in `auto`") for a person's own setting; an enforced `auto` keeps that ruling. To reverse: let the setting of whoever runs the landing decide.
8. The session's, while the product thinker is away (dcn-2611252138203673): abcd never arms a change to the project's value, a maintainer's with an approval included. It cannot tell an agent's change from a maintainer's, so it leaves every one to the forge's page. To reverse: arm one that a review from another account approves.
9. The session's, for review: an agent sets neither of a person's modes, `human` included, where the command or the guard can tell. To reverse: let an agent set `human`.
10. The session's, for review: nothing public says whose setting a person's was, only which mode. To reverse: name it in the landing's record.
11. Withdrawn with the per-project entry (decision 17): how a fork shared a person's entry.
12. Replaced by decision 18: one flag wrote the default for every project, beside the entries.
13. Withdrawn with the per-project entry (decision 17): how the identity record selected an entry.
14. The session's, for review: a person's setting is the one in the home the account database names; under a `HOME` other than that one a person has none, and a project's value stands (dcn-2611260341316692). To reverse: read the account's own whatever `HOME` is.
15. The session's, for review: of what auto does not yet do, the check on a technical question is built here, since it is one reader of the mode with a remedy on record, and the rest is a second intent, since it starts agents, spends, and closes work nobody asked it to. To reverse: build all of it here.
16. The session's, no question put: the command's name and the short words in the row are the spec's. It is not `abcd mode facilitator`, which already sets whose answer is owed.
17. The product thinker's, on 2026-11-26 ([adr-2611262044127943](../../decisions/adrs/2611262044127943-the-project-s-committed-facilitator-mode.md)): both clauses of adr-2610150603484802 that decision 2 touches are reversed, that `human` from any layer wins over `auto` and that the setting lives only in `.abcd/config.json`. Two places only decide the mode: what the repository enforces, committed on its default branch (`auto` enforced or `auto` prevented), which beats everything; otherwise the person's global setting in `~/.abcd.noindex/config.json`. Nothing set anywhere is a person. There is no setting of a person's for one repository: "simpler; if we ever need a third option, we can revisit". The same day they confirmed dcn-2611252138203673 (decision 8), answered question 2 (decision 7), and signed the plan off, so the intent is planned. To reverse: rule again.
18. The session's, on the coordinator's direction after decision 17: the command writes only the global file and takes no flag that chooses where, so there is one way to set it (dcn-2611260505138989). To reverse: add a flag once a second place exists.
19. Superseded by decision 21. The session's (dcn-2611262048349309): until the gate reads who opened a pull request, it keeps the shipped behaviour under the global `auto`, where a maintainer's pull request goes through once a review pair passes its head; that read is a step still to build. To reverse: hold the global `auto` as human at the gate until the read lands.
20. The session's (dcn-2611262048355464): the global setting keeps the words that shipped for the computer's file, `computer` on the row and `this computer's` on the board and in the reasons. To reverse: rename them to `yours` in one change.
21. The session's, on the coordinator's direction to build the product thinker's "only your own" (dcn-2611262202295309, superseding decision 19): where `auto` is the person's global setting, the gate asks the forge which account its client runs as and arms only a pull request that account opened; anyone else's, or one where that account cannot be read, is judged as `human` judges it, so an approval of its head arms it with no review receipt asked for. Under an enforced `auto` nothing changes and the account is not asked. To reverse: drop the account read, and the gate goes back to decision 19.
22. The session's (dcn-2611262202306256): `abcd land`, the build loop, the drain and a re-arm arm nothing, in any mode, for a pull request that changes the committed `facilitator.mode` line or leaves it unreadable, read at its head against each of its merge bases with the default branch's tip and in the merge of its head into that tip, and say the person merges it on the forge's page. The guard's refusal of the merge command, and taking an arming of abcd's own back off such a pull request, are larger than the change that built the gate's half and are spc-2611262159212693's steps 1 and 2. To reverse: build them, or drop the gate's check.

## Decomposition

Hand-run four-piece table, 2026-11-25, run a third time after the ruling on a project and a person (dcn-2611252132144530); a person confirms the routing. One capability here: a project and a person decide the facilitator's mode in one order. One capability beside it, itd-2611252117557299: a person always sees the mode. One capability later, on record only here and to be filed by the run before this intent ships: **in auto, abcd does the rest of the facilitator's work** (it assesses delivered work against its specs where no build did, by taking up each audit that is owed and each verdict that failed or was undecided, and it answers what a run hands back to the technical facilitator). It is kept apart because it starts agents and spends with nobody asking, needs a rule on when and how much, and would close work; it refines itd-97, whose composed "when" it is a part of, and takes up itd-83 and itd-165. The cut that was to end a fallback is withdrawn with the fallback. One trust rule, minted as [adr-2611262044127943](../../decisions/adrs/2611262044127943-the-project-s-committed-facilitator-mode.md) on the product thinker's ruling of 2026-11-26: a project's value on its default branch is above a person's own and no agent changes it; a person's global setting, read from their account's home, decides where the project leaves it to each; nothing set is human. It reverses in part adr-2610150603484802: the clause that human from any layer wins, for a project that enforces auto; and the clause that puts the declaration in `.abcd/config.json` alone. What stands of that record: a person is the facilitator until told otherwise. It refines adr-2609151528057260, whose facilitator is a mode a role can be in, by saying who decides the mode and that auto is the whole role. It departs from adr-2611011332427073 (proposed), which lists `facilitator.mode` among the keys a restriction only tightens and says a repository never grants capability: that record takes an exception before it is accepted (dcn-2611252138203673). The brief's invariants gain the rule with the ADR. No new stance. The resolver, the command, and the checks are plumbing, for the spec and the brief's configuration, status line, land, guard, question, drain, and landings chapters. Typed links: `builds_on` itd-2610150602547271, whose gate, guard and element this reads through, and itd-2611252117557299, whose row says whose it is; `refines` itd-97, which holds the per-repository declaration as a promise of its own, itd-2610241015120503, whose first open question the ruling answers for this mode and whose third, on a cloud session, the session's reading of it answers, open to correction, itd-2610301850454860 and itd-2610301750162657, whose listing and groups gain a row that reads `unrecorded`, and iss-2611241109321287, whose first part this carries; `related_intents` itd-2610300955026531 and itd-2610300955031405, which would let the forge and an approval tell an agent from its maintainer. itd-2610240850336695 stands beside it and is neither built on nor changed. Flagged for a person to confirm, never auto-classified: the ruling of 2026-10-29 that decision 7 narrows; the rulings of that day in itd-2610290858115394 and itd-2610150602547271 that no code-owners file and no approval rule is kept, which dcn-2611252138203673 leaves standing; the rulings of 2026-11-03 in itd-2611031342000215, a draft, whose `auto-merge` badge takes its word from the committed line; and cond-2610161609350397 of itd-2610150602547271 ("One facilitator mode per repository"), which this falsifies where a project leaves it to each. Verdict: SPLIT, two intents filed, one later and unfiled, and the trust rule to its ADR.

## Audit Notes

Review of 2026-11-25, applied to this draft and to itd-2611252117557299 but for these:

- Not applied as worded: the Decision that a fork shares the entry. Matching on the forge's name as well, which the same review asks for, left a fork sharing `human` alone (decision 11); both were withdrawn with the per-project entry on 2026-11-26.
- Overtaken by the ruling (dcn-2611252132144530): the readings of the first question, the cut-over and its migration, the permitting line, and a landing that names it. The committed `auto` is enforced, so none of them is left to say.
- Overtaken by the ruling: a merge with no record left unplaced whatever the first parent committed. A committed value now decides alone.
- Overtaken by the ruling: every reader human under another `HOME`. A project's value stands above a person's settings (decision 14).
- Corrected: the review's link to the draft that already holds two of the questions is itd-2610241015120503; the id it gave names no record.

Second review of 2026-11-25, finding by finding:

- MUST, what abcd holds of the project's line: taken, in the Press Release and the Mechanism.
- MUST, the list of what changes: taken, in the criterion and decision 5, with a refused person's file under a project that has a value.
- SHOULD, dcn-2611252138203673: taken; question 1 is resolved by it, and the criterion on the remote settings no longer changes.
- SHOULD, another maintainer's pull request: taken, as question 2, deferred, with the criterion marked.
- SHOULD, a merge that would change the line: taken.
- SHOULD, "marked critical": taken; left to iss-2611241109321287, and the check waits for it.
- SHOULD, the names the guard knows: taken.
- SHOULD, another word than `unread`: taken, `unrecorded`.
- SHOULD, the spot-check's reason: taken, cut to the mode.
- SHOULD, what selects an entry: taken, kept with the reading; withdrawn with the per-project entry on 2026-11-26.
- SHOULD, the row under a refused home: taken.
- SHOULD, "will read", and the draft status of the forge account: taken.
- SHOULD, the third question of itd-2610241015120503: taken, as the session's reading.
- SHOULD, the later capability: taken, as to be filed by the run.
- SHOULD, the reader table's re-arm row, and the home: taken; the re-arm asks origin, and the home is stated as a property.
