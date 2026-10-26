---
id: adr-2610222026265719
slug: a-session-s-checklist-names-a-credential
status: accepted
date: 2026-10-22
supersedes: null
superseded_by: null
related_intents: [itd-2610170638184486, itd-2610170634394993, itd-2610180912189125]
related_rfcs: []
related_adrs: [adr-2610222026262980]
---

# ADR-2610222026265719: A session's checklist names a credential by its kind, scope and expiry, never its value or anything that retrieves it

## Context

The checklist intent (itd-2610170638184486) gives every abcd session a file
on disk saying what it may decide, what it waits on and what it owns. Its
pre-run blocker sweep asks each session for a credential that is missing or
will expire before the person returns, and that answer is written into the
checklist. The coordination intent (itd-2610170634394993) gives every
autonomous session a short-lived token of its own from abcd's GitHub App
(its decision of 2026-10-22, C3), so a session knows of a credential that
matters to its night.

The checklist is the most-read file a session has. It is re-injected into
the model's context at every start and after every compaction, written by
the agent, read back by any later session in that working copy, summarised
into the person's return lists, and, in later work, uploaded to a private
shared copy and optionally published (itd-2610222029228127, split out on
2026-10-22 by dcn-2610222025212338). Anything written there is copied into every one of those places.
Agent-written state is also the least careful writer in the system: a model
told to record "the token that expires at 03:00" will as readily paste the
token.

On 2026-10-18 the agent routing the draft's decomposition proposed this as a
separate trust rule; on 2026-10-22 the product thinker ruled that it be
written up as one (dcn-2610222025224993). Through the decomposition table a
trust rule becomes an ADR and a clause of the brief's invariants. The rule
itself, the first two paragraphs of the decision below, is the product
thinker's; how it is enforced, the third, is the technical facilitator's,
decided on their delegation and open to change at the plan's sign-off.

## Decision

**A checklist records a credential only by its kind, its scope and its
expiry.** Kind is what sort of credential it is (an App installation token,
a model-service key, a signing key); scope is what it reaches (a repository
and the acts allowed, a service); expiry is when it stops working, or that it
has none. Nothing else about it is written.

**Never its value, and never anything that retrieves it.** A checklist holds
no secret, no part of one, no hash or encoding of one, and no pointer that
would hand one to whoever reads the file: not an environment variable's
value, not a path to a key file, not a keychain item's name and account, not
a command that prints a token. The person, or the coordination intent's
issuing step, gets the credential; the checklist only says one is needed and
by when.

**The write refuses, and the read refuses.** abcd's checklist writer checks
every field it writes with abcd's native secret scanner
(`internal/adapter/scanner`), the one capture and decide already run, and refuses a write that carries a secret
pattern, naming the field and the pattern kind and never echoing the match;
the session's last good checklist stays in place. A checklist read from disk
that fails the same check is not re-injected: the reader reports the field
and falls back to the last good copy, as it does for a file it cannot parse;
the last good copy is only ever a file a checked write produced, and it is
checked again on read, so a file that fails both is not re-injected at all.
The check is a pattern scan, so it catches the shapes it knows; the rule
binds whatever it misses, and the fields that name a credential are typed
(kind, scope and expiry, from fixed lists and a timestamp), never free text.

## Alternatives Considered

- **Kind, scope and expiry only, checked on write and read (chosen).** The
  blocker sweep needs exactly these three facts to tell the person what to
  renew before leaving, and none of them grants anything.
- **Store a reference the session can use to fetch the credential (a
  keychain item name, a file path).** Rejected: the checklist is re-injected,
  summarised and later uploaded, and a pointer that works for the session
  works for every reader of every copy.
- **Store the value, encrypted at rest.** Rejected: the key would have to be
  reachable by the session that re-reads the file, so it protects nothing
  against that session's transcript, compaction summary or upload, which is
  where the copies go.
- **Leave it to the agent's instructions, with no check.** Rejected: an
  instruction is a mistake filter at best, and agent-written state is the
  writer most likely to paste a value it was told about.

## Consequences

- The blocker sweep and the return lists can say "the App token for
  intentdriven/abcd, push and pull requests, expires 03:10" and nothing more;
  renewing it is a step the person or the issuing step takes.
- The checklist writer depends on the native secret scanner, so a new
  pattern added there protects the checklist too; a second scanner is not
  built (principle one-canonical-primitive).
- The later shared and public copies (the phone and shared-copies intent)
  inherit a file with no credential in it, so their own checks start from a
  clean source rather than being the only line of defence.
- The rule is stated in the brief's invariants as invariant 24, consumed by
  itd-2610170638184486.
