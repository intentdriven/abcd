---
id: adr-2610120558328615
slug: a-private-advisory-s-record-never-enters
status: accepted
date: 2026-10-12
supersedes: null
superseded_by: null
related_intents: [itd-2610112239022766, itd-149]
related_rfcs: []
related_adrs: [adr-44, adr-2610031751065746]
---

# ADR-2610120558328615: A private advisory's record never enters the public ledger, its store is never committed, and nothing public carries an advisory id or hint

## Context

A private security advisory is the one piece of work abcd's records cannot hold
in the open. The issue ledger is committed and public, so an advisory copied
into it discloses the hole before its fix exists; and every outward artefact a
release makes (commit messages, pull-request bodies, the changelog, the release
page, records made public) is a place an advisory can be named by accident.
The product thinker ruled on 2026-10-09
([iss-2610090703514921](../../../work/issues/open/iss-2610090703514921-a-drain-reads-only-the-issue-ledger-so.md))
that an advisory's record stays out of the public ledger until its fix lands,
and on 2026-10-12 adopted `abcd embargo`
([itd-2610112239022766](../../intents/planned/itd-2610112239022766-when-someone-reports-a-security-hole-in.md),
refining [itd-149](../../intents/drafts/itd-149-abcd-handles-inbound-security-advisories.md))
as the first rung that does it in code, including closing an advisory on GitHub
on the person's approval.

The hand-runs that preceded the intent kept the private records in an
uncommitted worktree and searched for every advisory id by hand before each
push. Those are trust rules, not features: they bind every verb that writes a
record or an outward text, not only `embargo`. Under the itd-84 routing
(capability to intent, trust rule to ADR and brief invariant, as adr-44 did for
remote writes) they are recorded here.

Already locked: worktrees live in the machine-scoped store
([adr-2610031751065746](2610031751065746-the-worktree-store-lives-under-the.md));
a remote write happens only through a dedicated, confirmed verb and the
caller's own identity ([adr-44](0044-remote-mutation-and-caller-identity.md)).

## Decision

1. **A private advisory's record never enters the public ledger while its
   advisory is unpublished.** abcd writes it only into the embargo store, a
   worktree at `~/.abcd.noindex/worktrees/<root-sha>/embargo/`, and never under
   the checkout's own `.abcd/work/issues/`. The link from an advisory to its
   record lives in the store's state, not in the record, and a record minted
   from an advisory takes a neutral slug, never one derived from its title.
2. **The store is never committed.** It is created detached, so nothing in it
   has a ref to push, and every verb that reads or writes it first refuses,
   writing nothing, when its HEAD is on a branch, when its index tracks a
   record the state names, when it resolves inside the current checkout, or
   when its state file is not owned by this account and closed to group and
   other writes.
3. **Nothing public carries an advisory id or a hint of one.** Every outward
   artefact is checked before it is made public, and abcd refuses one that
   names an unpublished advisory's id, a CVE id reserved for it, its title, or
   the location of the private records. A refusal names where the match was
   found and never repeats it. abcd's own listings that would show a private
   record's title (such as `peers`) skip the store. What a check cannot
   recognise, a paraphrase or a count of unfixed reports, is stated as its
   limit and stays the author's to avoid.
4. **Publishing and closing an advisory on GitHub each need one explicit
   approval per act, bound to exactly what was shown.** A publication plan and
   a closing text are each shown first with a digest; the act runs only with an
   approval naming that digest, and an approval for any other plan or text
   sends nothing. A host passes an approval only after the person, in that
   session, has seen what it covers and said yes; an agent never supplies one
   on its own. This refines adr-44's first rule: adr-44 requires a dedicated,
   confirmed verb for a remote write; this rule makes the confirmation one per
   act and binds it to the content confirmed, because an advisory published or
   closed with the wrong text cannot be taken back into private.

## Alternatives Considered

- **Keep the record in the public ledger with its text redacted.** Rejected:
  the record's file name, its date, its category and its severity are each a
  hint, and the 2026-10-09 ruling keeps the record out, not merely its words.
- **Hold the work in GitHub's temporary private forks.** Rejected: they skip
  CI, branch protection and the merge queue, so a fix made there lands
  unchecked.
- **A private remote or a separate private repository for the records.**
  Rejected for this rung: it adds a second remote to keep in step and a second
  place to leak from, for a need (sharing between machines) the intent's scope
  conditions exclude.
- **Encrypted records committed to the ledger.** Rejected: the ciphertext's
  presence and its file history are themselves a hint, and abcd would take on
  key management.
- **A detached, never-committed store in the machine-scoped worktree store,
  with an outbound check and per-act approvals.** Chosen: it is what the
  hand-runs proved, it holds records where the existing capture and resolve
  code already reads them, and each failure mode has a refusal a test can hold.

## Consequences

- `internal/core/embargo` carries the refusals in decisions 1 and 2 as its
  store check, and each is held by a named test in the intent's spec.
- Every outward text abcd composes during an embargo passes the outbound check
  of decision 3 before it is posted or committed; a later verb that writes
  outward text inherits the obligation.
- The records live on one machine only. Handling one release's advisories from
  two machines is out of reach until a later decision lifts it.
- A record that lands with its fix enters the public ledger only after the
  advisory is published, or redacted; redacting and restoring are a later rung
  and inherit decision 3.
- The brief carries the rule as invariant 22 in
  [`02-constraints/03-invariants.md`](../../brief/02-constraints/03-invariants.md),
  in force once this record is accepted.
