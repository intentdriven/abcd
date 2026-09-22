---
id: itd-2609221656361680
slug: a-repository-abcd-manages-reports-back-to-abcd-itself-one
spec_id: spc-2609221657168936
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-4]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609151838312703, itd-33, itd-116]
related_adrs: [adr-2609221009491186]
---

# A managed repository reports back to abcd, and abcd says so at its next start

## Press Release

> **A repository abcd manages files an enhancement proposal or a defect report against an abcd-issued template, into an inbox in the user account; abcd says at its next start how many wait and from how many repositories.**
>
> "Every useful thing my managed repositories learned about abcd reached me because an agent happened to mention it in a message, and I filed it by hand at midnight," said a product thinker running abcd on three repositories. "Now the repository files it, the report waits in my own account, and abcd greets me with one line saying three are waiting. I read them when I choose; nothing files itself."

## Why This Matters

Every managed repository already has a `for_abcd/` folder in its local tier, and nothing in abcd reads or writes it: on 2026-09-21 three accounts of a sibling repository's runs reached abcd only because an agent relayed them in a message and a person filed the findings by hand at the end of a long day. A report that lives in the managed repository's own tree reaches abcd only if someone carries it; a report in the user account's machine store, where abcd already keeps its history, transcripts, worktrees, sources and labs, is where abcd can find it by itself.

## Mechanism

We expect a reported finding to reach abcd's ledger where a relayed one is lost, because the report outlives the session that noticed it and abcd reads its own inbox without being asked; shown wrong if the inbox fills with reports nobody drains, or if findings keep arriving as chat messages after it ships.

## Scope Conditions

- Holds where the reporting repository and abcd are managed by the same user account on one machine; another account or another machine is the mailbox and broker drafts' question, not this record's. <!-- cond: cond-2609221657163508 -->
- Holds for a report small enough to write at the moment of the finding; a run's whole account is a document the report points at, not the report's body. <!-- cond: cond-2609221657162395 -->

## What's In Scope

- **The template**: abcd issues it (`abcd report --template` writes the skeleton), a machine-readable block (schema version, kind, severity, category, the abcd version and surface in play, a remedy where the reporter has one, evidence pointers) beside prose the reporter writes.
- **The verb**: `abcd report [<file>]` in the managed repository validates the template and files it; with no file it opens the skeleton for the reporter to fill.
- **Where**: `~/.abcd/inbox/<received-stamp>-<sender-key>.md` by default, the machine store keyed as the other stores are; never the managed repository's tree and never abcd's.
- **The greeting**: abcd's session-start line says how many reports wait and from how many repositories, once, and nothing else; the board's own row says the same.
- **The reading**: `abcd inbox` renders them newest first, naming the sender repository plainly, read-only; `abcd inbox show <id>` renders one.
- **The filing**: nothing is filed until a person or a session acts; `abcd inbox promote <id>` files it as a capture carrying the sender's FINGERPRINT (the repository's root-commit key) and a generic description, never its name, scanned before it is written, with the report's id as its evidence pointer.
- **The unknown version**: a report whose template version abcd does not know is listed as unreadable with the version named, and is never dropped or half-read.

## What's Out of Scope

- Cross-account and cross-machine delivery (the mailbox draft itd-2609151838312703 and the broker draft itd-2609151838327688).
- Any automatic filing.
- A reply channel: abcd's answer to a report is the record it files and the release that carries it.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609221009491186 records the vocabulary rulings it rests on):

1. A written account against an abcd-issued template, with a machine-readable block beside the prose (ruled 2026-09-22).
2. It lands in the user account's machine store, not in either repository's tree.
3. abcd says it is there at its next start; nothing is filed until a person or a session acts.
4. Identity in the inbox, fingerprint in the record: the inbox names the sender plainly so the reader knows who is asking; anything committed carries the root-commit key and a generic description.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a managed repository with a finding, **when** `abcd report` runs, **then** the report is validated against the abcd-issued template and filed, with its machine-readable block and its prose, and the verb names where it landed.
- **Given** any report, **when** it is filed, **then** it lands in the machine store's inbox under the user account and nothing is written into either repository's tree.
- **Given** reports waiting, **when** abcd starts a session, **then** one line says how many wait and from how many repositories, and the status board carries the same row.
- **Given** waiting reports, **when** `abcd inbox` runs, **then** they render newest first naming the sender repository, read-only, with nothing filed.
- **Given** a report, **when** nobody acts on it, **then** nothing is filed; **when** `abcd inbox promote` runs on it, **then** a capture exists carrying the sender's root-commit key, a generic description, the report's id as evidence, and no repository name, scanned before the write.
- **Given** a report whose template version abcd does not know, **when** the inbox renders, **then** it is listed as unreadable naming the version, and it is not dropped.
- **Given** the sender's name, **when** anything abcd commits is searched for it, **then** it is absent.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: three managed-repository accounts this week reached abcd only because an agent relayed them and a person filed them by hand; we expect a filed report to reach the ledger where a relayed one is lost; shown wrong if the inbox fills with reports nobody drains or findings keep arriving as chat messages
