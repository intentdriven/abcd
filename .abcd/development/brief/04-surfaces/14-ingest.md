# `/abcd:ingest` — Register a Source in the Corpus

Hand over a link or a document and get it into the corpus as a real, findable
source: clean reference metadata, keywords worth searching on, a
confidentiality class, and a stored text body someone can actually read later.
The cost is a few judgement calls the person handing it over is best placed to
make, and the payoff is that [`/abcd:consult`](13-consult.md) can find the
source months later without anyone remembering it exists.

It is the write side of the corpus at `~/.abcd/sources/`; `/abcd:consult` is
the read side and the provenance recorder.

It is a **host-delegated command**: a markdown workflow that runs in the host
agent, with **no Go verb** behind it. There is no top-level `abcd ingest` verb,
no bare-status render, and no CLI flags of its own. Every ingest path the binary
does have belongs to another verb, validates that verb's own input and never
writes this corpus; the generated CLI reference lists them, so this chapter
names none of them.


**Typing it at the CLI gets a second line that misdirects.** `abcd ingest` exits
on an unknown command, and because a command page of that name exists, the binary
adds its stale-surface note, reading that page as proof a newer build carries the
verb and telling the person to rebuild or update. For a host-delegated command
that advice can never come true, because there is no Go verb for a rebuild to
bring in. Every host-delegated page has the same shape, `/abcd:consult` and
`/abcd:prepare-this-repo` alongside this one. What the note should say is that the
command runs in the host agent rather than at the CLI.

## What it does

The work is split. The corpus's own registrar script does the deterministic
half: fetch, convert, store, guard the ledger. The command supplies the
judgement half, in five steps.

1. **Read the document first**, then extract the exact title, authors
   (`Family, Given`), year, venue, canonical URL, and CSL type.
2. **Decide class and key.** Web content is public by default; the signals for
   confidential are the user's own unpublished work, internal or NDA material,
   AI-generated content, and a private repo's documentation. The key is
   `<authorfamily><year><distinctiveword>`, checked for uniqueness against the
   corpus metadata.
3. **Register** through the corpus registrar. A URL alone is enough: the script
   fetches and stores the page.
4. **Quality-check the extraction.** Inspect the stored text for a sane word
   count and real prose, and repair the known failure modes by hand rather than
   leaving a stub that reads as a source.
5. **Close out.** Sync the ban-list into every guarded repo the session
   touched, keep any derived summary inside the source's own folder so it
   inherits the class by location, record an influence edge if a live decision
   motivated the ingest, and tell the user the key and class.

If the corpus is absent, the command says so and stops. It never creates it.

## Confidentiality contract

The hard rule from [`/abcd:consult`](13-consult.md) applies here in full. The
title of a confidential entry becomes a banned phrase, whitespace-flexible over
the whole title, so an internal artefact whose natural title reads like ordinary
prose is registered under a distinctive title instead. A confidential ingest
ends by syncing the ban-list in every guarded repo, so the new banned phrases
propagate before anything else is written.

## Acceptance

- **Given** a present corpus, **when** the user hands over a public URL,
  **then** a new entry is registered under a `<authorfamily><year>word` key with
  clean metadata, 5–10 real keywords, and a stored text body that passes the
  word-count and prose check.
- **Given** the user's own unpublished paper, **when** it is ingested, **then**
  it is registered confidential with a permission status and its title added to
  the ban-list, and the ban-list is synced in each guarded repo.
- **Given** a corpus that does not exist, **when** `/abcd:ingest` is invoked,
  **then** the command says so and stops.

## Composition

`/abcd:ingest` and `/abcd:consult` are the two halves of one corpus surface:
ingest writes sources in, consult reads them out and records which decisions
they influenced. Both share the confidentiality guard and the store.

The command prefers explicit registrar flags because it has better metadata in
hand than a bare fetch would. There is **no one-argument quick path** into the
registrar: no binary sub-verb and no repo-shipped script provides one, so where
a reader finds such a command it is an operator-local convenience outside the
corpus contract.

## References

- Plugin command: [`commands/ingest.md`](../../../../commands/ingest.md)
- Read side of the same corpus: [`13-consult.md`](13-consult.md)
- Corpus contract: `~/.abcd/sources/README.md`

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

There is no shipped surface: the command tree registers no `abcd ingest` verb, so there are no flags and no sub-verbs to list.

<!-- surface-appendix:end -->
