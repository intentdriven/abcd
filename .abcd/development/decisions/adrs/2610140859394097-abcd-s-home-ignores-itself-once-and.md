---
id: adr-2610140859394097
slug: abcd-s-home-ignores-itself-once-and
status: proposed
date: 2026-10-14
supersedes: null
superseded_by: null
related_intents: [itd-2610140858048004]
related_rfcs: []
related_adrs: []
---

# ADR-2610140859394097: abcd's home ignores itself once, and credentials never sit in a working tree

_Drafted by the facilitator on 2026-10-14 from the product thinker's rulings quoted below; the status stays `proposed` until the product thinker accepts it._

## Context

abcd keeps a person's machine-scoped records in `~/.abcd.noindex`: redacted session transcripts, run and loop state, labs, lifeboats, interview records, settings backups, and from iss-2610090740139804 private ideate verdicts. A person whose home folder is itself a git repository (a dotfiles repository pushed to a remote) has every one of those files one `git add -A && git push` from public. Today only the credential store's abcd home and a new sources corpus refuse to write inside a git working tree; every other writer writes there unchecked. The security review of iss-2610090740139804 found the route for private verdicts.

The product thinker's rulings, 2026-10-14, verbatim: "self-ignoring home by default, but instruct the user to manually remove the .gitignore file in ~/.abcd.noindex if they want to add it; aa deliberate user action to ensure it doesn't happen accidentally. Credentials still stay out even then"; and, on refusing every write instead, "I'm inclined to always refuse all of ~/.abcd.noindex/, but that may break things".

Constraints already in force: abcd never changes how a repository works on its own (the security standard of 2026-10-11, `abcd-fixes-what-it-adds`); the user's directory is theirs; nothing in abcd runs a repository's own git configuration to judge it where the credential rule already avoids doing so.

## Decision

We will keep abcd's home out of any repository in the person's home by a file inside abcd's own folder, laid once, and we will keep credentials out of every working tree regardless of it:

1. **abcd's home ignores itself.** Every abcd command but the status line, which writes nothing at all (iss-2610050556383525), before and after it runs, makes sure `~/.abcd.noindex/.gitignore` holding `*` has been laid in an existing abcd home, with a marker beside it recording that it was. It never creates the home to do so, never overwrites a file, never follows a link, and never reads or runs anything of a repository.
2. **Once, then the person's.** While the marker stands, abcd never lays the file again. Deleting `.gitignore` is the deliberate act by which a person lets a repository in their home track abcd's home, and abcd respects it.
3. **Credentials never sit in a git working tree.** The credential store's abcd home stays refused inside any working tree, the account's own included, whatever the ignore file says: an ignore file is a convention a person can override with one flag, and a credential is the one record whose leak cannot be taken back.

## Alternatives Considered

- **Refuse every write into a home inside a working tree.** The safest for privacy, and the product thinker's first inclination; rejected because it stops transcripts, worktrees, runs, the install and the dashboard for everyone whose home is a git repository.
- **Refuse private ideate verdicts only, like credentials.** Rejected as too narrow: transcripts are as sensitive, and every other writer stays exposed.
- **Ask git whether the home repository ignores abcd's folder.** Rejected: `git check-ignore` runs the home repository's own configuration, which the credential rule avoids on purpose, and it changes nothing for a repository that does not ignore it.
- **Lay the file whenever it is missing.** Rejected: it would undo the person's deliberate deletion, which the product thinker made the opt-in.
- **Chosen: a self-ignoring home, laid once, with credentials refused regardless.**

## Consequences

- A dotfiles repository in the home no longer stages abcd's files by default, including private ideate verdicts, which may therefore be written in the account's own git-tracked home.
- A file the home repository already tracks stays tracked: the ignore file protects new files only, and a `git add -f` overrides it. Both are stated in the intent's scope conditions.
- Every command walks the home twice more (before and after it runs), a few stats each; a failure to lay prints one stderr line and never fails the command, and a symlinked `~/.abcd.noindex`, which no lay can serve, reads as absent and prints nothing.
- A verb whose sentence says it writes nothing may be the command that lays the file in a home that lacks it: the write is the front door's, once per home, and the brief says so.
- The front door gains a write into the home before the command runs; the old home's stop still writes nothing.
