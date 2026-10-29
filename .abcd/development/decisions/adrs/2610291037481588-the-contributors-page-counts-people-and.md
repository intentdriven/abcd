---
id: adr-2610291037481588
slug: the-contributors-page-counts-people-and
status: accepted
date: 2026-10-29
supersedes: null
superseded_by: null
related_intents: [itd-136]
related_rfcs: []
related_adrs: [adr-47]
---

# ADR-2610291037481588: The contributors page counts people and names none

## Context

[ADR-47](0047-abcdev-app-rendered-from-this-repository.md) made the website a
surface of the repository, with a `/contributors/` page that its explorer
renders from git history alone. As itd-136 shipped it, the page listed every
human author by name with a commit count, linked each one whose address was a
forge privacy address to a profile built from the login inside it, and put the
same names and profile links in the `record.json` the site publishes beside
it. The health page added a panel pairing two author names it took for one
person, with the `.mailmap` line that would fold them, which named both and
quoted the login in the address.

That is how every site abcd builds behaved, for abcd's own repository and for
any repository abcd manages. On 2026-10-29 the product thinker ruled that what
abcd publishes names no person: no member's name and no member's login appears
in credits (iss-2610291028185585). The repository has since dropped its
`.mailmap`, so nothing in the tree folds one person's identities into one.

The rest of the page is not about people. The bots-and-tools row names tools,
and the `Assisted-by:` tally names models, which is the attribution escape
ADR-47 decision 3 declared; neither is touched by the ruling.

## Decision

We will publish the human authors of record as a count, not a list. The
contributors page states how many people authored the history and how many
commits they made, and nothing about any one of them. The record JSON carries
the same two numbers, `human_authors` and `human_commits`, in place of the
`humans` list, and no author entry carries a profile link. The authorship
reader keeps nothing per person: it counts each human as it reads them, so no
later page or export can publish a name that was never kept.

People are counted once each by a simple key. A forge privacy address counts by
the login it carries, compared without regard to case, so both forms of that
address (with and without the numeric prefix) are one person. Any other address
counts by itself, case-folded. A name is never a key, because two people can
share a name and a count that merged them would be wrong the other way. A
repository that needs a finer fold restores a `.mailmap`, which `git shortlog`
applies before the count is taken.

The health page's same-author panel is withdrawn: what it proposed is now
applied where people are counted, and showing it named two people. Its two
interface strings stay declared, because `ui.json` is decoded with unknown
fields refused and a managed repository that declares them must still build.

The bots-and-tools row and the `Assisted-by:` tally stay as they were. Bot
entries lose the profile link the export could carry for them, since the row
names the tool and needs nothing more.

This partly supersedes ADR-47: the naming of human authors on the contributors
page that itd-136 delivered under it, and nothing else. ADR-47's decisions,
including decision 3's escape for model names on that page, stand.

The product thinker made the ruling. The technical facilitator settled how it
is carried out (the counting key, withdrawing the health panel, and the
renamed JSON fields), decided on their delegation and open to change.

## Alternatives Considered

- **Keep the list and blank the names.** Rejected: an empty field in the JSON
  still says one row per person, and a later change that fills it again has
  nothing to stop it. Removing the fields means there is nothing to fill.
- **Show initials or numbered placeholders.** Rejected: a row per person with
  a commit count still singles people out in a small team, where the count
  alone names them.
- **Fold people by name as well as by address.** Rejected: two people can share
  a display name, and the honest error for a count is to count one person twice,
  which a `.mailmap` fixes, rather than to merge two people, which nothing on
  the page would show.
- **Count people, folded by address and login (chosen).** It is the smallest
  change that names no one and still gives a number a reader can trust.

## Consequences

- **What a reader sees.** The contributors page opens with how many people
  wrote the history and their commit total, then the tools by name, then the
  model tally. Nobody's name, login or profile link is on any page the site
  builds from history.
- **What a consumer of `record.json` sees.** `authorship.humans` is gone and
  `authorship.human_authors` and `authorship.human_commits` take its place;
  bot entries no longer carry `profile`. A reader of the old list breaks, so
  the change is recorded as breaking.
- **The count can run one high.** One person committing from two unrelated
  mailboxes, or from a mailbox and a privacy address, counts twice until the
  repository folds them in a `.mailmap`. The page does not say which identities
  were folded, since saying so would name them.
- **The provenance check no longer admits a person's name on the contributors
  page.** Only tool names and declared models are in its attribution
  vocabulary, so a name that reached the page would fail `abcd site check`.
- **Out of scope.** The site footer's copyright line prints the package
  metadata's author, which is the licence holder rather than a credit list;
  this decision does not change it.
