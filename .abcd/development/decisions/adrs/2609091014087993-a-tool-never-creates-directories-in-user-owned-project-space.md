---
id: adr-2609091014087993
slug: a-tool-never-creates-directories-in-user-owned-project-space
status: superseded
date: 2026-09-09
supersedes: null
superseded_by: adr-2609091248200336
related_intents: [itd-2609091014076309, itd-118]
related_rfcs: []
related_adrs: [adr-2609090717039680, adr-35]
---

# ADR-2609091014087993: A tool never creates directories in user-owned project space; agent and session scratch is machine-scoped

## Context

`AGENTS.md`'s concurrent-sessions convention makes the checkout the unit of
isolation: one checkout has one working tree, one HEAD and one index, the
record's gates read the whole tree, and a second session in the same checkout
fails both sessions' gates in both directions. A parallel agent therefore
needs its own worktree. That much is settled and is not reopened here.

What was never settled is where the worktree goes, and the answer by default
is git's: a sibling directory, next to the checkout, in whatever directory the
user keeps their projects in. On 2026-09-01 twenty-one spent worktrees
(1.4 GB) were removed by hand from the maintainer's project directory
([iss-2609020721142452](../../../work/issues/resolved/iss-2609020721142452-worktrees-for-parallel-lanes-are-created-one-directory-above.md)).
On 2026-09-06 a single session created twenty-two more, beside four projects
that have nothing to do with abcd, and `git worktree list` on that checkout
names twenty-seven. The maintainer's objection, verbatim: "I don't want a user
to be surprised that a folder is all of a sudden full of stuff."

The record already has a home for machine-scoped state and a key for it.
`~/.abcd/history/<root-sha>/`, `~/.abcd/transcripts/<root-sha>/` and
`~/.abcd/voyage/<root-sha>/` are each keyed on a repository's root-commit SHA,
for the reason `internal/core/history/location.go` gives: a checkout moves, is
renamed and is cloned twice on one machine, while its root commit does none of
that ([adr-2609090717039680](2609090717039680-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md)).
The voyage store moved there because an in-tree operations log would have
carried absolute paths into the repository ([adr-35](0035-lifeboat-as-coverage-experiment.md)),
and the same record made embark refuse to overwrite a directory abcd did not
produce. The read side of this rule is already in force: the loader refuses a
configuration root the caller does not own, and only a declaration in the
caller's own home (`~/.abcd/trusted-roots`) re-admits it. What the record lacks
is the write side, stated once: whose space a tool may create things in.

A neighbouring principle answers a different question and is not restated
here. [`durable-state-lives-where-the-platform-says-it-survives`](../../principles/durable-state-lives-where-the-platform-says-it-survives.md)
chooses a location by what the platform documents as surviving an update, a
re-clone or a sweep, priced against the 2026-08-21 plugin-update post-mortem.
It says where state survives. This record says whose space it is. A worktree
store under `~/.abcd/` satisfies both, and a location can pass one and fail the
other: a sibling directory survives every platform event and still belongs to
the user.

## Decision

**A tool never creates a directory in user-owned project space.** The parent
of a checkout, the directory the user keeps their projects in, and any
directory the user did not hand over are theirs. The checkout itself is handed
over only on the record's terms: `.abcd/.work.local/` is scratch by
declaration, `.abcd/work/` and `.abcd/development/` hold what the tiers say,
and the repository root holds nothing a tool put there on a guess (a stray
root file is already a `stray_root_docs` finding). Beside the checkout there
is no declared tier at all, so nothing goes there.

**Agent and session scratch is machine-scoped.** A worktree for a parallel
session, a verifier's scratch copy, an export nobody has asked for a home for:
each lives under the caller's own `~/.abcd/`, keyed on the repository's
root-commit SHA where it belongs to a repository, on the shape the history,
transcript and voyage stores already take. The store creates itself through
one seam, one level at a time, never through a symlink, exactly as the
transcript store does, because it needs no authority the caller does not hold.

**What a tool puts in its own space, it can list and reclaim.** Machine scope
is invisible to `ls` where the user works, which is the point, and the price
of that invisibility is a verb that lists what the store holds and a verb that
reclaims what is spent, with a line on the status board so the user learns the
count without asking. A store nobody can list is the same pile somewhere less
visible. [itd-2609091014076309](../../intents/drafts/itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md)
carries the worktree store, its list verb, its prune verb and the board line;
until it ships, the rule is followed by hand and says so in the principle.

**A directory the user made is theirs even inside the tool's own store.** The
list verb reports it; nothing moves or removes it. The store deletes only what
it can prove it created, on the ground embark's destination gate already
stands on.

**The rule binds an agent's conventions as much as the binary.** The
directories in the maintainer's project folder were created by agents
following `AGENTS.md`, not by `abcd`, and the rule would be hollow if it bound
only the Go. `AGENTS.md`'s concurrent-sessions convention names the store as
the place a session's worktree goes, and names no sibling-directory form.

## Alternatives Considered

1. **Git's default: a sibling directory, wherever the checkout is.** The status
   quo, and the shape every agent reaches for unprompted. Rejected: it is the
   thing the maintainer objected to. It also gives nothing a stable place to
   look, so a spent worktree is found by a human noticing a folder and
   remembering what it was.
2. **Inside the checkout, under `.abcd/.work.local/worktrees/`.** Gitignored,
   per-worktree, and with its owner. Rejected: a worktree inside the main
   working tree is walked by every tree scan — the payload gate, the name
   guard, docs-lint — which already flakes during worktree creation
   (iss-2608261331317889) and would slow every gate by a full tree per lane.
   The local tier is the right answer for small scratch and the wrong one for
   a second copy of the repository.
3. **A configured location, chosen per user in `config.json`.** Flexible, and
   a user who wants siblings could have them. Rejected: the default would still
   have to be something, and a setting that defaults to the user's project
   space reintroduces the surprise for everyone who never sets it; a setting
   that defaults to `~/.abcd/` is the chosen option with a knob nobody has
   asked for.
4. **Chosen: a machine-scoped store under `~/.abcd/`, keyed on the root
   commit, with list and reclaim.** The location the record already uses for
   everything machine-wide, the key that survives a checkout moving, and the
   two verbs that keep the store from being a pile.

## Consequences

- **The worktree store is an intent, not a fait accompli.** The rule is
  accepted today; its enforcement on the write side is
  itd-2609091014076309 in drafts, and the principle that carries the stance
  says the enforcement does not yet exist rather than implying it does.
- **The twenty-seven sibling worktrees on the maintainer's machine are a hand
  cleanup, not a migration.** The store never moves what it did not create, so
  nothing abcd ships will relocate them; they are removed with
  `git worktree remove` by the person whose directory they sit in.
- **`AGENTS.md` changes.** The concurrent-sessions convention gains the store
  as the place a worktree goes, and loses any wording that reads as a sibling
  directory. That edit lands with the intent, because a convention naming a
  verb that does not exist is a phantom gate.
- **The rule reaches beyond worktrees.** A verifier's scratch copy
  (`git archive HEAD | tar -x -C <scratch>`, per `AGENTS.md`) and any export
  whose home is unclear are covered by the same sentence; today's guidance
  routes them to `.abcd/.work.local/scratch/`, which remains right for what
  fits in a checkout and is joined by the machine scope for what does not.
- **The read side and the write side are now one stance.** `trusted-roots`
  refuses a root the caller does not own on the way in; this record refuses
  writing into space the caller was not handed on the way out. Neither cites
  the other yet; the principle names both.
- **What the harness itself creates is outside abcd's reach.** A harness's own
  checkout-isolation feature puts its worktree wherever the harness decides.
  The convention can only say which route a session should take; it cannot
  move what another tool made, and the intent's open questions carry whether
  the plugin surface should offer `add` as a host-run step for that reason.

## Status note

**Accepted by the maintainer's ruling of 2026-09-09**, recorded in
[`.abcd/work/DECISIONS.md`](../../../work/DECISIONS.md) and split there into
three records: this decision (the trust rule), the intent (the store and its
verbs) and the principle
[`the-users-directory-is-theirs`](../../principles/the-users-directory-is-theirs.md)
(the stance, with its enforcement stated as future work).
