# `/abcd:ingest` — Register a Source in the Corpus

`/abcd:ingest` registers a URL or local document in the personal sources corpus
(the user-level home's sources store, default `~/.abcd/sources/`), with extracted
reference metadata, real keywords, a confidentiality class, and a text-quality
check. It is the write side of the corpus; `/abcd:consult` is the read side and
the provenance recorder.

It is a **host-delegated command** — a markdown workflow that runs in the host
agent. **No Go verb backs it**: there is no top-level `abcd ingest` verb, no
bare-status render, and no CLI flags of its own. (The `reading ingest`,
`memory ingest` and `intent audit ingest` sub-verbs belong to other verbs and
validate other inputs — a reading's returned output, a source distilled into
memory, an intent-audit verdict — never this corpus.) The determinism it relies on
lives in the corpus's own `bin/add-source` registrar; the command supplies the
judgment half (clean metadata, real keywords, confidentiality, quality check).

## What it does

The corpus is split by a division of labour: the `add-source` script does the
deterministic half (fetch, convert, store, guard the ledger); the agent does the
judgment half. `/abcd:ingest` drives that agent side through five steps:

1. **Read the document first** — WebFetch a URL (or Read a local file) for
   judgment, then extract the exact title, authors (`Family, Given`), year, venue,
   canonical URL, and CSL type (`article-journal`, `webpage`, `book`, `report`,
   `motion_picture`).
2. **Decide class and key** — web content is `public` by default; the signals for
   `--confidential` are the user's own unpublished work, internal or NDA material,
   AI-generated content, or a private repo's documentation. The key is
   `<authorfamily><year><distinctiveword>`, uniqueness checked against
   `sources.json`.
3. **Register** — invoke `~/.abcd/sources/bin/add-source` with the extracted flags;
   the URL-only path lets the script fetch and store the page.
4. **Quality-check the extraction** — inspect the stored `text.md` for sane word
   count and real prose, repairing the known failure modes (`.mhtml` stubs,
   HTML-escaped SPA wrappers) by hand.
5. **Close out** — sync the ban-list into every guarded repo the session touched,
   keep any derived summary inside the source's own folder, record an influence
   edge if a live decision motivated the ingest, and tell the user the key and
   class.

## Confidentiality contract

The confidentiality hard rule from `/abcd:consult` applies here in full. Titles of
confidential entries become banned phrases (whole title, whitespace-flexible), so
an internal artifact whose natural title reads like ordinary prose is registered
under a distinctive title instead. A confidential ingest ends by running
`bin/sync-banlist <repo-root>` in every guarded repo, so the new banned phrases
propagate before anything else is written.

## Acceptance

- **Given** a present corpus, **when** the user hands over a public URL, **then**
  a new entry is registered under a `<authorfamily><year>word` key with clean
  metadata, 5–10 real keywords, and a stored `text.md` that passes the
  word-count/prose check.
- **Given** the user's own unpublished paper, **when** it is ingested, **then**
  it is registered `--confidential` with a permission status and its title added
  to the ban-list, and `sync-banlist` is run in each guarded repo.
- **Given** a corpus that does not exist, **when** `/abcd:ingest` is invoked,
  **then** the command says so and stops — it never creates the corpus.

## Composition

`/abcd:ingest` and `/abcd:consult` are the two halves of one corpus surface:
ingest writes sources in, consult reads them out and records which decisions they
influenced. Both share the confidentiality guard and the `~/.abcd/sources/` store.
The command prefers explicit `add-source` flags because it has better metadata
in hand. (The `abcd-ingest <url-or-file>` human quick path `commands/ingest.md`
names is **not a demonstrable surface** — no binary sub-verb, repo-shipped
script, or corpus-store wrapper provides it; where it exists it is an
operator-local convenience outside the corpus contract.)

## References

- Plugin command: [`commands/ingest.md`](../../../../commands/ingest.md)
- Read side of the same corpus: [`13-consult.md`](13-consult.md)
- Corpus contract: `~/.abcd/sources/README.md`
