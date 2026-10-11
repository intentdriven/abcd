---
name: library
description: "Render the sources corpus and the checkout's memory store, read-only: Writes nothing; refuses only when neither store can be read."
argument-hint: "[<empty>] | init | add [document] --key K --confidential|--public [...] | declassify <key> | ledger [--decision D --claim C --source K --influence I | --flip N | --list] | sync-banlist [--refresh] | cite-check <file|-> | ingest <path-or-https-url> [--keep-original] | ask <question> | lint"
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
  `cite-check`. `/abcd:consult` and `/abcd:ingest` are the judgement halves and
  call these verbs for every write.
- **The memory store**, at the checkout's `.abcd/memory/`, holds typed, cited
  pages distilled from a source. Its verbs are `ingest`, `ask` and `lint`.
  Nothing moves a corpus document into the memory store: the memory store's
  `ingest` distils a source into pages, and the corpus's `add` registers a
  document.

## The hard rule

Never write a confidential source's title, aliases, authors or any identifying
description into anything tracked by git or sent anywhere external: commits,
pull-request and issue text, docs, code comments, pasted output. Name a source by
its **key** only — every output of this command does. Conversation with the user
is exempt.

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

### Register a source

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" library add <document> --key <key> --confidential|--public \
  --meta <meta.json> [--type <csl-type>] [--year YYYY] [--venue V] [--url U] \
  [--permission <status>] [--ban-authors] [--text <extracted.md>] --json
```

- The class is required and never defaulted. When in doubt, ask whichever of the
  product thinker or the technical facilitator is adding the source, under the
  chip naming their role.
- For a confidential source, put the title, aliases and authors in a `--meta`
  JSON file (`{"title": …, "aliases": […], "author": [{"family": …, "given": …}],
  "keywords": […]}`) so they stay out of argv, and choose an **opaque** key
  (`conf2026a`): the verb refuses a key that contains an alias, the title or a
  longer title word, or an author's name.
- abcd fetches and converts nothing. A `.md` or `.txt` document is its own text;
  anything else needs `--text` with the extracted text; a URL alone registers a
  metadata stub.

### Record an influence

When a source meaningfully shapes a decision (supports, contradicts, supplies a
method, or shapes background):

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" library ledger --decision "<ADR id | intent id | date | text>" \
  --claim "<what was decided>" --source <key> --influence supports|contradicts|method|background \
  [--locator "§2"] [--used-in <path>]... --json
```

The line lands with `cited_publicly: false` and is committed in the corpus. A
correction is a new line: add `--corrects <N>`. Tell the user which key was
recorded against which decision and the line number.

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

### Ingest a source

Distil an external source (transcript / article / URL) into typed, cited
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
