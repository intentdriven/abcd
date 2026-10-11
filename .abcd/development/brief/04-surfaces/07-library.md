# `/abcd:library` — The Sources Corpus and the Memory Store

Two stores under one command, kept apart on purpose. The **sources corpus** keeps
the documents a person reads for a decision, including material they are not free
to name in public, and records which source shaped which decision. The **memory
store** keeps what a project learns from what it reads, as distilled, cited pages
inside the repository. One command reaches both, and nothing connects them:
nothing distils a corpus document into the memory store, and nothing moves a
memory page into the corpus (spc-2610100613109045, decision 10).

The command merges the two that came before it, `source` and `memory`, under
the product thinker's ruling of 2026-10-09 that the sources commands become one
person's command named `library`
([itd-2610090831227812](../../intents/planned/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md)).
Under [adr-40](../../decisions/adrs/0040-review-audit-lint-are-three-verbs.md) neither old name survives as
an alias. Every sub-verb kept its name (decision 7), so the memory store's
ingest, which distils a source into pages, sits beside the corpus's add, which
registers a document. The host-delegated pages
[`/abcd:consult`](13-consult.md) and [`/abcd:ingest`](14-ingest.md) carry the
corpus's judgement (what to search for, which class a document is, which
keywords matter) and call these verbs for every write; the page named ingest
means the corpus's add, not the memory store's ingest (iss-2610110233253365).

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
| `ask` | — | shipped |
| `cite-check` | gate | shipped |
| `declassify` | — | shipped |
| `ingest` | — | shipped |
| `init` | — | shipped |
| `ledger` | — | shipped |
| `lint` | lint | shipped |
| `sync-banlist` | — | shipped |

Six of the verbs act on the sources corpus: creating it, adding a source,
declassifying one, the ledger, the banlist sync and the scan. Three act on the
memory store: ingesting a source, asking a question and the store lint.

## Bare `abcd library`

Bare `abcd library` is read-only and renders both stores: the corpus (where it
is, how many sources sit in each class, the ledgers and their line counts, and
any entry whose folder and metadata disagree) and the checkout's memory store
(below). A store that cannot be read is one line saying why and naming what
would create it: the corpus's creating verb for a missing corpus, a checkout
for the memory store. The command refuses, exit 2, only when neither can be read
(decision 8). So inside a checkout with no corpus it renders the memory store
and a line about the corpus, and outside every checkout with a corpus it
renders the corpus and a line about the memory store. The JSON envelope
carries `corpus` and `memory`, each null when that store could not be read, with
`corpus_note` or `memory_note` saying why. The corpus is the default one under
the user-level home, `~/.abcd.noindex/sources`, unless the invocation names
another directory.

## The sources corpus

Read anything that bears on a decision, including material you are not free to
name in public, and never let a helpful footnote name it. The corpus is a
personal collection of documents in a local-only git repository; an append-only
ledger per repository records which source shaped which decision; every
confidential source's names are banned at commit time; and a ledger line becomes
a public citation only when a person decides to and the source permits it.

The cost is two habits: declaring a source's class once, when it is added, and
recording an influence when a source genuinely shapes a decision. The payoff is
that nothing confidential can reach a commit by accident, and the day a paper is
published the whole influence trail behind it is already written.

### The store

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
is added, and it is the person's to give: private (stored in the confidential
folder) or public. The add never defaults it, because a forgotten flag must not
file a confidential document as public. Given no class, the add asks the one
question at a terminal, naming both answers and what each means, and off a
terminal it refuses and writes nothing; a host asks the person through its own
question tool and passes the answer
(spc-2610100613109045, A13 and decision 9). Derived
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

### Keys are the only handle output carries

Every refusal, every guard message and every scan names a source by its key and
nothing else. So a confidential source's key must not name it: the add refuses a
key containing one of its aliases, its title or a longer title word, or an
author's name, and asks for an opaque key such as `conf2026a`. The identifying
strings themselves can travel in a metadata file or on stdin, so they stay out of
the process list and the shell history.

### Recording influence, and citing

Recording an influence appends `{ts, repo, decision_ref, claim, source_key,
locator, influence, cited_publicly}` with `cited_publicly` false, plus `used_in`,
the repository paths the influence landed in, when the caller names them, and commits it. There is no edit
path: a correction is a new line naming, in `corrects`, the line it corrects.

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

### The guard

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

### Absence is loud

With no corpus at the configured location every corpus verb but the creating one says
so on one line and exits 3, a code distinct from a refusal, so a script can tell
"nothing to check against" from "checked and clean". The guard says so on one line
and lets the commit proceed; the sync's refresh mode, which the guard runs, does
the same and exits 0, as it does for a repository with no private store. A
corpus the guard cannot refresh (no binary built or opted in, a binary without
the verb, or a refresh that fails) is one line on the commit, and the store is
checked as it stands.

### What it cannot enforce

The mechanical layer blocks literal identifying strings on one line of a staged
file. It cannot see a paraphrase that identifies a source without naming it, a
name split across a line break, or a spelling the entry does not carry (an
unrecorded alias, a decomposed Unicode form). Those are the consultation rule's
job and the human review before anything is published. Durability is bounded
too: a corpus with no remote survives the loss of its disk only through the
person's own backups and offline `git bundle` snapshots, which abcd documents and
cannot perform.

## The memory store

Keep what a project learns from the things it reads, in a form that is still
usable a year later: distilled pages rather than stored documents, each carrying
the citation, licence and content hash it came from, so an answer drawn out of
the store can always be traced back to a source and re-checked. The store lives
in the repo at `.abcd/memory/`, so it travels with the project and is reviewable
in a diff.

The cost is the discipline: sources are distilled and discarded by default
rather than hoarded, the quotation budget is a curation rule rather than a
storage limit, and licence and provenance are recorded at ingest instead of
reconstructed later.

The **substrate spec** (page-class enum, source-class taxonomy, curator
behaviour, lifecycle class) is
[`05-internals/07-memory.md`](../05-internals/07-memory.md). This section is the
surface contract: what the user types and what happens.

> **Provenance, and a warning about ids.** The memory store traces to itd-36,
> which sits in `intents/shipped/`. The write core and the lint family were
> specified in a predecessor store whose `spc-38` and `spc-39` collide with live
> ids in this repo's own store. So in this section the store an id belongs to is
> read from the sentence around it, never from the number.

**The store every memory verb addresses is the checkout's, resolved from the working directory and never taken to be it.** The front door asks [`gitutil.CheckoutRoot`](../../../../internal/gitutil/repo.go) before it builds a request — the same resolution `capture`, `decide` and `spec` address their stores through: git's toplevel where git will name one, and a refusal in the two remaining states rather than a guess (a repo-shaped tree git will not answer for, and no repository above at all). So a render from a package directory reports the checkout's pages instead of `store not present`, an ingest lands in the checkout's substrate instead of laying a second one beneath the caller, and outside a checkout every memory verb but the bare render exits **2** having read nothing and written nothing. It deliberately does not fall through to a marker walk, which would accept any directory carrying the name (iss-2609090947359464). A memory store found **below** the checkout root, on the chain between the caller and it, is named on stderr and left untouched — the deposit an unresolved front door leaves behind, reported to the person standing over it rather than stepped over in silence.

**Its half of bare `abcd library`** renders the store's state and nothing else: how many
pages there are by class, when the last ingest happened, the recent
contradictions, per-source quotation-budget headroom, and drift. It never
mutates and never rebuilds an index. Drift is a line saying that the catalogue
or the contradictions register no longer hash-matches what the store's pages
would render, naming the ingest as the verb that rebuilds it, so a reader
knows the numbers are stale rather than wrong; the text board prints
each line in the words the JSON's `drift` list carries, and a current store
prints none. Headroom is read-only in the same
spirit: a fresh index shows per-source warn and block headroom, a drifted one
says to run the lint, and an absent or unreadable one says the headroom is
unavailable rather than guessing at it.

**Ingesting a source** — a local path or an https URL — registers it as typed
pages with citation frontmatter, and appends to the ingest log. The host agent
is the distiller: it reads the source, produces the distilled pages, and passes
them to the binary as JSON, which is load-bearing and required for a
source the store has not seen (an already-known source re-ingests from the
registry without it). The binary computes provenance, licence and content hash,
validates every page, and writes atomically.

Three refusals matter, because the store copies fetched text and its licence
verbatim into a durable artefact. A plaintext `http://` source is refused by
name, and a redirect that leaves https is refused per hop. A credential the URL
carries (basic-auth userinfo, or a query key such as `token` or `api_key`) is
stripped before the fetched address becomes the stored origin, and masked in
every fetch-failure message. And PDF is a later seam: the binary rejects
PDF sources with a clear error until a text-extraction dependency is wired.

**The original is not stored by default.** Ingest can opt into keeping
it under `.abcd/memory/sources/`. The licence gate that would police publishing
such a file belongs to the lifeboat, not to launch (adr-28): launch excludes
`.abcd/**` wholesale, so it never publishes the files the gate checks. That gate
is not built, and the shipped ingest classifies restrictive licences without
ever refusing on them.

**Asking a question** synthesises an answer with per-citation provenance, every
citation naming its source class, citation and content hash. The host-produced
answer can be filed back as a new memory page, and the retrieval depth is the
caller's to set.

**The store lint** is the curator health-check over the whole store. It
always crawls the full store, rebuilds the regenerable coverage index, and
writes its findings to a run log under `.abcd/.work.local/logs/memory/`. It
mutates no memory-store state, and its exit code is the decision: blockers exit
non-zero, warnings alone exit 0, because most of what it reports is curator
advice rather than a fault.

Seven codes ship, in four families: per-page and cumulative quotation budgets
with a diagnostic for the case where coverage cannot be computed (`MQ001`,
`MQ002`, `MQ003`); source-class findings raised page by page, one where a page
rests on a single class of source and one where a page mixes classes without
saying how it weighs them against each other (`MS001`, `MS002`); a missing
licence on an external source (`ML001`); and secret or identity residue in
stored text (`MR001`). `MR001` is the read side of the write-time redactor, run
over every page, the source registry and each stored original, and over every
page name, which it judges as the write side judges a filename: split into its
parts and held to the hard-fail bar alone. It names the kind and the line, never
the span, and the lint never rewrites the store. The write-time redactor carries
a repository's armed gitleaks (`.abcd/config/gitleaks.json`): its findings are
masked with the native ones, a gitleaks run that fails refuses the ingest, and
armed with no binary installed the ingest writes on the native scanner and its
`scan_gap` names what is missing
([dependencies](../02-constraints/02-dependencies.md)).

Four of the seven can stop the run. `MR001` is the sharpest: residue in the
store is a fault, never advice. `ML001` and `MS002` join it, because a source
with no licence and a page that silently blends trust levels are both defects in
what the store claims rather than suggestions about how to curate it. `MQ002`
blocks only in its strict form, when one source's quoted coverage passes the
block threshold on unambiguous single-source attribution alone; the same
coverage reached through passages attributable to several sources is capped at a
warning, because that arithmetic cannot prove any one source was over-quoted.
Everything else, `MS001` included, is advisory and leaves the exit code at zero.

The quotation budget is applied as curation discipline at distil time and
computed at lint time. Nothing enforces it at ingest.

### What ships, and what does not

The write core (bare, ingest and ask) and the lint family are on the binary.
Contradictions are rendered by the write core's own reconciliation and surfaced
by the bare render; orphan and stale-claim audits are deferred.

Not built: a curator role on the `principle-distiller` agent (the agent and the
disembark sub-verb it feeds ship; this role does not), ingest and ask run
reports, user-scope memory outside the repo, and the automatic extraction of a
spec's modification grammar into the store at spec completion.

### Composition with adjacent surfaces

- **`/abcd:disembark`** is designed to export curated project memory and
  provenance into the lifeboat. What the lifeboat is specified to carry is the
  curated provenance surface named in
  [`02-disembark.md`](02-disembark.md), not a verbatim copy of the store;
  declaring an exact payload is deferred to the disembark spec that wires the
  packer (adr-28). The recovery-humility framing applies: the lifeboat is the
  floor of recoverable theory, not the theory.
- **`/abcd:embark`** carrying the store forward is designed behaviour. The
  source-class enum carries forward, and the receiver runs the store lint after
  unpacking to check that quotation budgets and licences have not drifted.
- **`/abcd:launch`** does not consume the licence gate at all: the public
  payload excludes `.abcd/**` wholesale as policy, so the launch preview
  surfaces no licence verdict (see
  [`04-launch.md § 2`](04-launch.md#2-curated-release-artefact-default-deny)).

**Tight coupling**, logged as a principal risk: itd-36 is structurally
non-decomposable, because the sub-verbs need the schema, the schema needs the
curator role, and the curator role needs the lint codes. A partial ship is not
meaningful.

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

- Plugin command: [`commands/library.md`](../../../../commands/library.md)
- The host-delegated halves of the corpus: [`13-consult.md`](13-consult.md),
  [`14-ingest.md`](14-ingest.md)
- The private banlist layer the guard shares: [`20-banlist.md`](20-banlist.md)
- The corpus's trust boundary: [adr-41](../../decisions/adrs/0041-corpus-trust-boundary.md),
  brief invariant 9
  ([`../02-constraints/03-invariants.md`](../02-constraints/03-invariants.md))
- [`05-internals/07-memory.md`](../05-internals/07-memory.md): the memory store's substrate spec
- [`05-internals/09-provenance-substrate.md`](../05-internals/09-provenance-substrate.md): provenance and licence subsystem
- Consuming intents: [itd-76](../../intents/shipped/itd-76-source-provenance-ledger.md) (the corpus),
  [itd-36](../../intents/shipped/itd-36-memory-unification.md) (the memory store), and
  [itd-2610090831227812](../../intents/planned/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md) (the merge)
- [`research/related-work.md § Karpathy LLM Wiki`](../../research/related-work.md#karpathy-llm-wiki--pattern-source-for-abcdmemory): pattern source for the memory store

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd library`

Sub-verbs: `abcd library add`, `abcd library ask`, `abcd library cite-check`, `abcd library declassify`, `abcd library ingest`, `abcd library init`, `abcd library ledger`, `abcd library lint`, `abcd library sync-banlist`.

| Flag | Type |
|---|---|
| `--corpus` | string |

### `abcd library add`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--alias` | stringArray |
| `--author` | stringArray |
| `--ban-authors` | bool |
| `--key` | string |
| `--keywords` | stringArray |
| `--meta` | string |
| `--permission` | string |
| `--private` | bool |
| `--public` | bool |
| `--text` | string |
| `--title` | string |
| `--type` | string |
| `--url` | string |
| `--venue` | string |
| `--year` | int |

### `abcd library ask`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--file-back` | bool |
| `--page-json` | string |
| `--top-n` | int |

### `abcd library cite-check`

Sub-verbs: none.

Flags: none.

### `abcd library declassify`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--permission` | string |

### `abcd library ingest`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--keep-original` | bool |
| `--pages-json` | string |

### `abcd library init`

Sub-verbs: none.

Flags: none.

### `abcd library ledger`

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

### `abcd library lint`

Sub-verbs: none.

Flags: none.

### `abcd library sync-banlist`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--refresh` | bool |

<!-- surface-appendix:end -->
