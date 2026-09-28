# Plan: Confidential-sources corpus and provenance ledger (convention-first scaffold)

> Forward-looking record: [itd-76](../intents/shipped/itd-76-source-provenance-ledger.md).
> Research grounding: [SOTA survey](../research/notes/2026-07-08-confidential-sources-provenance-sota.md).
> This plan documents the convention-first scaffold that dogfoods the intent
> before any Go code lands.

## Context

The user works from material they are not free to name publicly (working
papers, private repositories, NDA-shared notes). Agents need to *consult* that
material when it bears on a decision, must never *cite* it in public artifacts
(docs, commits, PR text), and every meaningful influence must be recorded in a
local-only, append-only ledger so citation remains a deliberate human act —
and so an academic paper can later be reconstructed from sources + decisions.

Chosen shape (rejected alternatives are in the SOTA survey): one **global
corpus** in a user-level directory outside every repo (`~/.abcd/sources/`),
itself a **git repository with no remote**; **convention + skill + hooks
first**, with promotion to `abcd source` verbs recorded as itd-76.

## The scaffold (user-level, outside this repo)

```
~/.abcd/sources/
  README.md               the manual: schemas, workflows, guard usage
  sources.json             CSL-JSON bibliography — metadata source of truth;
                           custom: {confidential, permission_status, keywords,
                           aliases, ban_authors, file}
  confidential/<key>/      one folder per confidential source: original.<ext>,
                           text.md (extraction), summaries, notes — together
  public/<key>/            same shape for freely citable sources
  ledger/<repo>.jsonl      append-only influence records, one file per repo
  bin/add-source           register + convert + commit (class set at ingestion)
  bin/sync-banlist         confidential entries → generated block in a repo's
                           untracked private-names banlist (itd-74 layer)
  bin/cite-guard           scan text; reports offending entries by key only
```

**Location is the classification**: confidentiality is declared once, at
ingestion, and the folder is the single source of truth — `sources.json`
mirrors it and the pattern derivation refuses on any mismatch. Derived
artifacts (summaries, notes) live inside the source's folder and inherit its
class; declassification is a `git mv` to `public/` — one visible commit.

Ledger record: `{ts, repo, decision_ref, claim, source_key, locator,
influence: supports|contradicts|method|background, cited_publicly: false}` —
one JSON line per influence; corrections are new lines; `cited_publicly` flips
only by the user's hand. Public citation is a **two-level AND**: the source's
`permission_status` grants the right; the line's `cited_publicly` exercises it
per claim.

An agent-side skill (host-level, not part of this plugin) instructs the
consultation flow: grep the corpus, read matches, append a ledger line when a
source meaningfully influences a decision, never name confidential entries in
tracked or external content, and keep `sync-banlist` current so the pre-commit
guard enforces the rule mechanically.

## Guard chain

1. `sources.json` marks an entry `confidential: true` with `aliases`.
2. `sync-banlist` derives case-insensitive patterns — title and aliases
   always; full author names only by per-source opt-in
   (`custom.ban_authors: true`, for a collaboration that is itself secret;
   the common confidential types — own submitted work, purchased reports,
   private repos — are protected by title/aliases, and banning their authors
   would mostly ban legitimate names) — and maintains a fenced generated
   block in the repo's untracked `.abcd/.work.local/private-names.txt`;
   hand-added lines survive.
3. The committed `.githooks/pre-commit` guard (itd-74's private layer)
   **auto-refreshes** the generated block when the corpus is present (no-op
   otherwise — CI, fresh clones), then blocks any staged line matching — the
   confidential string never enters history and the guard is never stale.
4. `cite-guard` clears prose before it is shared anywhere git does not gate.

**Boundary (stated, not hidden):** the mechanical layer blocks literal
identifying strings only. Paraphrase that identifies a source without naming
it is handled behaviourally (the skill forbids identifying description) and by
human review of every publish. Durability: the no-remote corpus rides the
machine backup, plus occasional `git bundle` snapshots to an encrypted
external volume; multi-machine ledger ownership is deferred until a second
machine exists.

## Team sharing (citation data only)

A repo is a team surface even when every corpus is personal. Public entries'
*citation data* — never documents, never ledgers — flow through the committed
`.abcd/work/references.json` (CSL-JSON): `share` writes an entry there,
refusing `confidential: true` mechanically; `ingest` imports the repo's shared
entries into a teammate's local corpus. Recorded in itd-76; scripts are built
when a second contributor exists. The hand-curated
`development/research/_references.md` registry stays human-owned and may later
derive from this file.

## Verification performed (2026-07-08)

- Fake confidential entry added; `sync-banlist` generated four patterns; a
  staged file containing the fake title was **blocked** by the pre-commit
  guard; removing the entry and re-syncing shrank the generated block to zero
  with hand-kept lines intact. Re-running `sync-banlist` is idempotent.
- Consult flow: keyword grep over `corpus/` surfaced the seeded entry; the
  frontmatter `key:` mapped the hit to its bibliography entry.
- Ledger: appended record validated with `jq -s`; the corpus repo's git log
  shows the append and the later removal as separate commits (history
  preserved — the tamper-evidence property).
- `cite-guard`: exit 1 with key-only report on offending text; exit 0 clean.

## Out of scope (recorded, not built)

- **Paper pipeline**: Quarto, two project profiles (internal/public); the
  public render is proven clean twice — structurally (it renders from a
  generated bibliography that omits confidential entries, so an unpermitted
  key fails the build) and by a deterministic post-render check of the
  output's citations against both gates. Build when the ledger has real
  entries.
- **`abcd source` verbs** (add / consult / ledger / share / ingest /
  cite-check): itd-76, as a standalone core domain (itd-16's audit chain is a
  possible later ledger backend, not a dependency). The user-level home it
  reads is itd-77's surface.
- **Retrieval upgrades** (SQLite FTS5, embeddings): only if grep over the
  corpus gets noisy; the survey records why grep is the default.
