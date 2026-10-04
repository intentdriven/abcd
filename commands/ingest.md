---
name: ingest
description: Ingest a URL or document into the local sources corpus (the user-level home's sources store, default ~/.abcd.noindex/sources) with extracted reference metadata, keywords, and a text-quality check. Use when the user says "ingest this", "add this source/URL/paper to the corpus", "register this source", or hands over a document/link to be stored. For consulting the corpus or recording provenance, use /abcd:consult.
argument-hint: <url-or-file>
---

# Ingest a source

Register a URL or local document in the corpus at `~/.abcd.noindex/sources/`. The
`abcd source add` verb does the deterministic half (store, classify, commit);
you do the judgment half (fetching, converting, clean metadata, real keywords,
confidentiality, quality check). abcd fetches and converts nothing. The
confidentiality hard rule from `/abcd:consult` applies here in full. First run
`"${CLAUDE_PLUGIN_ROOT}/abcd" source --json`: exit 3 means there is no corpus —
say so and stop, because this command never creates it.

## 1. Read the document first

- URL: fetch it (metadata and the content). Save what you will store — the
  page or document, and its text as Markdown — to a scratch file; the verb
  stores files you hand it and never fetches.
- Local file: Read it (PDFs via the Read tool; big files: first pages suffice).

Extract: exact title (no site suffixes), authors (each as `Family, Given`),
publication year, venue, canonical URL. Map the type to CSL: `article-journal`
(papers), `webpage` (posts/docs), `book`, `report` (white papers, internal
docs), `motion_picture` (video).

Two rules hold for author names:

- **Resolve a name before caveating it.** An author given by initials only is
  looked up through the entry's own DOI or URL, which usually resolves the
  full name in one fetch. A "to be checked" caveat is for a fact that is
  genuinely unreachable, never for a lookup that was skipped.
- **Take names from the publisher's current record.** A name that has changed
  since publication is recorded as the publisher now gives it, never reverted
  to a former name found in an older copy, a citation elsewhere, or an index.

## 2. Decide class and key

- **Class.** Web content is `public` by default. Signals for
  `--confidential`: the user says so; it is their own unpublished/submitted
  work; internal or NDA material; AI-generated content (never citable); a
  private repo's documentation. When confidential, set `abcd mode
  product-thinker` or `abcd mode facilitator` for whichever of the product
  thinker or the technical facilitator is adding the source, then ask them for
  every identifying name variant (aliases — repo names, codenames, domains) and
  pick `--permission`: `no-public-citation`, `internal-never-cite`,
  `ai-generated-never-cite`, or `ask-author`. The class is declared once, here;
  `add` refuses to guess it.
- **Titles and aliases of confidential entries become banned phrases** (whole
  phrase, case-insensitive, whitespace-flexible), and each needs at least three
  letters or digits. For internal artifacts whose natural title reads like
  normal prose, register a distinctive title instead — e.g.
  "meeting-notes-2026-07 (internal)" — so the ban cannot trip on legitimate
  text. Authors are banned only with `--ban-authors`, for a source whose
  authorship is itself identifying.
- **Key.** Public: `<authorfamily><year><distinctiveword>`, lowercase ASCII
  (e.g. `naur1985theory`). Confidential: an **opaque** key (e.g. `conf2026a`) —
  the key is the one handle every refusal and scan prints, and the verb refuses
  one that contains an alias, the title or a longer title word, or an author's
  name. A duplicate key is refused.

## 3. Register

```sh
"${CLAUDE_PLUGIN_ROOT}/abcd" source add [file] --key <key> --confidential|--public \
  --type <csl-type> [--year YYYY] [--venue <venue>] [--url <url>] \
  [--permission <status>] [--ban-authors] [--text <extracted.md>] \
  --meta <meta.json> --json
```

- `meta.json` carries the identifying strings, so they stay out of argv and
  shell history: `{"title": "…", "aliases": ["…"], "author": [{"family": "…",
  "given": "…"}], "keywords": ["…"]}`. For a public source `--title`,
  `--author "Family, Given"`, `--alias` and `--keywords` work as flags too.
- A `.md` or `.txt` file is its own text. Any other file (a PDF, a saved page)
  needs `--text` with the Markdown you extracted. A URL with no file registers
  a metadata stub.
- **Keywords are the retrieval surface** — write 5–10 from having actually
  read the piece: topics, named tools/techniques, the claims it makes. Never
  generic filler ("AI", "software").

There is no one-argument quick path, and none is assumed here: `source add`
with explicit flags is the registrar's front door, and the better one anyway,
since you have just read the source and hold metadata a bare fetch could not
recover.

## 4. Quality-check the extraction

Check `~/.abcd.noindex/sources/<class>/<key>/text.md` — word count sane, real prose
present. Known failure modes: `.mhtml` (unsupported → stub; extract by hand),
saved SPA/artifact pages whose content sits HTML-escaped in a wrapper
(unescape entities, `pandoc -t gfm`, rebuild text.md below its frontmatter,
commit in the corpus repo — a hand repair to a stored file is the one write
the verb does not make). If the source is webloc/link-only there is no body
— a metadata+URL stub is correct.

## 5. Close out

- Confidential ingest → run `"${CLAUDE_PLUGIN_ROOT}/abcd" source sync-banlist`
  in every guarded repo the session touches (a guard that refreshes the block
  does so on the next commit, but only a store that already exists).
- If the user wants a summary or review kept: write it to the source's own
  folder (`summary.md`, notes as siblings) — derived artifacts inherit the
  source's class by location, never anywhere else.
- If the ingest was motivated by a live decision, record the influence edge in
  the ledger per `/abcd:consult` (`abcd source ledger`).
- Tell the user the key and class you registered.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
