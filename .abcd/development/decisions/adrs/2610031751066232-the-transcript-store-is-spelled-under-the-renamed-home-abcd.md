---
id: adr-2610031751066232
slug: the-transcript-store-is-spelled-under-the-renamed-home-abcd
status: proposed
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610030720038073]
related_rfcs: []
related_adrs: [adr-2609091248201071, adr-2610030720195401, adr-2610031751065746]
---

# ADR-2610031751066232: The transcript store is spelled under the renamed home, `~/.abcd.noindex/transcripts/<root-sha>/`, and the rest of its decision is unchanged

## Context

[adr-2609091248201071](2609091248201071-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md)
settles where the session-transcript store sits and how it comes to exist: a
sibling of ahoy's registry at `~/.abcd/transcripts/<root-sha>/{records,staging}/`,
created by `history.Resolve` through `fsutil.EnsureRealDir`, with a per-repo
location the caller opts into by declaring a checkout in
`~/.abcd/local-transcript-roots`, and a loud move of a corpus left at the
earlier location under `~/.abcd/history/<root-sha>/`. Brief invariant 15
(`02-constraints/03-invariants.md`) cites it for that single spelling, and
`TestOnlyTheHistoryPackageNamesTheStorePath`
(`internal/core/history/store_boundary_test.go`) holds every other package to
it.

[itd-2610030720038073](../../intents/planned/itd-2610030720038073-abcd-keeps-its-home-folder-out-of-desktop-search-by-default.md)
renames the home those spellings sit in. The product thinker ruled on
2026-10-03 that the whole home is renamed `~/.abcd` to `~/.abcd.noindex`, a
name the macOS indexer passes over when it scans (the intent's decisions 1
and 2), and that abcd moves nothing: an existing `~/.abcd` stops every command
and hook until the person renames it (decision 6). The intent's decision 4
names the transcript store's single spelling under invariant 15 as one of the
reversals that rename forces, and requires a superseding record for it rather
than an amended one, because an ADR is never amended. This is that record.

[adr-2610030720195401](2610030720195401-abcd-keeps-its-own-folders-out-of-desktop-indexing-only-by.md)
is the rule the rename stands on: abcd keeps its folders out of indexing only
by changing its own folder, never the computer's search settings.

## Decision

**The transcript store is spelled `~/.abcd.noindex/transcripts/<root-sha>/{records,staging}/`,
and the declaration that opts a checkout into the per-repo location is
`~/.abcd.noindex/local-transcript-roots`.** Every other spelling
adr-2609091248201071 gives of the caller's abcd home reads `~/.abcd.noindex/`
in place of `~/.abcd/`: ahoy's registry at `~/.abcd.noindex/history/`, the
earlier location a corpus is moved out of at
`~/.abcd.noindex/history/<root-sha>/{transcripts,staging}/`, and the
`path-entry` and `trusted-roots` declarations whose idiom the opt-in follows.

**Nothing else of adr-2609091248201071 changes.** It is carried forward whole:
the corpus is a sibling of ahoy's registry, not a sub-tree of it;
`internal/core/history` is the only package that lays out or judges the
store's path; `history.Resolve` is the one door, walking the chain on every
call through `fsutil.EnsureRealDir` at the store's mode of `0o700`; the key is
the repository's root-commit SHA, as a directory; the per-repo location is an
opt-in pull declared in the caller's own home, honoured only while the
declaration is a regular file this account owns that no one else can write,
and keeps the store at `<repo>/.abcd/.work.local/transcripts/<root-sha>/`,
whose repository-tier spelling this record does not touch; and a corpus at the
earlier location is moved loudly, file by file, behind a tombstone.

**The texts this record binds** are the ones that spell the store or its
declaration, and each changes in the change that makes this record true:

- brief invariant 15 (`02-constraints/03-invariants.md`), its parenthesis
  naming the store and the declaration;
- `AGENTS.md`, the working-tree layout's `.abcd/.work.local/` entry, which
  names the declaration and the user-level store;
- the history surface chapter (`04-surfaces/11-history.md`) and the
  configuration chapter's § The transcript corpus
  (`05-internals/03-configuration.md`);
- `commands/history.md` and `docs/how-to/install.md`.

Invariant 15's test changes shape in the home resolver's step, not here: its
needles keep the old literal spellings and gain the new one, and the invariant
it holds (one package touches the store's path) is unchanged.

**It stays `proposed` until the change that makes it true.** That change,
step 3 of spc-2610031309233367, renames the home, accepts this record, writes
`supersedes: adr-2609091248201071` here and `superseded_by` with
`status: superseded` there, and changes invariant 15 and `AGENTS.md` in the
same diff; step 4 changes the rest. Until then adr-2609091248201071 is in
force and its spelling is the shipped one.

## Alternatives Considered

1. **Amend adr-2609091248201071's spelling in place.** Rejected: an ADR is
   never amended, always superseded, and the intent's decision 4 says so for
   this spelling by name.
2. **Keep the transcript store at `~/.abcd/transcripts/` while the rest of
   the home moves.** Rejected by the intent's decisions 1 and 2: the whole
   home is renamed, and the transcript store, about 3.2 GB on the product
   thinker's machine, is among the stores the indexer scans. It would also
   leave a reader of the old folder, which decision 6's stop forbids.
3. **Read both spellings during a move.** Rejected by decision 6: abcd moves
   nothing and reads no fallback, so there is no window in which two names are
   read; the stop replaces it.
4. **Chosen: a successor that changes only the spelling and carries the rest
   of the decision forward unchanged.**

## Consequences

- **adr-2609091248201071 is superseded and retained once this record is
  accepted.** Invariant 15, the brief's history and configuration chapters,
  `store_boundary_test.go` and adr-2609091248200336 cite it, so it keeps its
  decision text as written, with both halves of the supersession in its
  frontmatter. The chain is ADR-29, superseded by adr-2609090717039680,
  superseded by adr-2609091248201071, superseded by this record; the earlier
  links are untouched.
- **A corpus and a declaration move with the folder.** The person renames the
  whole home, so the store's records, its staging, and an existing
  `local-transcript-roots` arrive at the new spelling with it; nothing in the
  store is rewritten, and the move-loudly migration finds a legacy corpus at
  its renamed earlier location as before.
- **The worktree store's location is its own record,**
  [adr-2610031751065746](2610031751065746-the-worktree-store-lives-under-the-renamed-home-abcd-noindex.md),
  because adr-2609091248200336 is a separate original with its own
  `superseded_by`.
