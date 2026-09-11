---
id: adr-2609091248200336
slug: a-tool-never-creates-directories-in-user-owned-project-space
status: accepted
date: 2026-09-09
supersedes: adr-2609091014087993
superseded_by: null
related_intents: [itd-2609091014076309, itd-118]
related_rfcs: []
related_adrs: [adr-2609091014087993, adr-2609090717039680, adr-35]
---

# ADR-2609091248200336: A tool never creates directories in user-owned project space; the store's location binds now and its verbs bind when the store ships

## Context

[adr-2609091014087993](2609091014087993-a-tool-never-creates-directories-in-user-owned-project-space.md)
settled the trust rule this record carries: a tool never creates a directory
in space the user did not hand it, and agent and session scratch is
machine-scoped. Its occasion was twenty-one spent worktrees (1.4 GB) cleared
by hand from the maintainer's project directory on 2026-09-01
([iss-2609020721142452](../../../work/issues/open/iss-2609020721142452-worktrees-for-parallel-lanes-are-created-one-directory-above.md))
and twenty-two more created there by a single session on 2026-09-06, beside
four projects that have nothing to do with abcd, with `git worktree list`
naming twenty-seven; the maintainer's objection, verbatim, was "I don't want a
user to be surprised that a folder is all of a sudden full of stuff". The
record already had a home for machine-scoped state, `~/.abcd/`, and a key for
it, the repository's root-commit SHA, for the reason
`internal/core/history/location.go` gives: a checkout moves, is renamed and is
cloned twice on one machine, while its root commit does none of that. The read
side of the rule was already in force — the loader refuses a configuration
root the caller does not own, and only `~/.abcd/trusted-roots` re-admits it —
and the write side, stated once, was what the record lacked. None of that is
reopened here, and the neighbouring principle
[`durable-state-lives-where-the-platform-says-it-survives`](../../principles/durable-state-lives-where-the-platform-says-it-survives.md)
still answers a different question: where state survives, not whose space it
is.

What forces a new record is that the original says two things about
`AGENTS.md` that cannot both be read at face value
([iss-2609091129426411](../../../work/issues/resolved/iss-2609091129426411-adr-2609091014087993-s-consequences-still-say-the-agents-md.md)).
Among its normative clauses, in the present indicative: "`AGENTS.md`'s
concurrent-sessions convention names the store as the place a session's
worktree goes, and names no sibling-directory form." Among its Consequences:
"That edit lands with the intent, because a convention naming a verb that does
not exist is a phantom gate." The first states as fact what the second
schedules, and when the record was accepted neither was true of the file:
§ Concurrent sessions named no store, gave a parallel session no location at
all, and treated a worktree inside the checkout as ordinary — the shape the
same record rejects as its alternative 2.

The contradiction was closed from the other end on 2026-09-09
(`805bb023`). § Concurrent sessions names
`~/.abcd/worktrees/<root-sha>/<name>/` as where a session's worktree goes,
gives the original record and the principle as its grounds, refuses the
sibling and in-checkout forms, and says in as many words that the store has no
verbs: a plain `git worktree add` aimed at the path, with nothing to enumerate
the lane or prune a spent worktree until
[itd-2609091014076309](../../intents/drafts/itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md)
ships. That made the normative clause true. What it left is a Consequence
bullet that reads as though the whole edit is still ahead, when its location
half is behind and only its verb half remains, and nothing in the record says
the edit lands in two steps. An ADR is never amended, always superseded, so
the split is stated here.

A second, smaller correction rides with it, because two records superseding
one original would leave its `superseded_by` naming one of them and the other
orphaned. The original commits the worktree store to creating itself "through
one seam, one level at a time, never through a symlink, exactly as the
transcript store does". On 2026-09-09 (`24c2f2e3`) the three copies of that
sequence in the tree — `internal/core/history`'s, `internal/core/intent`'s and
`internal/core/lifeboat`'s — were consolidated into
`internal/fsutil.EnsureRealDir` and `EnsureRealDirAll`, because
[`one-canonical-primitive`](../../principles/one-canonical-primitive.md)
forbids a third copy and the branch that added the third had made three before
consolidating
([iss-2609091155525689](../../../work/issues/resolved/iss-2609091155525689-two-adrs-describe-the-real-dir-helper-at-its-pre-consolidation-home.md)).
The original's sentence is now satisfied by calling that primitive, but it
describes a mechanism where it could name the implementation, and that reading
is exactly what would produce a fourth copy.

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
one seam, because it needs no authority the caller does not hold, and the
create-then-prove step at every level is `fsutil.EnsureRealDir`
(`internal/fsutil/fsutil.go`) — a single
non-following `os.Mkdir` at the store's own mode, then the proof that the
result is a real directory rather than a symlink or a file — or
`EnsureRealDirAll` where a chain is walked, which proves each level as it
creates it. That is how the transcript store creates itself
([adr-2609091248201071](2609091248201071-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md)),
and the worktree store calls the same primitive rather than carrying the
sequence: the store owns its layout and its mode, and nothing else about
directory creation.

**What a tool puts in its own space, it can list and reclaim.** Machine scope
is invisible to `ls` where the user works, which is the point, and the price
of that invisibility is a verb that lists what the store holds and a verb that
reclaims what is spent, with a line on the status board so the user learns the
count without asking. A store nobody can list is the same pile somewhere less
visible. itd-2609091014076309 carries the worktree store, its list verb, its
prune verb and the board line; until it ships, the rule is followed by hand
and says so in the principle.

**A directory the user made is theirs even inside the tool's own store.** The
list verb reports it; nothing moves or removes it. The store deletes only what
it can prove it created, on the ground embark's destination gate already
stands on ([adr-35](0035-lifeboat-as-coverage-experiment.md)).

**The rule binds an agent's conventions as much as the binary, and it binds
them in two halves.** The directories in the maintainer's project folder were
created by agents following `AGENTS.md`, not by `abcd`, and the rule would be
hollow if it bound only the Go. **The location half binds now**: `AGENTS.md`'s
concurrent-sessions convention names `~/.abcd/worktrees/<root-sha>/<name>/`
as the place a session's worktree goes, names no sibling-directory form, and
refuses the in-checkout form. **The verb half binds when the store ships**:
the convention names `abcd worktree add` as the way a session gets its own
checkout only once that verb exists, which is itd-2609091014076309's
acceptance criterion and not this record's.

The split honours the original's reason rather than evading it. Its deferral
rested on one sentence — a convention naming a verb that does not exist is a
phantom gate — and that sentence is about a verb. A location is not a gate: a
plain `git worktree add` reaches `~/.abcd/worktrees/<root-sha>/<name>/` today,
so a convention naming it asserts nothing the reader cannot do. The principle
[`the-users-directory-is-theirs`](../../principles/the-users-directory-is-theirs.md)
had already taken the same split — "until it ships the rule is applied by
hand" — so the convention is now consistent with the principle rather than
silent beside it. What would be a phantom gate is the verb, and the verb stays
owed.

## Alternatives Considered

The original's four alternatives on location are carried as decided, because
nothing here reopens them:

1. **Git's default: a sibling directory, wherever the checkout is.** The status
   quo, and the shape every agent reaches for unprompted. Rejected: it is the
   thing the maintainer objected to, and it gives nothing a stable place to
   look.
2. **Inside the checkout, under `.abcd/.work.local/worktrees/`.** Gitignored,
   per-worktree, with its owner. Rejected: a worktree inside the main working
   tree is walked by every tree scan — the payload gate, the name guard,
   docs-lint — which already flakes during worktree creation
   (iss-2608261331317889) and would slow every gate by a full tree per lane.
3. **A configured location, chosen per user in `config.json`.** Rejected: the
   default would still have to be something, and a default in the user's
   project space reintroduces the surprise for everyone who never sets it.
4. **Chosen: a machine-scoped store under `~/.abcd/`, keyed on the root
   commit, with list and reclaim.**

On the contradiction, the three options iss-2609091129426411 put to the
maintainer:

5. **Accept the Consequence bullet as a prediction fulfilled in two steps, and
   record that reading in `DECISIONS.md`.** Cheapest. Rejected: it leaves the
   record saying one thing and meaning another, and a reader of the ADR alone
   still looks for an unmade change. A reading that lives only in the log is
   not in the record.
6. **Revert the `AGENTS.md` edit and hold both halves for the intent.**
   Rejected: it restores the contradiction the record carried when it was
   accepted, and it is defensible only if naming a location without a verb is
   a phantom gate, which it is not — the location is reachable by hand and the
   principle already said so.
7. **Chosen: supersede with a record that states the split.**

On the helper clause:

8. **Leave the sentence as a description of a mechanism and let the code
   comment name the primitive.** Rejected: the record is what a builder reads
   first, and a mechanism with no named home is the reading under which three
   copies accumulated. The principle's point is that the canonical home is
   discoverable at the moment of temptation.
9. **Chosen: name the primitive, and draw the boundary — the store owns
   layout and mode, `internal/fsutil` owns create-then-prove.**

On the number of records:

10. **Two successors for one original, one per issue.** Rejected: an original's
    `superseded_by` names one record, so the second would be unlinked from the
    record it corrects. One successor per original is the shape the store's
    typed links can carry, and it is what the maintainer's ruling — a separate
    record for each original — asks for.

## Consequences

- **adr-2609091014087993 is superseded and retained.** `AGENTS.md`, the
  principle, the brief and two intents cite it, so it stays with both halves
  of the supersession in its frontmatter and its decision text as written.
  Every clause it decided is carried here; the two sentences that could not be
  read together are replaced by the split.
- **The worktree store is still an intent, not a fait accompli.** The rule is
  accepted; its enforcement on the write side is itd-2609091014076309 in
  drafts, and the principle that carries the stance says the enforcement does
  not yet exist rather than implying it does.
- **`AGENTS.md` changed in `805bb023`, and one criterion is still owed.** The
  concurrent-sessions convention names the store, refuses the sibling and
  in-checkout forms, and says the store has no verbs. What remains is
  itd-2609091014076309's last acceptance criterion — that the convention
  "names the verb as the way a session gets its own checkout" — which lands
  with the intent and closes the verb half.
- **A store that wants this creation discipline calls `internal/fsutil`.** A
  private `ensureRealDir` is refused by
  `TestNoNonCanonicalAtomicWritePrimitives`, whose detector names that
  spelling since `24c2f2e3`; the worktree store, when built, calls
  `EnsureRealDirAll` from the caller's home and carries nothing of its own.
- **The twenty-seven sibling worktrees on the maintainer's machine are a hand
  cleanup, not a migration.** The store never moves what it did not create,
  so nothing abcd ships will relocate them.
- **The rule reaches beyond worktrees.** A verifier's scratch copy and any
  export whose home is unclear are covered by the same sentence; today's
  guidance routes them to `.abcd/.work.local/scratch/`, which remains right
  for what fits in a checkout and is joined by the machine scope for what does
  not.
- **The read side and the write side are one stance.** `trusted-roots`
  refuses a root the caller does not own on the way in; this record refuses
  writing into space the caller was not handed on the way out. Neither cites
  the other yet; the principle names both.
- **What the harness itself creates is outside abcd's reach.** A harness's own
  checkout-isolation feature puts its worktree wherever the harness decides;
  the convention can only say which route a session should take, and the
  intent's open questions carry whether the plugin surface should offer `add`
  as a host-run step for that reason.
- **No detector exists for the class of defect that occasioned this record.**
  A normative clause asserting what a named file says can be checked only by
  reading the file, and no record-lint rule cross-checks a claim against the
  convention it names. The principle
  [`enforcement-claims-are-facts`](../../principles/enforcement-claims-are-facts.md)
  names that promotion; until it exists, the check is a reviewer's.

## Status note

**Accepted by the maintainer's ruling of 2026-09-09** on iss-2609091129426411
(supersede with a record that states the split) and iss-2609091155525689 (a
separate record for each original), recorded in
[`.abcd/work/DECISIONS.md`](../../../work/DECISIONS.md). It is a new record
rather than an edit to adr-2609091014087993 by the standing instruction that
an ADR is never amended and always superseded, so the original's decision text
stands as written and this record carries both corrections.
