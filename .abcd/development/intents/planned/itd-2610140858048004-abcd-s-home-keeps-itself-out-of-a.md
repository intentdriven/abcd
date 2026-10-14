---
id: itd-2610140858048004
slug: abcd-s-home-keeps-itself-out-of-a
spec_id: spc-2610140858477542
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# abcd's home keeps itself out of a dotfiles repository

## Press Release

> abcd's home keeps itself out of a dotfiles repository. Frank, a DevOps engineer, keeps their whole home folder in a git repository they push to a public remote, and abcd keeps session transcripts, private idea verdicts and run state under ~/.abcd.noindex in that same home. The first time abcd runs, it lays a .gitignore holding * in ~/.abcd.noindex, so Frank's next git add -A skips everything abcd keeps there and nothing private leaves the machine by accident. abcd lays the file once: if Frank deletes it because they want their dotfiles repository to track abcd's home, abcd leaves it deleted. Credentials stay out of every git working tree whatever the file says. "I found out about the .gitignore when I went looking for why my dotfiles diff was clean — which is exactly how I wanted to find out."

## Why This Matters

abcd keeps a person's machine-scoped records in `~/.abcd.noindex`: redacted session transcripts, private ideate verdicts (iss-2610090740139804), run and loop state, labs, lifeboats, interview records and settings backups. For anyone whose home folder is itself a git repository, every one of those files is one `git add -A && git push` from a public remote, and today nothing but the credential store and a new sources corpus refuses to write there. The person did nothing wrong; abcd put the files in the path of their ordinary habit. abcd added the risk, so abcd removes it (the abcd-fixes-what-it-adds principle), and does so inside its own folder, without touching the person's repository or its configuration.

## Mechanism

We expect a `.gitignore` holding `*` inside `~/.abcd.noindex` to keep every file abcd writes out of a dotfiles repository, because git applies a `.gitignore` to the paths beneath its own directory, so the file excludes abcd's untracked files without abcd reading, running or changing anything of the person's repository. It would be shown wrong by a `git add -A` in such a home that stages a file abcd wrote after the file was laid.

## Scope Conditions

- The files under `~/.abcd.noindex` are not already tracked by the home repository: an ignore file never untracks a committed file. <!-- cond: cond-2610140858479946 -->
- The person stages with `git add` and not `git add -f`, which overrides every ignore file. <!-- cond: cond-2610140858479857 -->
- The person's git honours a `.gitignore` in a subdirectory, as every git release abcd supports does. <!-- cond: cond-2610140858471836 -->

## What's In Scope

- Laying `~/.abcd.noindex/.gitignore` holding `*`, owner-only, once per home, from the step every abcd command passes through, the status line excepted: it writes nothing at all (iss-2610050556383525).
- A marker beside it that records the file was laid, so a deleted file is never laid again.
- Homes that predate the change: the first run after it lays the file.
- A private ideate verdict written into the account's own git-tracked home, which the laid file keeps out of that repository.
- The help and docs saying what the file is for, and that deleting it is how a person lets their home repository track abcd's home.

## What's Out of Scope

- Reading, running or changing the person's own repository: its ignore files, its configuration, its hooks. abcd judges no repository to lay the file; it lays it in every home.
- Untracking a file the home repository already tracks.
- Loosening the credential rule: the abcd home store stays refused inside any git working tree, the account's own included, whatever the file says.

## Acceptance Criteria

- Given a home with no `~/.abcd.noindex`, When any abcd command creates it, Then `~/.abcd.noindex/.gitignore` holds `*`, is readable by the account alone, and the marker records that abcd laid it.
- Given a `~/.abcd.noindex` from before this change with no `.gitignore` and no marker, When the next abcd command runs, Then the file is laid once and marked.
- Given abcd laid the file and the person deleted it, When any abcd command runs, Then abcd does not lay it again, and says nothing about it.
- Given a home folder that is a git working tree with the file laid, When the person runs `git add -A` in the home, Then nothing under `~/.abcd.noindex` is staged.
- Given a home folder that is the account's own git working tree with the file laid, When `abcd ideate record --private` runs, Then the verdict is written; and Given a HOME planted inside another checkout, Then the private verdict is refused with nothing written.
- Given any home inside a git working tree, with or without the file, When a credential is stored in the abcd home, Then the store refuses it, as it does today.
- Given the old `~/.abcd` still stands, or `~/.abcd.noindex` is a symlink, When any abcd command runs, Then nothing is laid and nothing is said: the stop writes nothing, and the lay never writes through a link, reading a symlinked home as absent.
- Given the file cannot be laid (a read-only home), When an abcd command runs, Then the command still runs, and one line on stderr says the file was not laid and why, except under `--json` and in a hook call, where a machine reads stderr and nothing is added to it.
- Given a home with no file laid, When the harness runs the status line, Then nothing is written: the status line never lays the file.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a .gitignore holding * in ~/.abcd.noindex keeps every file abcd writes out of a dotfiles repository's git add -A; a staged abcd file after the lay would show it wrong
