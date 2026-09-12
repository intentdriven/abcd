---
name: ingest
description: Ingest a URL or document into the local sources corpus (the user-level home's sources store, default ~/.abcd/sources) with extracted reference metadata, keywords, and a text-quality check. Use when the user says "ingest this", "add this source/URL/paper to the corpus", "register this source", or hands over a document/link to be stored. For consulting the corpus or recording provenance, use /abcd:consult.
argument-hint: <url-or-file>
---

# Ingest a source

Register a URL or local document in the corpus at `~/.abcd/sources/`. The
script does the deterministic half (fetch, convert, store, guard); you do the
judgment half (clean metadata, real keywords, confidentiality, quality check).
Corpus contract: `~/.abcd/sources/README.md`. The confidentiality hard rule
from `/abcd:consult` applies here in full. If the corpus is absent, say so and
stop — this command never creates it.

## 1. Read the document first

- URL: WebFetch it (metadata + a skim of the content). A local fetch for
  storage happens in step 3 regardless — WebFetch is for your judgment only.
- Local file: Read it (PDFs via the Read tool; big files: first pages suffice).

Extract: exact title (no site suffixes), authors (each as `Family, Given`),
publication year, venue, canonical URL. Map the type to CSL: `article-journal`
(papers), `webpage` (posts/docs), `book`, `report` (white papers, internal
docs), `motion_picture` (video).

## 2. Decide class and key

- **Class.** Web content is `public` by default. Signals for
  `--confidential`: the user says so; it is their own unpublished/submitted
  work; internal or NDA material; AI-generated content (never citable); a
  private repo's documentation. When confidential, ask the user for every
  identifying name variant (`--aliases` — repo names, codenames, domains) and
  pick `--permission`: `no-public-citation`, `internal-never-cite`,
  `ai-generated-never-cite`, or `ask-author`.
- **Titles of confidential entries become banned phrases** (whole title,
  whitespace-flexible). For internal artifacts whose natural title reads like
  normal prose, register a distinctive title instead — e.g.
  "meeting-notes-2026-07 (internal)" — so the ban cannot trip on legitimate
  text.
- **Key**: `<authorfamily><year><distinctiveword>`, lowercase ASCII (e.g.
  `naur1985theory`). Check uniqueness: `jq -r '.[].id' ~/.abcd/sources/sources.json`.

## 3. Register

```sh
~/.abcd/sources/bin/add-source --key <key> --title "<title>" \
  --type <csl-type> [--author "Family, Given"]... [--year YYYY] \
  --keywords "<k1, k2, ...>" [--aliases "a,b"] [--confidential] \
  [--permission <status>] [--url <url>] [file]
```

- URL only (no file): `--url` makes the script fetch and store the page.
- Local file + known URL: pass both; the URL is recorded, the file stored.
- **Keywords are the retrieval surface** — write 5–10 from having actually
  read the piece: topics, named tools/techniques, the claims it makes. Never
  generic filler ("AI", "software").

There is no one-argument quick path, and none is assumed here: `add-source` with
explicit flags is the registrar's front door, and the better one anyway, since
you have just read the source and hold metadata a bare fetch could not recover.

## 4. Quality-check the extraction

Check `~/.abcd/sources/<class>/<key>/text.md` — word count sane, real prose
present. Known failure modes: `.mhtml` (unsupported → stub; extract by hand),
saved SPA/artifact pages whose content sits HTML-escaped in a wrapper
(unescape entities, `pandoc -t gfm`, rebuild text.md below its frontmatter,
commit in the corpus repo). If the source is webloc/link-only there is no body
— a metadata+URL stub is correct.

## 5. Close out

- Confidential ingest → run `~/.abcd/sources/bin/sync-banlist <repo-root>` in
  every guarded repo the session touches.
- If the user wants a summary or review kept: write it to the source's own
  folder (`summary.md`, notes as siblings) — derived artifacts inherit the
  source's class by location, never anywhere else.
- If the ingest was motivated by a live decision, record the influence edge in
  the ledger per `/abcd:consult`.
- Tell the user the key and class you registered.
