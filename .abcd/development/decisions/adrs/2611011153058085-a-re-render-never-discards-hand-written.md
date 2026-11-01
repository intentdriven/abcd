---
id: adr-2611011153058085
slug: a-re-render-never-discards-hand-written
status: proposed
date: 2026-11-01
supersedes: null
superseded_by: null
related_intents: [itd-2611011151475128, itd-2611011151512956, itd-2611011239589823, itd-2610031348087517, itd-102]
related_rfcs: []
related_adrs: [adr-2610031751065746]
---

# ADR-2611011153058085: A re-render never discards hand-written text unasked

## Context

itd-2611011151475128 renders every documentation surface (the README and the
`docs/` pages) from the one home of each claim, and makes the brief-sourced
sections compulsory in abcd-managed repositories once the brief has an editing
surface. A rendered page is still a file anyone can open and edit, and
people will rewrite their README whatever the convention says. A render that
writes the surface whole would then destroy whatever was written by hand since
the last one, silently and in the owner's own repository.

Three standing records bear on it. itd-2610031348087517 never touches the
owner's own words and leaves a hand-edited part as it is; that is the
behaviour this decision would replace for repositories under the new rule
(the reversal waits on the product thinker's confirmation at the intents'
planning interview), so the replacement has to be at least as safe. itd-102
settled that re-rendering from the identity block is always a proposed diff
a person adopts, never an autonomous rewrite; whether the new rule renders in
place or holds each section to its home by a proposed diff is still open in
itd-2611011151475128, and this decision binds under either answer. And `every-claim-has-one-home` makes
the brief, not the surface, the place the owner's words belong, so the safe
behaviour cannot be to keep the hand edit in the surface for ever.

## Decision

In a repository under the new rule, we will never let a render discard text
written by hand into a rendered page unless the owner has chosen to, whether
the render writes the section in place or proposes a diff for the person to
adopt (a proposed diff that drops hand-written text is a discard too). Before
a render replaces or proposes to replace a section,
abcd compares the section with what it last rendered there. Text that differs,
or prose added outside any rendered section, is hand-written: the render
leaves it in place and reports it, and itd-2611011151512956's reverse path
offers to move it into its source: the brief for a page rendered from the
brief, the how (itd-2611011239589823) for a guide or tutorial rendered from
the how. The section is rendered over only once the piece has been moved
into its source, or the owner has explicitly chosen to drop it.

## Alternatives Considered

- **Render over, and rely on version control to recover the text.** Rejected:
  the text is recoverable only by someone who knows it was lost, and a
  compulsory render in someone else's repository that deletes their writing is
  the trust failure the whole rule must avoid.
- **Refuse to render while any hand edit exists.** Rejected: one stray edit
  would freeze every section of the surface, and the brief's changes would
  stop reaching it; the hold belongs on the edited section only.
- **Keep the hand edit in the surface permanently, as the setup parts do
  today.** Rejected for repositories under the new rule: it leaves a second
  home for the claim, which is the drift `every-claim-has-one-home` exists to
  remove.
- **Leave it in place, report it and offer it back (chosen).** The edit is
  never lost, the section it sits in waits only until the owner decides, and
  every other section keeps rendering.

## Consequences

- abcd must keep what it last rendered into each section, or a digest of it,
  so a hand edit can be told from a stale render. The setup parts' marker
  already carries a checksum over its body (`internal/core/ahoy/readme_marker.go`,
  which itd-2610031651058674's check reads). A section rendered in place
  reuses that marker rather than invent a second one; a section held to its
  home by a proposed diff needs the same record of what was last proposed,
  in a form the marker question settles.
- A section can be held from rendering by an undecided piece of hand-written
  text; the reverse path has to surface that hold, or a section can go stale
  unnoticed behind it.
- A render stays the person's act and writes only the sections abcd renders
  (`the-users-directory-is-theirs`, adr-2610031751065746); committing what it
  wrote is the person's act too.
- The brief-sourced sections can be compulsory without overwriting anyone:
  the rule that makes them safe is this one, so it ships before or with them.
- This needs a brief invariant beside the record rules, so a later change to
  rendering cannot quietly drop it.
