---
name: source
description: Maintain the personal sources corpus (the user-level home's sources store, default ~/.abcd/sources) and its provenance ledger by invoking the abcd binary — register a source, record an influence, project confidential names into this repo's private banlist, and scan text before it leaves the machine. Bare invocation is a read-only render.
argument-hint: "[init | add [document] --key K --confidential|--public [...] | declassify <key> | ledger [--decision D --claim C --source K --influence I | --flip N | --list] | sync-banlist [--refresh] | cite-check <file|->]"
---

# `/abcd:source` — the sources corpus and its ledger

The corpus holds documents the user may consult but is not always free to cite:
working papers, private notes, purchased reports. The folder a source sits in —
`confidential/<key>/` or `public/<key>/` — is its classification. The ledger
records which source shaped which decision in this repository. `/abcd:consult`
and `/abcd:ingest` are the judgement halves and call these verbs for every write.

## The hard rule

Never write a confidential source's title, aliases, authors or any identifying
description into anything tracked by git or sent anywhere external: commits,
pull-request and issue text, docs, code comments, pasted output. Name a source by
its **key** only — every output of this verb does. Conversation with the user is
exempt.

## Render the corpus (bare, read-only)

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" source --json
```

Report `present`, the `confidential` and `public` counts, each ledger's `repo` and
`lines`, and every entry in `problems` by key and reason. A non-zero `remotes` is
a warning to relay: documents and ledgers never leave this machine. Exit 3 means
there is no corpus: say so, and stop. Only the user decides to create one, with
`source init`.

## Register a source

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" source add <document> --key <key> --confidential|--public \
  --meta <meta.json> [--type <csl-type>] [--year YYYY] [--venue V] [--url U] \
  [--permission <status>] [--ban-authors] [--text <extracted.md>] --json
```

- The class is required and never defaulted. Ask the user when in doubt.
- For a confidential source, put the title, aliases and authors in a `--meta`
  JSON file (`{"title": …, "aliases": […], "author": [{"family": …, "given": …}],
  "keywords": […]}`) so they stay out of argv, and choose an **opaque** key
  (`conf2026a`): the verb refuses a key that contains an alias, the title or a
  longer title word, or an author's name.
- abcd fetches and converts nothing. A `.md` or `.txt` document is its own text;
  anything else needs `--text` with the extracted text; a URL alone registers a
  metadata stub.

## Record an influence

When a source meaningfully shapes a decision (supports, contradicts, supplies a
method, or shapes background):

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" source ledger --decision "<ADR id | intent id | date | text>" \
  --claim "<what was decided>" --source <key> --influence supports|contradicts|method|background \
  [--locator "§2"] [--used-in <path>]... --json
```

The line lands with `cited_publicly: false` and is committed in the corpus. A
correction is a new line: add `--corrects <N>`. Tell the user which key was
recorded against which decision and the line number.

**Never run `ledger --flip`.** It is the user's act of citing a line publicly. It
checks the source first — folder under `public/` and `permission_status` citable
— and refuses naming the failing gate. If the user asks how to cite, tell them
the command; do not run it for them.

`ledger --list` prints the ledger, numbered. The ledger is named by this
checkout's root commit unless `--repo` names it.

## Keep the guard current

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" source sync-banlist --json
```

Projects every confidential source's title and aliases (authors only under
`ban_authors`) into this repository's untracked private banlist, as a fenced block
it owns. Run by hand, it creates the store when there is something to ban. The
committed pre-commit guard runs `sync-banlist --refresh` on every commit, which
updates a store that already exists and never creates one, so the first sync in a
repository is always the user's own, and running it by hand also matters after an
add or a declassification, before the next commit. A refusal naming keys means folders and entries disagree: repair
those entries (the block already written keeps banning meanwhile).

## Scan before sharing

Before any text that drew on confidential material is posted, shared or pasted
anywhere git does not gate:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" source cite-check <file> --json   # or - for stdin
```

Exit 1 means a finding: each names the source's key, the field (`title`,
`alias-N`, `author-N`), the line and the byte offset — never the matched text, so
the report is safe to relay. Reword generically and scan again. A clean scan
covers literal strings only, never an identifying paraphrase.

## Declassify a published source

When the user says a confidential source is now published and citable:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" source declassify <key> [--permission <status>] --json
```

The folder moves to `public/` in one corpus commit; then run `sync-banlist`.

## No corpus

Every verb but `init` exits 3 with one line when there is no corpus at the
location. Say so and stop; never create a corpus the user did not ask for.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
