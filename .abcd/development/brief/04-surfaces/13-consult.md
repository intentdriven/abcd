# `/abcd:consult` — Confidential Sources Corpus

Bring private reading into a design decision without risking it into a commit.
`/abcd:consult` searches a local-only corpus of working papers, private notes,
PDFs and books, discusses what it finds with you freely, and records which
source shaped which decision in an append-only ledger. What it never does is
let an identifying string from a confidential source reach a tracked file or
anything external.

The cost is one habit: when a source genuinely shapes a decision, the influence
is recorded, and you are told which key was recorded against which decision, so
the choice about ever citing it publicly stays yours.

It is **host-delegated**: no Go verb backs it and there is no bare-status
render. The workflow runs in the host agent from
[`commands/consult.md`](../../../../commands/consult.md), orchestrating the
corpus with `grep`, file reads, and `git`.

## What it does

The corpus lives at `~/.abcd/sources/`, each source held as a folder
`<class>/<key>/` under `confidential/` or `public/`. **The path IS the
classification**: any hit under `confidential/` falls under the hard rule
below. Metadata lives in `sources.json` as CSL-JSON. If the corpus is absent,
the command says so and stops; it never creates it.

## Flow

1. **Search** the corpus, trying keywords, author surnames, and CSL keys. If
   nothing relevant surfaces, say so and do not pad: an empty answer is an
   answer, and inventing relevance is how a consult stops being evidence.
2. **Read** matched files freely for conversation. The folder's class governs
   what may leave it.
3. **Record influence** whenever a source meaningfully shapes a decision
   (supports, contradicts, supplies a method, or informs background): one line
   appended to `~/.abcd/sources/ledger/<repo>.jsonl` and committed in the corpus
   repo. The ledger is append-only, so corrections are new lines rather than
   edits, and its `used_in` field traces the influence to the consuming document
   in both directions.
4. **Tell the user which key was recorded against which decision.** The
   `cited_publicly` flag is only ever flipped by hand, so a user cannot make
   that decision about a ledger line they were never told exists. This step is
   what turns the hard rule's second gate into something a human can operate.

## The hard rule

For any entry marked confidential, never write its title, authors, aliases, or
any identifying string into anything tracked by git or sent anywhere external:
commits, pull-request and issue text, docs, code comments, published artefacts.
This covers **identifying paraphrase** too. The mechanical guards catch literal
strings only, so this rule is the paraphrase layer. Refer generically ("a
working paper on X").

Public citation requires **both** the source's permission status **and** the
ledger line's `cited_publicly` flag, and only the user flips the second, by
hand. Conversation with the user is exempt: confidential sources may be
discussed freely there.

## Guard wiring

The rule is backed mechanically rather than trusted alone. The corpus ships
three programs. The first is the registrar on the write side: it takes a source
in, fixing its class at that moment and never afterwards. The second maintains a
generated block in the repo's untracked `.abcd/.work.local/private-names.txt`,
which the repo's pre-commit guard reads. The third scans a document before it is
committed or shared and exits non-zero when a confidential identifier is
present, naming only the CSL key so the report itself is safe to relay.

Both guard programs refuse wholesale on a corpus whose classes disagree: if any
entry's declared confidentiality does not match the folder it sits in, neither
the sync nor the scan does any work, and each says which entries to repair
first. That is the safe direction, and the failure to watch for is the quiet one:
a refused sync leaves the generated block exactly as it was, so a newly added
confidential source is not covered by it, and a refused scan clears nothing. Read
the exit code, not the absence of complaint.

**That file has two writers, and abcd is the other one.**
`.abcd/.work.local/private-names.txt` is abcd's own private banlist layer
(`banlist.PrivateRelPath`, itd-74 / spc-20), maintained by the banlist verb's
private-layer add and remove. The two writers coexist on a format
contract: the file's first line must be exactly `# abcd-banlist: keyed`, the
corpus script refuses a target whose first line is not that declaration, and it
confines itself to a fenced generated block, so hand-added and verb-added lines
outside that block survive. See [`20-banlist.md`](20-banlist.md) for the store
itself.

**The mechanical layer is narrower than the hard rule, deliberately.** Patterns
are derived from every confidential entry's title and aliases always, and from
full author names only where that entry opts in with `custom.ban_authors`.
Authors are inside the hard rule unconditionally, so on an entry that has not
opted in the author names are held by the rule and by the reader applying it,
and by no mechanical check at all. Set `ban_authors` on any entry whose
authorship is itself identifying.

**The opt-in buys banlist coverage only, and not the scan.** The sync honours
`ban_authors` and puts the author names into the generated block, so the
pre-commit guard catches them; the document scan matches on titles and aliases
alone and reads no author field at all. A document naming a banned author and
nothing else therefore passes the scan and reports clean, and is then caught at
the commit. Treat a clean scan as covering what a source is called, never who
wrote it.

## Acceptance

- **Given** a present corpus, **when** a search returns a hit under
  `confidential/`, **then** the source is read and discussed freely with the
  user and no identifying string from it reaches a tracked file or any external
  destination.
- **Given** a source that meaningfully shaped a decision, **when** the influence
  is recorded, **then** exactly one line is appended to the repo's ledger with
  `cited_publicly: false`, it is committed in the corpus repo, and the user is
  told in conversation which key was recorded against which decision.
- **Given** a search that surfaces nothing relevant, **when** the command
  answers, **then** it says so and pads with nothing.
- **Given** a corpus that does not exist, **when** `/abcd:consult` is invoked,
  **then** the command says so and stops: it never creates the corpus.
- **Given** a repo with the guard installed and a corpus whose classes agree,
  **when** a confidential source is added, **then** the ban-list sync is run so
  the generated block in `.abcd/.work.local/private-names.txt` covers it before
  the next commit.
- **Given** a corpus holding a class mismatch, **when** the ban-list sync is run,
  **then** it writes nothing, names the entries to repair, and exits non-zero, so
  the run is not mistaken for coverage.

## Composition

`/abcd:consult` is the read-and-record side of the sources system;
[`/abcd:ingest`](14-ingest.md) is the write side that adds a source. The two
share the corpus and its ledger.

## References

- Plugin command: [`commands/consult.md`](../../../../commands/consult.md)
- Corpus store and its schema: `~/.abcd/sources/README.md`
- Write side of the same corpus: [`14-ingest.md`](14-ingest.md)
- The banlist store the guard wiring shares: [`20-banlist.md`](20-banlist.md)
- The trust boundary the hard rule restates:
  [adr-41](../../decisions/adrs/0041-corpus-trust-boundary.md), brief invariant 9
  ([`../02-constraints/03-invariants.md`](../02-constraints/03-invariants.md))
- Consuming intent: [itd-76](../../intents/planned/itd-76-source-provenance-ledger.md)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

There is no shipped surface: the command tree registers no `abcd consult` verb, so there are no flags and no sub-verbs to list.

<!-- surface-appendix:end -->
