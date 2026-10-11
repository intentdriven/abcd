---
name: library
description: "Render the sources corpus and the checkout's memory store, read-only: Writes nothing; refuses only when neither store can be read. Use when the user says 'ingest this', 'add this source/URL/paper to the corpus' or 'register this source' (that is library add), or 'consult sources', 'check the corpus' or 'what do my sources say'."
argument-hint: "[<empty>] | init | add [document] --key K [--private|--public] [...] | declassify <key> | ledger [--decision D --claim C --source K --influence I | --flip N | --list] | sync-banlist [--refresh] | cite-check <file|-> | ingest <path-or-https-url> [--keep-original] | ask <question> | lint"
block: people
---

# `/abcd:library` — the sources corpus and the memory store

Two stores under one command, never connected to each other:

- **The sources corpus**, at `~/.abcd.noindex/sources/` (another with
  `--corpus`), holds documents the user may consult but is not always free to
  cite: working papers, private notes, purchased reports. The folder a source
  sits in — `confidential/<key>/` or `public/<key>/` — is its classification.
  Its ledger records which source shaped which decision in this repository.
  Its verbs are `init`, `add`, `declassify`, `ledger`, `sync-banlist` and
  `cite-check`. You carry the judgement half — what to search for, reading a
  document before it is registered, its class and key, its keywords — and these
  verbs make every write.
- **The memory store**, at the checkout's `.abcd/memory/`, holds typed, cited
  pages distilled from a source. Its verbs are `ingest`, `ask` and `lint`.
  Nothing moves a corpus document into the memory store.

**Which verb registers a document.** `library add` registers a document in the
sources corpus. `library ingest` registers nothing: it distils a source into the
memory store's pages. When the user says "ingest this", "add this source" or
"register this paper", they mean `library add`, through the steps under
[Register a source](#register-a-source) below.

## The hard rule

Never write a confidential source's title, aliases, authors or any identifying
description into anything tracked by git or sent anywhere external: commits,
pull-request and issue text, docs, code comments, published artefacts, pasted
output. Name a source by its **key** only — every output of this command does.

This covers **identifying paraphrase** too: do not describe a confidential source
so specifically that a reader could identify it ("a forthcoming paper showing X
beats Y on Z", a private repository's distinctive architecture). The mechanical
guards catch literal strings only; this rule is the paraphrase layer. Refer
generically ("a working paper on X", "prior private work"). The ledger holds the
real reference, and citing it publicly is the user's decision, made when both the
source's `permission_status` and the ledger line's `cited_publicly` allow it.

Conversation with the user is exempt: discuss confidential sources freely there.

## Status (bare, read-only)

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" library --json
```

The envelope carries each store, or a note saying why it could not be read:

- `corpus`: report `present`, the `confidential` and `public` counts, each
  ledger's `repo` and `lines`, and every entry in `problems` by key and reason. A
  non-zero `remotes` is a warning to relay: documents and ledgers never leave
  this machine. When `corpus` is null, `corpus_note` says why — with no corpus,
  say so; only the user decides to create one, with `library init`.
- `memory`: summarise `pages` and `by_class` (page count per source class),
  `last_ingest`, any `contradictions`, per-source `headroom` lines, and every
  `drift` line verbatim — each says the index or the contradictions register is
  stale and names the ingest that rebuilds it. When `memory` is null,
  `memory_note` says why: outside a checkout there is no memory store.

The command exits 2 only when neither store can be read. The bare render never
rebuilds or mutates the memory store's coverage index.

## The sources corpus

Before any corpus work, run the bare status. A null `corpus`, whose `corpus_note`
says so, means there is no corpus: say so and stop, because only the user decides
to create one.

### Search the corpus

When the user asks what their sources say, or a design or research decision
arises where prior literature or private working material plausibly matters:

1. Search: `grep -ril "<term>" ~/.abcd.noindex/sources/confidential/
   ~/.abcd.noindex/sources/public/` — try keywords, author surnames and keys.
   Metadata lives in `sources.json` (CSL-JSON; the `custom` block carries
   `confidential`, `permission_status`, `keywords`, `aliases`, `ban_authors`).
   **The path is the classification**: any hit under `confidential/` falls under
   the hard rule. Each source is a folder (`<class>/<key>/`) holding
   `original.<ext>`, `text.md`, and any summaries or notes.
2. Read matched files freely for conversation; the folder's class governs what
   may leave it.
3. If nothing relevant surfaces, say so — do not pad.

Reading is plain search; every write goes through the verbs below.

### Read the document first

To add a source, read it before anything is registered. abcd fetches and
converts nothing, so this half is yours.

- A URL: fetch it (metadata and content), and save what you will store — the
  page or document, and its text as Markdown — to a scratch file; the verb
  stores files you hand it and never fetches.
- A local file: read it (a big file: the first pages suffice).

Extract the exact title (no site suffixes), the authors (each as `Family,
Given`), the year, the venue and the canonical URL. Map the type to CSL:
`article-journal` (papers), `webpage` (posts, docs), `book`, `report` (white
papers, internal docs), `motion_picture` (video). Two rules hold for author
names:

- **Resolve a name before caveating it.** An author given by initials only is
  looked up through the entry's own DOI or URL, which usually resolves the full
  name in one fetch. A "to be checked" caveat is for a fact that is genuinely
  unreachable, never for a lookup that was skipped.
- **Take names from the publisher's current record.** A name that has changed
  since publication is recorded as the publisher now gives it, never reverted to
  a former name found in an older copy, a citation elsewhere, or an index.

### Decide the class and the key

- **Class.** Web content is usually public. The signals for private: the user
  says so; it is their own unpublished or submitted work; internal or NDA
  material; AI-generated content (never citable); a private repository's
  documentation. These are signals for the question you put, never an answer in
  its place: the class is the person's (see [Register a source](#register-a-source)).
  For a private source, ask the same person for every identifying name variant
  (aliases — repository names, codenames, domains) and the `--permission`:
  `no-public-citation`, `internal-never-cite`, `ai-generated-never-cite`, or
  `ask-author`.
- **A private entry's title and aliases become banned phrases** (whole phrase,
  case-insensitive, whitespace-flexible), and each needs at least three letters
  or digits. For an internal artefact whose natural title reads like ordinary
  prose, register a distinctive title instead — "meeting-notes-2026-07
  (internal)" — so the ban cannot trip on legitimate text. Authors are banned
  only with `--ban-authors`, for a source whose authorship is itself identifying.
- **Key.** Public: `<authorfamily><year><distinctiveword>`, lowercase ASCII
  (`naur1985theory`). Private: an **opaque** key (`conf2026a`) — the key is the
  one handle every refusal and scan prints. A duplicate key is refused.
- **Keywords are the retrieval surface**: write 5–10 from having actually read
  the piece — topics, named tools and techniques, the claims it makes. Never
  generic filler ("AI", "software").

### Register a source

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" library add <document> --key <key> --private|--public \
  --meta <meta.json> [--type <csl-type>] [--year YYYY] [--venue V] [--url U] \
  [--permission <status>] [--ban-authors] [--text <extracted.md>] --json
```

- **The class is the person's to give, and it is never guessed.** `--private`
  files the item under `confidential/`: it is consulted, never cited, and its
  title and aliases become banned phrases. `--public` files it under `public/`.
  Before you run `library add`, ask whichever of the product thinker or the
  technical facilitator is adding the source whether it is private or public,
  through the host's question tool under the chip naming their role, unless the
  person already said, and pass the answer as `--private` or `--public`. The
  question follows the asking rules, and the guard's question check applies to
  it as to any question. Given neither flag, the binary asks at a terminal and,
  everywhere else (your shell included), refuses with exit 2 and writes
  nothing.
- For a private source, put the title, aliases and authors in a `--meta`
  JSON file (`{"title": …, "aliases": […], "author": [{"family": …, "given": …}],
  "keywords": […]}`) so they stay out of argv, and choose an **opaque** key
  (`conf2026a`): the verb refuses a key that contains an alias, the title or a
  longer title word, or an author's name.
- abcd fetches and converts nothing. A `.md` or `.txt` document is its own text;
  anything else needs `--text` with the extracted text; a URL alone registers a
  metadata stub. For a public source `--title`, `--author "Family, Given"`,
  `--alias` and `--keywords` work as flags too.
- There is no one-argument quick path: `library add` with explicit flags is the
  registrar, and the better one anyway, since you have read the source and hold
  metadata a bare fetch could not recover.

### Quality-check the extraction

Check `~/.abcd.noindex/sources/<class>/<key>/text.md`: a sane word count and real
prose. Known failure modes: `.mhtml` (unsupported, so a stub; extract by hand),
and saved single-page-app pages whose content sits HTML-escaped in a wrapper
(unescape the entities, `pandoc -t gfm`, rebuild `text.md` below its frontmatter,
and commit in the corpus repository — a hand repair to a stored file is the one
write the verb does not make). A link-only source has no body, and a
metadata-and-URL stub is then correct.

Then close out:

- A private source: run `sync-banlist` (below) in every guarded repository the
  session touches.
- A summary or review the user wants kept goes in the source's own folder
  (`summary.md`, notes beside it): derived artefacts inherit the source's class
  by location, never anywhere else.
- If a live decision motivated the addition, record the influence (below).
- Tell the user the key and class you registered.

### Record an influence

When a source meaningfully shapes a decision (supports it, contradicts it,
supplies a method, or shapes background understanding — not mere incidental
reading), record one line:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" library ledger --decision "<ADR id | intent id | date | text>" \
  --claim "<what was decided>" --source <key> --influence supports|contradicts|method|background \
  [--locator "§2"] [--used-in <path>]... --json
```

The line lands with `cited_publicly: false` and is committed in the corpus.
`--used-in` makes the acknowledgement machine-readable in both directions: an idea
is traced to its source even when the consuming document only paraphrases a
public source or must stay silent about a confidential one. Fill it whenever the
influence landed in an identifiable document, not just a conversation. A
correction is a new line: add `--corrects <N>`. Tell the user which key was
recorded against which decision and the line number, so they can decide about
citing.

**Never run `ledger --flip`.** It is the user's act of citing a line publicly. It
checks the source first — folder under `public/` and `permission_status` citable
— and refuses naming the failing gate. If the user asks how to cite, tell them
the command; do not run it for them. The shell guard refuses it from an agent
(`abcd-source-ledger-flip`).

`ledger --list` prints the ledger, numbered. The ledger is named by this
checkout's root commit unless `--repo` names it.

### Keep the guard current

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" library sync-banlist --json
```

Projects every confidential source's title and aliases (authors only under
`ban_authors`) into this repository's untracked private banlist, as a fenced block
it owns. Run by hand, it creates the store when there is something to ban. The
committed pre-commit guard runs `library sync-banlist --refresh` on every commit
(in a managed repository, only once the clone opts in with
`git config --local abcd.sourcesBinary /absolute/path/to/abcd`), which updates a
store that already exists and never creates one, so the first sync in a
repository is always the user's own, and running it by hand also matters after an
add or a declassification, before the next commit. A store in the legacy format is
not refreshed: the guard and the verb both name `abcd banlist migrate`. A refusal
naming keys means folders and entries disagree: repair those entries (the block
already written keeps banning meanwhile). A guard committed before the verb was
named `library` still runs `source sync-banlist`; `ahoy` reports it as a gap and
`ahoy install` rewrites it.

### Scan before sharing

Before any text that drew on confidential material is posted, shared or pasted
anywhere git does not gate:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" library cite-check <file> --json   # or - for stdin
```

Exit 1 means a finding: each names the source's key, the field (`title`,
`alias-N`, `author-N`), the line and the byte offset — never the matched text, so
the report is safe to relay. The offset counts from the start of the whole text,
not from the start of the line, and it can point at the one byte before the
phrase that bounds it. Reword generically and scan again. A clean scan
covers literal strings only, never an identifying paraphrase.

### Declassify a published source

When the user says a confidential source is now published and citable:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" library declassify <key> [--permission <status>] --json
```

The folder moves to `public/` in one corpus commit; then run `sync-banlist`.

### No corpus

Every corpus verb but `init` exits 3 with one line when there is no corpus at the
location. Say so and stop; never create a corpus the user did not ask for.

## The memory store

**The store addressed is the checkout's, from anywhere in the tree.** Every
memory verb resolves the repository root before it reads or writes, so a render
run from a package directory reports the checkout's pages rather than "store not
present", and an ingest lands in the checkout's store rather than laying a second
one under the directory you happen to be standing in. Outside a repository there
is no store to address: `ingest`, `ask` and `lint` exit **2**, read nothing and
write nothing, because pages filed outside every checkout are committed by
nothing and read by nothing. If a memory store also exists below the repository
root, the command names it on stderr and leaves it alone; relay that line,
because pages sitting there are read by no `ask`, no `lint`, and no reader of the
checkout's store.

### Distil a source into pages

This is `library ingest`, and it registers nothing in the sources corpus: to
register a document, see [Register a source](#register-a-source). Distil an external source (transcript / article / URL) into typed, cited
memory pages. A remote source must be **https**: a plaintext `http://` source
is refused by name, and a redirect that leaves https is refused per hop, because
the store copies a fetched source's text — and the licence header lifted out of
it — verbatim into durable provenance. PDF is a later-phase seam: the binary
rejects a PDF source with a clear error, because no text-extraction dependency
is wired.

A credential carried by the URL never reaches the store: basic-auth userinfo
(`https://user:pass@host/doc`) and credential-shaped query keys (`token`,
`api_key`, `apikey`, `access_token`, `password`, `secret`, case-insensitive —
names that carry a secret in essentially every usage, so an addressing
parameter such as `?key=` is never truncated) are stripped before the fetched address becomes the stored
origin and title, and masked in every fetch-failure message — including the
transport's own error, which is unwrapped to its cause so it cannot re-print
the address behind the mask. The rest of the address is reproduced unchanged.

**You** are the distiller: read the source, produce the
`DistilledPage` JSON array, and pass it to the binary via `--pages-json`
(a file, or `-` for stdin). The binary computes the provenance, licence, and
content hash, validates every page, and writes atomically.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" library ingest <path-or-https-url> --pages-json distilled.json --json
```

Add `--keep-original` to retain the source at
`.abcd/memory/sources/<sha256>.<ext>` (the lifeboat licence gate — not launch —
governs its export). Report `status`, `licence`, and the written `pages`, and
`scan_gap` whenever it is present: the repository armed gitleaks in
`.abcd/config/gitleaks.json` and the binary is not installed, so the pages were
redacted by the native scanner alone. A gitleaks run that fails refuses the
ingest. An
already-known source re-ingests from the registry with no `--pages-json`.

### Ask memory

Deterministic retrieval over the store, then a cited answer:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" library ask "<question>" --json
```

The default answer is the deterministic citation-renderer over the top-ranked
pages; every citation references `source_class`, `citation`, and `source_hash`.
`--top-n` sets the retrieval depth; `0` uses the pinned default. Optionally
file the answer back as a new page with `--file-back --page-json <file|->` (one
`DistilledPage` object you produce from the retrieved matches).
Report the `answer` and, if present, the `file_back` result.

### Lint

Full-store curator health-check — per-page quotation budgets, cumulative source
coverage, source-class and licence advisories, and secret or identity residue
in stored text (`MR001`, a blocker: the store's write-time redactor run over
every page, the sources registry and each text kept-original, and every page
name — each page file's and each registry back-link's — judged as the write
side judges a filename, for a hard-fail secret alone; the finding names the kind
and the line, never the span, and lint never rewrites the store):

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" library lint --json
```

It rebuilds the regenerable `.coverage_index.json` and writes a report under
`.abcd/.work.local/logs/memory/lint-<ts>/`. `report_dir`, `store_path` and each
finding's `file` are named relative to the repository. Summarise `summary.blockers` /
`summary.warnings` / `summary.infos` and each finding's `code` and `message`.
Blockers exit nonzero; warn-only exits 0.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
