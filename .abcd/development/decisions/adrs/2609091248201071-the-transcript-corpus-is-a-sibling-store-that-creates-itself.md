---
id: adr-2609091248201071
slug: the-transcript-corpus-is-a-sibling-store-that-creates-itself
status: accepted
date: 2026-09-09
supersedes: adr-2609090717039680
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: [adr-22, adr-29, adr-56, adr-2609090717039680, adr-2609091248200336]
---

# ADR-2609091248201071: The transcript corpus is a sibling store that creates itself through the canonical directory primitive

## Context

[ADR-29](0029-native-transcript-corpus.md) settled that abcd captures its own
session transcripts into a native local store — no external tool, keyed on the
repo's root-commit SHA, never committed, redacted at write time — and
[adr-2609090717039680](2609090717039680-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md)
settled where on the machine that store sits and how it comes to exist: a
sibling of ahoy's registry at `~/.abcd/transcripts/<root-sha>/`, created by
the store itself through one seam, with a per-repo location the caller opts
into from their own home and a loud migration from the location the store
inherited. Every one of those properties still holds, and none of them is
reopened here.

What that record got wrong is the name and the home of one mechanism. It
names `ensureRealDir` as a helper of `internal/core/history` and presents its
sequence — walk the chain top-down, create each level with `os.Mkdir` rather
than `MkdirAll`, re-verify every level as a real directory on every resolve —
as a property of that package. On 2026-09-09 (`24c2f2e3`) three copies of that
sequence stood in the tree, in `history`, `intent` and `lifeboat`, and
[`one-canonical-primitive`](../../principles/one-canonical-primitive.md)
forbids the third by name; the branch that added `history`'s had made three
before consolidating
([iss-2609091128479544](../../../work/issues/resolved/iss-2609091128479544-a-third-copy-of-the-real-dir-primitive-which-the-principles-forbid.md)).
The consolidation moved the sequence to `internal/fsutil` as
`EnsureRealDir(dir, perm)`, one level created with a non-following `Mkdir`
and then proved a real directory, and `EnsureRealDirAll(base, rel, perm)`,
that step walked over a chain under a trusted root and proving every level as
it creates it. The mode became a parameter because the copies legitimately
differed on it, and the weakest copy — `intent`'s, which lstat'd the leaf and
then called `MkdirAll` through a symlinked ancestor — landed on the strongest
guarantee.

The behaviour the record describes is therefore unchanged and the symbol is
gone: a reader following adr-2609090717039680 to the code finds nothing at the
name it gives
([iss-2609091155525689](../../../work/issues/resolved/iss-2609091155525689-two-adrs-describe-the-real-dir-helper-at-its-pre-consolidation-home.md)).
An ADR is never amended, always superseded, so the correction is this record,
which carries the original's decision whole and names the primitive where the
original named a helper.

## Decision

**The transcript corpus is a sibling of ahoy's registry, not a sub-tree of it.**
It lives at `~/.abcd/transcripts/<root-sha>/{records,staging}/`, and
`~/.abcd/history/` keeps exactly what ahoy owns: the index and the per-repo
metadata. `internal/core/history` is the only package that lays out or judges
the corpus path — brief invariant 15, held by
`internal/core/history/store_boundary_test.go` rather than promised.

**The store creates itself through one seam, and the create-then-prove step
is the canonical primitive's, not the store's.** `history.Resolve(repoRoot,
rootSHA)` is the single door every read and write path goes through, so the
store cannot be reached by a route that skipped the creation, the symlink
check or the migration. `Resolve` owns the layout: it builds the chain and
walks it top-down on every call, and at each level it calls
`fsutil.EnsureRealDir` with the store's own mode. The primitive creates that
one level with a single non-following `os.Mkdir` — never `MkdirAll`, which
creates a whole chain without judging any of it — and then proves the result
is a real directory rather than a symlink or a file. Because the walk runs on
every resolve, a parent swapped between calls is caught before the leaf is
opened. What `ownedDirsReal` was protecting was never absence: it was creating
or writing *through* a planted symlink, and that refusal is kept, made
explicit, and made once. The store keeps its typed refusal — a fault from the
primitive is mapped onto `*StorePathError` by a shaping function that
contains no directory logic — and it keeps its mode: `storeDirPerm` is
`0o700`, because records are redacted but still a verbatim account of the
caller's sessions and staging holds them unredacted, and the primitive never
widens or narrows a directory that already exists. Creation needs no authority
the caller does not already hold, since every level is under their own home.
The boundary is the point: `internal/core/history` decides which directories
exist and at what mode, and `internal/fsutil` decides how a directory is
created and proved. A store that wants this discipline calls the primitive; it
does not reproduce it.

**The key stays the repo's root-commit SHA, as a directory.** One user-level
store still has to answer "this repo's transcripts", and the root commit is
the one name that survives what a checkout does: it moves, it is renamed, it
is cloned twice on one machine, and its root commit changes under none of
that. The key being a directory rather than a field means `list`, `show`,
`staged` and `drain` each read one repo's lane, and no cross-repo filter
exists to get wrong.

**The per-repo location is an opt-in pull the caller declares in their own
home.** `~/.abcd/local-transcript-roots` takes one absolute checkout path per
line with `#` comments, matched both as written and symlink-resolved, and is
honoured only while it is a regular file this uid owns that no one else can
write. A declared checkout keeps its store at
`<repo>/.abcd/.work.local/transcripts/<root-sha>/`, in the gitignored,
per-worktree local tier, so a pulled-in transcript is never a commit candidate
and never merge-conflicts between concurrent sessions. The declaration is
home-scoped on the `~/.abcd/path-entry` and `~/.abcd/trusted-roots` idiom, and
for the reason those give: a file inside the checkout would let a cloned repo
redirect the machine's transcripts into its own working tree, and an
environment variable clears the letter of that bar and not its spirit, because
a repo can ship the shell or task-runner configuration that sets it. A
declaration that is present but not honoured says so on stderr, since an
ignored opt-in and one never written are otherwise the same silence.

**A corpus at the earlier location is moved, loudly.** The first resolve moves
every record and staged file out of
`~/.abcd/history/<root-sha>/{transcripts,staging}/` into the store, file by
file rather than by directory rename, which is idempotent under a concurrent
peer and correct across filesystems. Nothing is deleted except a source whose
bytes are already at the destination; a non-regular entry or a name collision
with different content is left where it is and counted. `Resolve` returns
notes the way `rules.Resolve` does and every front door prints them on stderr,
and a `transcripts.moved` tombstone at the old path names the new one, withheld
while anything is left behind so the next resolve looks again.

## Alternatives Considered

The original's four alternatives on location are carried as decided:

1. **Create the directory where it stood, under `~/.abcd/history/`.**
   Rejected: a capture hook laying out a directory inside ahoy's install-owned
   namespace is the arrangement invariant 15 exists to forbid.
2. **Read both locations, and write to the new one.** Rejected: two stores
   diverging from the first capture onward, merged on every read forever.
3. **Refuse, and name the remedy.** Rejected: a machine that *had* installed
   would stop capturing until someone ran the remedy, and a shutdown hook is
   where nobody reads the message.
4. **Chosen: a sibling store, self-creating through one seam, with a
   home-declared per-repo opt-in and a move-loudly migration.**

On the correction this record exists for:

5. **Leave adr-2609090717039680 standing and let the code comment redirect
   the reader.** Rejected: the record is what a builder reads first, and a
   decision that names a symbol should not outlive the symbol. A pointer in
   the code answers only the reader who already found the code.
6. **Describe the mechanism and name no symbol at all.** Rejected: that is the
   reading the original invited, and under it a fourth copy is a faithful
   implementation of the record. The principle's whole point is that the
   canonical home is discoverable at the moment of temptation, so the record
   names it.
7. **Chosen: carry the decision whole, name `fsutil.EnsureRealDir` as the
   create-then-prove step, and draw the boundary between the store's layout
   and the primitive's mechanism.**

## Consequences

- **adr-2609090717039680 is superseded and retained.** The brief's ahoy and
  configuration chapters, `store_boundary_test.go`, ADR-29's own
  `superseded_by` and a draft intent cite it, so it stays with both halves of
  the supersession in its frontmatter and its decision text as written. The
  chain is ADR-29, superseded by adr-2609090717039680, superseded by this
  record; ADR-29's link is untouched, because a superseding record links
  backwards and never rewrites what it did not decide.
- **Every consequence the original drew stands.** Capture has no
  install-shaped precondition; the `history.transcripts_missing` gap and its
  SessionStart notice are retired; `ahoy install` opens the store rather than
  laying it out; the read verbs take the repo root; turning the opt-in on does
  not move a corpus that already exists; and the store can sit inside a
  repository tree, which the cold-reading exclusion floor states as a
  `directory` exclusion rather than assuming
  ([adr-56](0056-an-exclusion-control-asserts-only-what-it-can-prove.md)).
- **A private copy of the sequence is refused, by name.**
  `TestNoNonCanonicalAtomicWritePrimitives` scans every package under
  `internal/` other than `fsutil` for a lowercase `ensureRealDir`, a spelling
  its detector did not carry while the three copies accumulated. A store that
  wants create-then-prove calls `fsutil.EnsureRealDir` or `EnsureRealDirAll`
  and carries only its mode and its error wording.
- **The worktree store inherits the same call.**
  [adr-2609091248200336](2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md)
  commits it to creating itself "as the transcript store does", and this
  record is what that phrase now resolves to.
- **No gate reads decision prose against Go identifiers.** An ADR that names a
  package-local symbol survives that symbol's removal unnoticed, which is how
  the original stood for a day describing a helper that was already gone. The
  record-lint rule family has no cross-reference from a record to the tree;
  the check is a reviewer's until one exists.

## Status note

**Accepted by the maintainer's ruling of 2026-09-09** on iss-2609091155525689
that each original record describing the helper at its pre-consolidation home
is superseded by its own record, recorded in
[`.abcd/work/DECISIONS.md`](../../../work/DECISIONS.md). It is a new record
rather than an edit to adr-2609090717039680 by the standing instruction that
an ADR is never amended and always superseded, so the original's decision text
stands as written and this record carries the correction.
