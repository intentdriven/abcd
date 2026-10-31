---
id: adr-2610301638558260
slug: a-published-refusal-code-is-never
status: proposed
date: 2026-10-30
supersedes: null
superseded_by: null
related_intents: [itd-2610301428378830, itd-2610301346085246]
related_rfcs: []
related_adrs: [adr-45]
---

# ADR-2610301638558260: A published refusal code is never renamed or reused

## Context

[itd-2610301428378830](../../intents/planned/itd-2610301428378830-when-abcd-refuses-an-agent-s-command-the.md)
gives every refusal abcd makes a code, a one-line hint and the next command
to try, and `abcd explain <code>` prints what the code means. An agent, a
script or a page that matches on a code binds to it the way a script binds to
a flag. If a code is renamed, a matcher that used to work stops matching and
nothing says so. If a code is reused, a matcher keeps matching but now means
something else, which is worse.

The product thinker accepted this cost when they chose refusal codes in the
planning interview of 2026-10-30. The record of that choice, decision 1 of the
intent, reads: "a code that is never renumbered once published". The intent's
open question sent the rule to an ADR, to be written with the spec that
introduces codes
([spc-2610301431108175](../../specs/open/spc-2610301431108175-when-abcd-refuses-an-agent-s-command-the.md)).

abcd is written by several sessions at once, in separate checkouts and on
separate machines. [adr-45](0045-record-ids-are-timestamp-numeric-and.md)
moved record ids off ordinals because two checkouts handed out the same next
number. A counter of refusal codes would collide the same way.

## Decision

- **A refusal code is a short name, not a number:** the verb, a slash and the
  reason, lower-case kebab-case, such as `capture/remedy-missing` or
  `usage/unknown-flag`. Two branches that each add a code pick names from
  what each refusal says, so they collide only when they describe the same
  refusal. The uniqueness test in the merge queue catches that case before
  either branch lands.
- **Once a release has carried a code, the code stays.** It is never renamed
  and never given to a different refusal. A refusal that changes meaning gets
  a new code. A refusal that goes away keeps its code in the registry, marked
  retired, so `abcd explain <code>` still answers ("retired in vX.Y.Z") and
  the name can never be handed out again.
- **The code is the stable part; the words are not.** A refusal's message,
  hint and next command can be reworded in any change. Only the code is a
  promise.
- **A release gate holds the rule.** A committed ledger of codes is written
  from the registry. The release cut compares it with the ledger at the last
  release tag, the way the surface snapshot is compared, and refuses a cut in
  which any code the tag carried is missing.

## Alternatives Considered

- **Numbered codes (`E0001`), as rustc uses.** Compact, and a well-known
  form. Rejected: in this repository a shared counter collides between
  checkouts, which is exactly why adr-45 retired ordinal ids. A number also
  says nothing until it is looked up.
- **Codes free to change until a 1.0 release.** Cheaper while the set
  settles. Rejected: agents use abcd today, and the intent's point is that a
  refused agent recovers in one step. A code that moves under a matcher breaks
  that step without any sign.
- **No codes, only a stable message.** Rejected by the product thinker's
  choice in the interview. A message is reworded for clarity often, and a
  matcher on prose is brittle.
- **Short names, never renamed or reused (chosen).** They are readable,
  collide only on the same meaning, and need no coordination between
  checkouts.

## Consequences

- An agent or script can match on a code and keep working across releases.
  `abcd explain <code>` answers for every code a release ever carried.
- Naming a code is a decision that cannot be undone once a release carries
  it. A badly named code stays, so a code is named for the refusal it
  reports, not for the code path that raises it.
- The registry only grows. Retired entries stay in it for good, and the
  release gate is one more check a cut must pass.
- Before the first release that carries codes, codes can still be renamed
  freely. The rule binds from the first tag whose ledger lists them.
