---
id: adr-2610031751065746
slug: the-worktree-store-lives-under-the-renamed-home-abcd-noindex
status: accepted
date: 2026-10-03
supersedes: adr-2609091248200336
superseded_by: null
related_intents: [itd-2610030720038073, itd-2609091014076309]
related_rfcs: []
related_adrs: [adr-2609091248200336, adr-2610030720195401, adr-2610031751066232]
---

# ADR-2610031751065746: The worktree store lives under the renamed home, `~/.abcd.noindex/worktrees/<root-sha>/<name>/`, and the rest of its rule is unchanged

## Context

[adr-2609091248200336](2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md)
binds where a session's or an agent's worktree goes, in its location half:
`~/.abcd/worktrees/<root-sha>/<name>/`, keyed on the repository's root-commit
SHA, under the caller's own abcd home. Its trust rule (a tool never creates a
directory in space the user did not hand it), its verb half (the store's
verbs bind when itd-2609091014076309 ships), its create-then-prove call to
`fsutil.EnsureRealDir` and `EnsureRealDirAll`, and its list-and-reclaim
obligation are not in question here.

[itd-2610030720038073](../../intents/shipped/itd-2610030720038073-abcd-keeps-its-home-folder-out-of-desktop-search-by-default.md)
renames that home. On 2026-10-02 eight lane worktrees opened in the store set
off a desktop indexing burst (load 110 to 148 with no test running, the
indexer's processes named in the run log), and the product thinker ruled on
2026-10-03 that the whole home is renamed `~/.abcd` to `~/.abcd.noindex`, a
name the macOS indexer passes over when it scans (the intent's decisions 1
and 2). abcd moves nothing: an existing `~/.abcd` stops every command and
hook until the person renames it (decision 6). The intent's decision 4 names
the location adr-2609091248200336 binds as one of the reversals that rename
forces, and requires a superseding record for it rather than an amended one,
because an ADR is never amended. This is that record.

[adr-2610030720195401](2610030720195401-abcd-keeps-its-own-folders-out-of-desktop-indexing-only-by.md)
is the rule the rename stands on: abcd keeps its folders out of indexing only
by changing its own folder, never the computer's search settings.

## Decision

**The worktree store's location is `~/.abcd.noindex/worktrees/<root-sha>/<name>/`.**
A session's or an agent's worktree goes there, keyed on the repository's
root-commit SHA as before. Every spelling adr-2609091248200336 gives of the
caller's abcd home, the store's root and the `trusted-roots` declaration it
cites on the read side, reads `~/.abcd.noindex/` in place of `~/.abcd/`.

**Nothing else of adr-2609091248200336 changes.** It is carried forward whole:
a tool never creates a directory in user-owned project space; agent and session
scratch is machine-scoped and keyed on the root-commit SHA; the store creates
itself through `fsutil.EnsureRealDir` and `EnsureRealDirAll`, owning its layout
and its mode and nothing else about directory creation; what a tool puts in
its own space it can list and reclaim; a directory the user made is theirs even
inside the store; and the rule binds an agent's conventions in two halves, the
location now and the verb when the store ships. The location half binds at the
new spelling from the change that renames the home.

**The texts this record binds** are the ones that spell the store's location,
and each changes in the change that makes this record true:

- `AGENTS.md`, § Concurrent sessions (the store's path), and the
  `trusted-roots` re-admission command in the rule-loader section;
- the principle
  [`the-users-directory-is-theirs`](../../principles/the-users-directory-is-theirs.md),
  its Enforcement paragraph;
- the brief's § The worktree store (`05-internals/03-configuration.md`) and
  the build surface chapter (`04-surfaces/34-build.md`);
- `commands/build.md` and `commands/implement.md`, which name the lane
  worktree's path;
- the bundled rules' scratch rule (`internal/core/rules/defaults/rules.json`);
- the path spelling in
  [itd-2609091014076309](../../intents/planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md)
  and its open spec spc-2609301811532881, each of which gains a dated decision
  line naming this record and changes nothing else.

**It stays `proposed` until the change that makes it true.** That change,
step 3 of spc-2610031309233367, renames the home, accepts this record, writes
`supersedes: adr-2609091248200336` here and `superseded_by` with
`status: superseded` there, and changes the texts above that it carries; step 4
changes the rest. Until then adr-2609091248200336 is in force and its spelling
is the shipped one.

## Alternatives Considered

1. **Amend adr-2609091248200336's location in place.** Rejected: an ADR is
   never amended, always superseded, and the intent's decision 4 says so for
   this location by name.
2. **Rename only the worktree store, `~/.abcd/worktrees.noindex/`, and leave
   the home's name alone.** Rejected by the intent's decision 2: the whole home
   is renamed, because the home's other stores (the lab, transcripts, sources,
   run logs) cost the indexer the same scan.
3. **Leave the location's text naming `~/.abcd/` and let the reader infer the
   rename.** Rejected: the location half binds a convention an agent follows by
   reading it, and a convention naming a folder that stops abcd sends every
   session's worktree into the folder the stop refuses.
4. **Chosen: a successor that changes only the spelling and carries the rest
   of the rule forward unchanged.**

## Consequences

- **adr-2609091248200336 is superseded and retained once this record is
  accepted.** `AGENTS.md`, the principle, the brief and two intents cite it,
  so it keeps its decision text as written, with both halves of the
  supersession in its frontmatter; its own supersession of
  adr-2609091014087993 is untouched.
- **A worktree made under `~/.abcd/worktrees/` before the rename moves with
  the folder.** The person renames the whole home, so its worktree lane
  arrives at the new spelling with it; git's own records of where each
  worktree lives are the person's to repair with `git worktree repair`, as for
  any moved checkout. abcd moves and repairs nothing (the intent's decision 6).
- **The store's verbs land at the new spelling.** spc-2609301811532881 builds
  `abcd ahoy worktree` from the store root; whichever of its step 1 and the
  home resolver lands second takes the root from the resolver.
- **The transcript store's spelling is its own record,**
  [adr-2610031751066232](2610031751066232-the-transcript-store-is-spelled-under-the-renamed-home-abcd.md),
  because adr-2609091248201071 is a separate original with its own
  `superseded_by`.
