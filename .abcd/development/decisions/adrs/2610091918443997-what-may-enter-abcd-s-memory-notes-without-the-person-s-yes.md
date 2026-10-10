---
id: adr-2610091918443997
slug: what-may-enter-abcd-s-memory-notes-without-the-person-s-yes
status: proposed
date: 2026-10-09
supersedes: null
superseded_by: null
related_intents: [itd-2610091918433290, itd-39, itd-3]
related_rfcs: []
related_adrs: []
---

# ADR-2610091918443997: What may enter abcd's memory notes without the person's yes, and how a note reaches the record

## Context

The product thinker asked on 2026-10-09 (itd-2610091918433290) for agents to keep memory notes through `/abcd:memory`, recalled by the rules loader. Notes are written mostly by agents, often mid-session, so what may be written without the person's yes is a trust question, and so is the step that moves a note from the person's machine into the project's committed record. Constraints already in force: abcd never touches the host's `~/.claude` (the product thinker's ruling of 2026-10-05); the memory-graduates-to-record principle (a lesson every agent should receive belongs in the committed record, never only one person's local memory); the private-names banlist and the scanner's redaction for anything committed; and the loader's ownership and size checks on the machine's rules file.

Open before this record can be decided:

- Which notes an agent may write on its own (for example a fact about the person's machine) and which need the person's yes (for example anything about another person, or anything promoted).
- Whether promotion is the person's act alone, as the source ledger's flip is, or an agent's with disclosure.
- What a note may never hold (secrets, private names, live network identifiers), and which check refuses it.

## Decision

_Not yet decided; settled at the planning interview of itd-2610091918433290._

## Alternatives Considered

_To be laid out at planning: an agent writes freely to the machine layer and promotion needs the person's yes; every write needs the person's yes; an agent writes and promotes, disclosed in the commit._

## Consequences

_To be written with the decision._
