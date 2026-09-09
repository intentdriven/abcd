# `/abcd:consult` — Confidential Sources Corpus

`/abcd:consult` searches a local-only corpus of source documents — working
papers, private-repo notes, PDFs, books — that the agent may **consult** but must
never **cite** publicly, and records source→decision provenance in an append-only
ledger. It is **host-delegated**: no Go verb backs it, there is no `abcd consult`
binary sub-verb and no bare-status render — the workflow runs entirely in the host
agent, orchestrating the corpus with `grep`, file reads, and `git` in the corpus
repo.

## What it does

The corpus lives at `~/.abcd/sources/` (the user-level home's sources store), with
each source held as a folder `<class>/<key>/` under `confidential/` or `public/`.
**The path IS the classification**: any hit under `confidential/` falls under the
hard rule. Metadata lives in `sources.json` (CSL-JSON; the `custom` block carries
`confidential`, `permission_status`, `keywords`, `aliases`, `ban_authors`, and
`file`). `ban_authors` is the one that changes what the mechanical guard covers,
and § Guard wiring below says how. If the corpus is absent, the command says so
and stops — it never creates it.

## Flow

1. **Search** the corpus with `grep -ril "<term>"` across `confidential/` and
   `public/`, trying keywords, author surnames, and CSL keys. If nothing
   relevant surfaces, say so and do not pad: an empty corpus answer is an
   answer, and inventing relevance is how a consult stops being evidence.
2. **Read** matched files freely for conversation; the folder's class governs what
   may leave it.
3. **Record influence**: whenever a source meaningfully shapes a decision
   (supports, contradicts, supplies a method, or informs background), append ONE
   line to `~/.abcd/sources/ledger/<repo>.jsonl` and commit it in the corpus repo.
   The ledger is append-only, so corrections are new lines, never edits, and
   `used_in` traces the influence to the consuming document in both directions.
4. **Tell the user which key was recorded against which decision**, in
   conversation. `cited_publicly` is only ever flipped by hand, so the user
   cannot make that decision about a ledger line they were never told exists:
   this step is what turns the hard rule's second gate into something a human
   can actually operate.

## The hard rule

For any entry with `custom.confidential: true`, never write its title, authors,
aliases, or any identifying string into anything tracked by git or sent anywhere
external — commits, PR/issue text, docs, code comments, published artifacts. This
covers **identifying paraphrase** too: the mechanical guards catch literal strings
only, so this rule is the paraphrase layer. Refer generically ("a working paper on
X"). Public citation requires BOTH the source's `permission_status` AND the ledger
line's `cited_publicly` flag — and `cited_publicly` is only ever flipped by the
user, by hand. Conversation with the user is exempt: confidential sources may be
discussed freely there.

## Guard wiring

The command's rule is backed mechanically, not trusted alone:

- `~/.abcd/sources/bin/sync-banlist <repo-root>` maintains a generated block in
  the repo's untracked `.abcd/.work.local/private-names.txt`, which the repo's
  pre-commit guard reads: run it on first use in a repo and after any
  confidential source is added.
- `~/.abcd/sources/bin/cite-guard <file>` runs before any document that drew on
  confidential material is committed or shared (exit 1 = confidential identifier
  present; its report names only the CSL key, so the report itself is safe to
  relay).

**That file has two writers, and abcd is the other one.**
`.abcd/.work.local/private-names.txt` is abcd's own private banlist layer
(`banlist.PrivateRelPath`, itd-74 / spc-20), maintained by `abcd banlist add
--private` and `remove --private` and rendered by `abcd banlist`. The two
writers coexist on a format contract: the file's first line must be exactly
`# abcd-banlist: keyed` (the guard's `format_decl`), `sync-banlist` refuses a
target whose first line is not that declaration, and the corpus script confines
itself to a fenced generated block so hand-added and verb-added lines outside it
survive. See [`20-banlist.md`](20-banlist.md) for the store itself.

The mechanical layer is narrower than the hard rule, deliberately.
`sync-banlist` derives patterns from every confidential entry's title and
aliases **always**, and full author names **only** where that entry sets
`custom.ban_authors: true`. `cite-guard` scans for title and aliases too, and
for nothing else. Authors are inside the hard rule unconditionally, so on an
entry that has not opted in the author names are held by the rule and by the
reader applying it, and by no mechanical check at all. Set `ban_authors` on any
entry whose authorship is itself identifying.

## Acceptance

- **Given** a present corpus, **when** a search returns a hit under
  `confidential/`, **then** the source is read and discussed freely with the
  user and no identifying string from it reaches a tracked file or any external
  destination.
- **Given** a source that meaningfully shaped a decision, **when** the influence
  is recorded, **then** exactly one line is appended to
  `~/.abcd/sources/ledger/<repo>.jsonl` with `cited_publicly: false`, it is
  committed in the corpus repo, and the user is told in conversation which key
  was recorded against which decision.
- **Given** a search that surfaces nothing relevant, **when** the command
  answers, **then** it says so and pads with nothing.
- **Given** a corpus that does not exist, **when** `/abcd:consult` is invoked,
  **then** the command says so and stops: it never creates the corpus.
- **Given** a repo with the guard installed, **when** a confidential source is
  added, **then** `sync-banlist <repo-root>` is run so the generated block in
  `.abcd/.work.local/private-names.txt` covers it before the next commit.

## Composition

`/abcd:consult` is the read-and-record side of the sources system; `/abcd:ingest`
is the write side that adds a source to the corpus. The two share the corpus at
`~/.abcd/sources/` and its ledger.

## References

- Plugin command: [`commands/consult.md`](../../../../commands/consult.md)
- Corpus store and its schema: `~/.abcd/sources/README.md`
- Write side of the same corpus: [`14-ingest.md`](14-ingest.md)
- The banlist store the guard wiring shares: [`20-banlist.md`](20-banlist.md)
- The trust boundary the hard rule restates:
  [adr-41](../../decisions/adrs/0041-corpus-trust-boundary.md), brief invariant 9
  ([`../02-constraints/03-invariants.md`](../02-constraints/03-invariants.md))
- Consuming intent: [itd-76](../../intents/planned/itd-76-source-provenance-ledger.md)
