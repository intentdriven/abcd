# `/abcd:source` — The Sources Corpus and Its Provenance Ledger

Read anything that bears on a decision, including material you are not free to
name in public, and never let a helpful footnote name it. `/abcd:source` keeps a
personal corpus of documents in a local-only git repository, records which source
shaped which decision in an append-only ledger per repository, bans every
confidential source's names at commit time, and turns a ledger line into a public
citation only when a person decides to and the source permits it.

The cost is two habits: declaring a source's class once, when it is added, and
recording an influence when a source genuinely shapes a decision. The payoff is
that nothing confidential can reach a commit by accident, and the day a paper is
published the whole influence trail behind it is already written.

It is the binary half of the sources system. The host-delegated pages
[`/abcd:consult`](13-consult.md) and [`/abcd:ingest`](14-ingest.md) carry the
judgement (what to search for, which class a document is, which keywords matter)
and call these verbs for every write.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `add` | — | shipped |
| `cite-check` | gate | shipped |
| `declassify` | — | shipped |
| `init` | — | shipped |
| `ledger` | — | shipped |
| `sync-banlist` | — | shipped |

Bare `abcd source` is read-only: where the corpus is, how many sources sit in
each class, the ledgers and their line counts, and any entry whose folder and
metadata disagree. The corpus is the default one under the user-level home,
`~/.abcd/sources`, unless the invocation names another directory.

## The store

The corpus is a directory the person owns, outside every repository, and itself a
git repository with no remote. Its history is the tamper-evidence layer: every
write a verb makes is committed there.

| path | holds |
|---|---|
| `sources.json` | a CSL-JSON array; each entry's `custom` block carries `confidential`, `permission_status`, `keywords`, `aliases`, `ban_authors` and `file` |
| `confidential/<key>/` | the original document, its extracted text as `text.md`, and anything derived from them |
| `public/<key>/` | the same shape for a freely citable source |
| `ledger/<repo>.jsonl` | one repository's influence records, one JSON object per line |

**The folder is the classification.** A source's class is declared once, when it
is added: the add requires the class, confidential or public, and never defaults,
because a forgotten flag must not file a confidential document as public. Derived
artefacts (summaries, notes) live in the source's folder and inherit its class.
The entry's `confidential` flag mirrors the folder, and a corpus where the two
disagree (a folder moved by hand, an entry edited by hand) is refused wholesale by
the banlist sync and the scan, naming each key to repair. That is the safe
direction: the block already written keeps banning.

A repository's ledger is named by the first twelve hex digits of its root commit,
which every clone and worktree of one repository shares whatever its directory is
called, unless the invocation names it.

Creating the corpus is its own explicit step, and it refuses a location inside
another repository's working tree, where one `git add -A` would carry documents into it. Nothing
fetches and nothing converts: a Markdown or plain-text document is its own text,
any other needs its extracted text passed alongside, and a URL alone registers a
metadata stub.

## Keys are the only handle output carries

Every refusal, every guard message and every scan names a source by its key and
nothing else. So a confidential source's key must not name it: the add refuses a
key containing one of its aliases, its title or a longer title word, or an
author's name, and asks for an opaque key such as `conf2026a`. The identifying
strings themselves can travel in a metadata file or on stdin, so they stay out of
the process list and the shell history.

## Recording influence, and citing

Recording an influence appends `{ts, repo, decision_ref, claim, source_key,
locator, influence, cited_publicly}` with `cited_publicly` false, and commits it. There is no edit
path: a correction is a new line naming the line it corrects.

A public citation needs both gates of
[adr-41](../../decisions/adrs/0041-corpus-trust-boundary.md). The source grants
the right: its folder is under `public/` and its `permission_status` is
`citable`, the one value in the closed vocabulary that grants it. The person
exercises the right: the flip of line N checks the first gate, refuses naming it
when it fails, and on success appends a new line, a copy of line N with
`cited_publicly` true and `flips` naming N. An agent never runs the flip. The
binary cannot tell a person from an agent, so that rule lives in the command
pages.

Declassification is how a confidential source becomes citable when it is
published: its folder moves `confidential/ → public/` by `git mv`, its entry's
flag and permission follow, and both land in one corpus commit. The next banlist
sync drops its strings, and its ledger lines become flippable.

## The guard

The banlist sync projects every confidential source into a fenced block of this
repository's untracked private banlist, `.abcd/.work.local/private-names.txt` (the
private layer of [`/abcd:banlist`](20-banlist.md)): the title and each alias
always, the authors only where the entry sets `ban_authors`. Lines outside the
fence, hand-written or verb-added, survive every sync. The by-hand sync creates
the store when there is something to ban; the refresh mode updates a store that
already exists and declares the keyed format, and never creates one, because a
store created by a commit's side effect would write every confidential title into
a repository's local tier without the person asking. The committed pre-commit
guard runs the sync in its refresh mode before it reads the store, so the block
is never more than one commit stale, and then refuses a staged commit carrying a
banned string by key. The refresh's own line is relayed on the commit, never
discarded: its count is how a refresh that wrote fewer patterns than the last
one shows itself.

The two copies of the guard find the binary differently. The copy this
repository runs builds `./cmd/abcd` from the checkout, as its commit-msg hook
does, and never runs an installed abcd, which in a source checkout is the last
release and may lack the verb entirely; a tree that does not build is one
warning line. The copy `abcd ahoy` scaffolds refreshes only on opt-in, per
clone: the repo-local git setting `abcd.sourcesBinary` names an absolute path
to the binary, and nothing else is consulted, neither `PATH` nor an
environment variable. How a scaffolded hook finds abcd, and whether it does so
by default, is a ruling owed to the product thinker
(iss-2609250834251447); opt-in decides none of it, and the ruling can widen
it. Without the setting, AC3 holds in this repository and in a managed
repository that opted in, and the scaffolded guard names the setting on one
line.

A legacy private store is not refreshed. The guard names the banlist verb's
migration ([`20-banlist.md`](20-banlist.md)) on the first commit that meets it
and not again, since the store's format line already says "legacy" on every
commit.

Each phrase is projected into the pattern language the guard enforces: literal,
case-insensitive, and whitespace-flexible across ASCII and Unicode space
separators, with a non-ASCII letter written as its case-fold alternatives because
the guard's C-locale engine folds ASCII only. The boundary is a test on the
neighbouring bytes, never `\b`: POSIX extended expressions do not define it, and
RE2's ASCII-only reading finds no word start before a non-ASCII initial. The neighbour test
errs towards matching: a phrase beside a non-ASCII letter still counts as found.
A phrase with fewer than three letters or digits is refused, because it would
ban ordinary words.

The scan reads a file or stdin through the same projection and the same engine
as the guard, so a text it calls clean is a text the guard would pass. It
reports each finding by key, field (`title`, `alias-N`, `author-N`), line and
byte offset, and never by the text matched, so its report is safe to relay. The
offset is the guard engine's own: it counts from the start of the whole text,
not from the start of the line, to the start of the matched span, which can be
the one byte before the phrase that bounds it. It exits 1 when anything is found.

## Absence is loud

With no corpus at the configured location every verb but the creating one says
so on one line and exits 3, a code distinct from a refusal, so a script can tell
"nothing to check against" from "checked and clean". The guard says so on one line
and lets the commit proceed; the sync's refresh mode, which the guard runs, does
the same and exits 0, as it does for a repository with no private store. A
corpus the guard cannot refresh (no binary built or opted in, a binary without
the verb, or a refresh that fails) is one line on the commit, and the store is
checked as it stands.

## What it cannot enforce

The mechanical layer blocks literal identifying strings on one line of a staged
file. It cannot see a paraphrase that identifies a source without naming it, a
name split across a line break, or a spelling the entry does not carry (an
unrecorded alias, a decomposed Unicode form). Those are the consultation rule's
job and the human review before anything is published. Durability is bounded
too: a corpus with no remote survives the loss of its disk only through the
person's own backups and offline `git bundle` snapshots, which abcd documents and
cannot perform.

## Acceptance

- **Given** a document and its metadata, **when** it is added as confidential,
  **then** the bibliography gains its CSL-JSON entry with the `custom` block, and
  the document and its extracted text land under `confidential/<key>/`.
- **Given** a consulted source that shaped a decision, **when** the influence is
  recorded, **then** one line is appended with `cited_publicly` false, and a
  correction is another new line.
- **Given** confidential entries in the corpus, **when** a commit runs in a
  managed repository, **then** the guard refreshes the generated block (titles
  and aliases always, authors only under `ban_authors`) and refuses a commit
  carrying a banned string.
- **Given** text about to leave the machine, **when** the scan reads it,
  **then** offending sources are reported by key only.
- **Given** a ledger line whose source lacks citation permission, **when** a flip
  is attempted, **then** it is refused naming the failing gate; with permission
  present the flip succeeds as a new line.
- **Given** a machine with no corpus, **when** abcd runs in a managed repository,
  **then** every corpus-dependent step says so on one line, and none fails.
- **Given** a confidential source that is published, **when** it is declassified,
  **then** the next refresh drops its strings and its ledger lines become
  eligible for the flip.

## References

- Plugin command: [`commands/source.md`](../../../../commands/source.md)
- The host-delegated halves: [`13-consult.md`](13-consult.md),
  [`14-ingest.md`](14-ingest.md)
- The private banlist layer the guard shares: [`20-banlist.md`](20-banlist.md)
- The trust boundary: [adr-41](../../decisions/adrs/0041-corpus-trust-boundary.md),
  brief invariant 9
  ([`../02-constraints/03-invariants.md`](../02-constraints/03-invariants.md))
- Consuming intent: [itd-76](../../intents/shipped/itd-76-source-provenance-ledger.md)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd source`

Sub-verbs: `abcd source add`, `abcd source cite-check`, `abcd source declassify`, `abcd source init`, `abcd source ledger`, `abcd source sync-banlist`.

| Flag | Type |
|---|---|
| `--corpus` | string |

### `abcd source add`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--alias` | stringArray |
| `--author` | stringArray |
| `--ban-authors` | bool |
| `--confidential` | bool |
| `--key` | string |
| `--keywords` | stringArray |
| `--meta` | string |
| `--permission` | string |
| `--public` | bool |
| `--text` | string |
| `--title` | string |
| `--type` | string |
| `--url` | string |
| `--venue` | string |
| `--year` | int |

### `abcd source cite-check`

Sub-verbs: none.

Flags: none.

### `abcd source declassify`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--permission` | string |

### `abcd source init`

Sub-verbs: none.

Flags: none.

### `abcd source ledger`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--claim` | string |
| `--corrects` | int |
| `--decision` | string |
| `--flip` | int |
| `--influence` | string |
| `--list` | bool |
| `--locator` | string |
| `--repo` | string |
| `--source` | string |
| `--used-in` | stringArray |

### `abcd source sync-banlist`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--refresh` | bool |

<!-- surface-appendix:end -->
