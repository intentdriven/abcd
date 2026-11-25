---
id: adr-2611251746369732
slug: a-machine-is-published-on-the-shared
status: proposed
date: 2026-11-25
supersedes: null
superseded_by: null
related_intents: [itd-2611141610197753, itd-2611141610201875, itd-2609151838312703, itd-2611251809580662]
related_rfcs: []
related_adrs: [adr-2610150729333337, adr-2610031352212697, adr-2611251746361324]
---

# ADR-2611251746369732: A machine is published on the shared remote only as a chosen label, a state word, times and counts

## Context

A drive's one line of state (itd-2611141610197753) is stored in the checkout the drive runs from, and only a status surface run in that checkout shows it. With drives on several machines and accounts working one ledger, nobody can tell from one machine whether another machine's drive is working, and `abcd peers` shows another account's lanes only at their last renewal.

The product thinker ruled on 2026-11-25 that a person sees in one view everything associated with their GitHub login across machines. The technical facilitator settled how: each drive publishes its state line on every change to the person's own status branch on origin, the channel `abcd announce` already writes ([adr-2610150729333337](2610150729333337-a-status-branch-moves-without-a.md)), under a machine label the person chooses, and `abcd status` and the board gain a view by login: every drive with its state, and that login's reservations and pull requests. The product thinker was told, and accepted, that anyone who can read the repository then sees those lines, as they see reservations today.

Two constraints were already locked. A status branch can never be deleted or rewritten, so whatever is pushed there is permanent and public for as long as the repository is (adr-2610150729333337, Consequences). And an announcement carries no free text: its form is a kind, an item, the holder id and times, and nothing else (`.abcd/development/brief/04-surfaces/38-announce.md`). What a drive's line holds on its own machine is wider than that: its `on` part and its act can carry a refusal's own remedy, and a remedy can name a path.

## Decision

We will publish a machine on the shared remote only as a label the person chose, a state word, times and counts.

1. **What may be published.** A drive's published line carries, and carries only:
   - the machine label the person chose;
   - the state word (`working`, `waiting`, `stalled` or `stopped`), and for a wait or a block the enumerated word of its cause (an outage, a usage window, a fast-forward declined), never the sentence the drive composed for it;
   - times: since when, until when where the wait names an end, when the line was written, and the drive's end time;
   - counts: lanes done, in progress, set aside and handed back, and the records waiting on that machine for a person to take in;
   - the ids of records and pull requests the remote already shows, such as the issue a reservation names, and where a fast-forward of the drive's checkout was declined, how many commits it is behind;
   - what every reservation already publishes and what names no machine: the opaque holder id, the working copy's random key and abcd's version.
2. **What is never published.** A host name, a path (absolute or relative, of a checkout, a worktree, a log or a home), an account name on the machine, a network address, and any free text the drive composed: a refusal's remedy, a log line, an issue's title. The published form is built from typed fields, never by cutting the local line, so nothing composed for the terminal reaches the remote by accident.
3. **The label is the person's, and it is checked.** It is `drive.label` in the person's own configuration on that machine (`~/.abcd.noindex/config.json`), never in a repository file, so no repository names a person's machines. It is 1 to 24 lower-case letters, digits and hyphens, and a label equal to the machine's host name or the account's name, in any case, is refused. Until a label is chosen, the drive publishes under a label drawn from the working copy's random key, which names no machine, and its start says how to set one.
4. **Where and when.** One empty-tree commit of fixed fields on the person's own status branch, validated field by field as an announcement is, but a form of its own, since an announcement's kinds and day-long life do not fit a state line. It is pushed when the state word or what the drive waits on changes, and with the hourly renewal of what the drive holds, never more often, and at the drive's end it says `stopped`. Its author and committer are those a reservation's commit has. The person's forge login is not written into it: the view is headed with the login read on the reading machine, and a person is told by their holder id and the ids their home declares, as reservations already tell them (itd-2611251809580662, decision 7).

## Alternatives Considered

- **A label, a state word, times and counts, from typed fields (chosen).** It answers "is my drive on the other machine working?" from any machine, and nothing in it identifies a machine beyond the word the person chose for it, on a branch whose history can never be erased.
- **Publish the local state line as it is.** Rejected: its `on` part and its act carry the drive's own sentences, and a remedy can name a path on the machine. Once pushed to a status branch it can never be removed.
- **Identify a machine by its host name.** Rejected: a host name often carries a person's name or an organisation's, and it is permanent once pushed. A label the person chose is as useful to them and says nothing to anyone else.
- **Publish nothing, and have each machine report only locally.** Rejected by the product thinker's ruling: a person would still have to visit each machine to learn whether its drive is working.
- **A relay or a service outside the repository.** Rejected: a status branch is an existing channel with an existing reader (`abcd status`), and a service would carry the same lines through a party outside the repository, with its own place to run and its own trust to establish.

## Consequences

- **Easier:** one view by login shows every drive of a person, its state and its counts, from any machine that can read the repository, with no service to run.
- **Harder:** everyone who can read the repository sees each drive's state, times and counts, permanently, which the product thinker accepted. A machine with no label chosen still publishes, under a label drawn from its working copy's random key.
- **A declined fast-forward is published as its cause word**, with no path, so the view by login carries it (itd-2611141610201875, decision 5) without naming the dirty file.
- **New obligations:** the brief gains an invariant beside the reservation and status-branch rules, written in the change that builds the published line: nothing about a machine but a chosen label, a state word, times, counts and already-public ids reaches the shared remote. The reader that renders the view treats every published field as untrusted text, as it does an announcement. A test feeds the publisher a state whose local line names a path and a host name, and asserts that neither reaches the commit.
