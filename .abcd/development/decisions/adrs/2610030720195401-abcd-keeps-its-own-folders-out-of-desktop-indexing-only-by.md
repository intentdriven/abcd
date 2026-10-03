---
id: adr-2610030720195401
slug: abcd-keeps-its-own-folders-out-of-desktop-indexing-only-by
status: proposed
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610030720038073]
related_rfcs: []
related_adrs: []
---

# ADR-2610030720195401: abcd keeps its own folders out of desktop indexing only by changing those folders, never the computer's search settings

## Context

abcd's home under the person's home folder holds transient stores (lane worktrees, run logs, transcripts, caches) that the desktop indexer reads as ordinary user files. On 2026-10-02 a burst of new worktrees drove the machine's load to 148 through indexing alone (itd-2610030720038073). Keeping those folders out of the index is wanted by default. The computer's own search settings belong to the person: changing them needs administrator rights and reaches beyond abcd's declared space, which the principle `the-users-directory-is-theirs` keeps abcd out of.

## Decision

abcd keeps a folder out of desktop indexing only by changing that folder itself, inside the space abcd already owns (a marker or naming the indexer honours). It never edits the computer's search settings or privacy list, never asks for administrator rights to do so, and never touches a folder outside its own space. A person who wants a system-wide exclusion sets it themselves. Routed by the itd-84 decomposition and confirmed by the product thinker on 2026-10-03; it stays `proposed` until the intent's method is settled at planning.

## Alternatives Considered

- Change only abcd's own folders (chosen): needs no extra rights, is undone by deleting the marker, and stays inside abcd's declared space.
- Add abcd's home to the system search privacy list: excludes reliably, but needs administrator rights and edits a setting that is the person's, across every program.
- Turn indexing off for the whole volume: far beyond what is needed, and the person's search stops working.
- Do nothing and document a manual step: leaves the load burst as the default, the outcome the intent exists to remove.

## Consequences

The method is limited to what the indexer honours per folder; if no per-folder method works on a given OS version, abcd says so loudly rather than escalating to system settings. A brief invariant stating the rule is owed in the change that ships itd-2610030720038073.
