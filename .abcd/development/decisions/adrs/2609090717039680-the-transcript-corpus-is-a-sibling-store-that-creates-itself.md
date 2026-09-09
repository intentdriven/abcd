---
id: adr-2609090717039680
slug: the-transcript-corpus-is-a-sibling-store-that-creates-itself
status: accepted
date: 2026-09-09
supersedes: adr-29
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: [adr-22, adr-29, adr-56]
---

# ADR-2609090717039680: The transcript corpus is a sibling store that creates itself, and the per-repo location is an opt-in pull

## Context

[ADR-29](0029-native-transcript-corpus.md) settled that abcd captures its own
session transcripts into a native local store: no external tool, keyed on the
repo's root-commit SHA, never committed, redacted at write time through the
two-stage sanitise-then-verify model of ADR-6, with an optional private
companion as a separate opt-in path. Every one of those properties still holds
and none of them is reopened here.

What ADR-29 did not settle is where on the machine the store sits, and the
location it inherited is what made the store fail silently. The corpus was laid
out at `~/.abcd/history/<root-sha>/transcripts/`, inside ahoy's registry
namespace: `index.json` and the per-repo `meta.json`, written by `abcd ahoy
install`. `history.Capture` required that directory to exist and deliberately
never created it, because `ownedDirsReal` treated an absent level and a
symlinked one alike and refused both. On a machine where install had not run,
`hook session-end` logged one line to stderr, exited 0 as a shutdown hook must,
and staged nothing. Session after session, with no marker anywhere: a store that
reads as wired while the corpus never accrues, which is the exact failure the
transcript work exists to prevent (iss-95).

Creating the directory where it stood would have closed the defect and left the
fault. `~/.abcd/history/` is another package's install-owned substrate, and a
capture hook that lays out a directory inside it is one package writing into
another's namespace. Brief invariant 15 already asks for the opposite: only the
history core package touches the store's path, held by a boundary test. The
location is therefore what has to move, and moving it is what makes the creation
clean.

## Decision

**The transcript corpus is a sibling of ahoy's registry, not a sub-tree of it.**
It lives at `~/.abcd/transcripts/<root-sha>/{records,staging}/`, and
`~/.abcd/history/` keeps exactly what ahoy owns: the index and the per-repo
metadata. `internal/core/history` is the only package that lays out or judges
the corpus path, which is invariant 15 satisfied rather than promised.

**The store creates itself, through one seam, and keeps the discipline it
replaces.** `history.Resolve(repoRoot, rootSHA)` is the single door every read
and write path goes through, so the store cannot be reached by a route that
skipped the creation, the symlink check or the migration. `ensureRealDir` walks
the chain top-down, creates each level with `os.Mkdir` rather than `MkdirAll`
(which creates a whole chain without judging any of it), and re-verifies every
level as a real directory on every resolve, so a parent swapped between calls is
caught before the leaf is opened. What `ownedDirsReal` was protecting was never
absence: it was creating or writing *through* a planted symlink, and that refusal
is kept and made explicit. Creation needs no authority the caller does not
already hold, since every level is under their own home. Directories are `0o700`:
records are redacted but still a verbatim account of the caller's sessions, and
staging holds them unredacted.

**The key stays the repo's root-commit SHA, as a directory.** One user-level
store still has to answer "this repo's transcripts", and the root commit is the
one name that survives what a checkout does: it moves, it is renamed, it is
cloned twice on one machine, and its root commit changes under none of that. The
key being a directory rather than a field means `list`, `show`, `staged` and
`drain` each read one repo's lane, and no cross-repo filter exists to get wrong.

**The per-repo location is an opt-in pull the caller declares in their own
home.** `~/.abcd/local-transcript-roots` takes one absolute checkout path per
line with `#` comments, matched both as written and symlink-resolved, and is
honoured only while it is a regular file this uid owns that no one else can
write. A declared checkout keeps its store at
`<repo>/.abcd/.work.local/transcripts/<root-sha>/`, in the gitignored,
per-worktree local tier, so a pulled-in transcript is never a commit candidate
and never merge-conflicts between concurrent sessions. The declaration is
home-scoped on the `~/.abcd/path-entry` and `~/.abcd/trusted-roots` idiom, and
for the reason those give. A file inside the checkout would let a cloned repo
redirect the machine's transcripts into its own working tree, where its own
`.gitignore` decides whether they are committable, which is the tree asserting
where the machine's session record is kept. An environment variable clears the
letter of that bar and not its spirit, because a repo can ship the shell, direnv
or task-runner configuration that sets it, and the tree asserts it one
indirection out. A declaration that is present but not honoured says so on
stderr, since an ignored opt-in and one never written are otherwise the same
silence.

**A corpus at the earlier location is moved, loudly.** The first resolve moves
every record and staged file out of
`~/.abcd/history/<root-sha>/{transcripts,staging}/` into the store, file by file
rather than by directory rename, which is idempotent under a concurrent peer and
correct across filesystems, as the per-repo opt-in can be. Nothing is deleted
except a source whose bytes are already at the destination; a non-regular entry
or a name collision with different content is left where it is and counted. The
move is loud both ways: `Resolve` returns notes the way `rules.Resolve` does and
every front door prints them on stderr, and a `transcripts.moved` tombstone at
the old path names the new one for whoever goes looking there. The tombstone is
withheld while anything is left behind, so the next resolve looks again.

## Alternatives Considered

1. **Create the directory where it stood, under `~/.abcd/history/`.** The
   smallest diff, and it closes iss-95 on the machine that never installed.
   Rejected: it leaves a capture hook laying out a directory inside ahoy's
   install-owned namespace, which is the arrangement invariant 15 exists to
   forbid, and it buys the fix by conceding the boundary.
2. **Read both locations, and write to the new one.** No migration step and no
   window in which a record is in flight. Rejected: from the first capture
   onward there are two stores diverging, and every read path has to merge them
   forever, so the cost is permanent and grows.
3. **Refuse, and name the remedy.** Honest, loud, and one command from working.
   Rejected: it reopens iss-95 from the other end, because a machine that *had*
   installed would stop capturing until someone ran the remedy. That is the
   silent-corpus failure wearing a louder hat, and a shutdown hook is exactly
   where nobody reads the hat.
4. **Chosen: a sibling store, self-creating through one seam, with a
   home-declared per-repo opt-in and a move-loudly migration.** The corpus
   accrues on a machine that has done nothing but check the repo out, the
   symlink discipline survives intact, and one package owns the path.

## Consequences

- **Capture no longer has an install-shaped precondition.** Its only
  precondition is a home directory the caller can write, which they already
  have, so a machine that has never run `abcd ahoy install` accrues a corpus
  from its first session.
- **The `history.transcripts_missing` gap is retired**, and the SessionStart
  notice built on it goes with it. A self-creating store makes "absent" the
  ordinary state of a repo not yet captured, so a required gap there would have
  every status board assert that transcripts will not be captured, which is
  false. The test that pinned the notice is inverted: a session starting on a
  machine with no `~/.abcd` is silent and leaves a store behind.
- **`ahoy install` opens the store rather than laying it out.** It calls
  `history.Resolve` and points `meta.json`'s corpus block at what comes back, so
  that block carries the store's home-redacted path rather than a name relative
  to the `<root-sha>/` directory.
- **The read verbs take the repo root.** `List`, `Read`, `Stage` and
  `ListStaged` gain it, because the per-repo opt-in cannot be resolved without
  it; `Capture` and `Drain` already had it.
- **Turning the opt-in on does not move a corpus that already exists.** The
  migration is from the legacy location into the resolved store, not between the
  two current lanes, so a repo declared after it has been capturing has records
  in the user-level store and new records in its own tree. This is the accepted
  cost of a declaration that is a location choice rather than a migration
  command.
- **The store can now sit inside a repository tree**, which the cold-reading
  exclusion floor has to enforce rather than assume: its transcript-store row
  states a `directory` exclusion on `.abcd/.work.local/transcripts` instead of
  asserting that the store is unreachable, per
  [adr-56](0056-an-exclusion-control-asserts-only-what-it-can-prove.md).
- **ADR-29 is superseded rather than corrected.** Its native, redacted-on-write,
  root-SHA-keyed and never-committed properties are carried here unchanged; only
  the location it inherited does not survive. It is retained rather than pruned,
  because the brief and the adapter table still cite it.

## Status note

**Accepted by the maintainer's ruling** that transcripts go into
`~/.abcd/transcripts/` by default and that the per-repo location becomes an
opt-in pull. It is a new record rather than an edit to ADR-29 by the maintainer's
standing instruction that an ADR is never amended and always superseded, so
ADR-29's decision text stands as written and this record carries the change.
