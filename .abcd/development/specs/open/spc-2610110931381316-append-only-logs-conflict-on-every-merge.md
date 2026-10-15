---
id: spc-2610110931381316
slug: append-only-logs-conflict-on-every-merge
intent: itd-2609151138388536
origin: researcher-authored
production_mode: hand-written
---
# append-only-logs-conflict-on-every-merge

## Summary

This spec delivers
[itd-2609151138388536](../../intents/planned/itd-2609151138388536-append-only-logs-conflict-on-every-merge.md):
a decision is a record of its own, `dcn-<id>-<slug>.md` in the flat folder
`.abcd/work/decisions/`, and the single log `.abcd/work/DECISIONS.md` is
retired. The rule is
[adr-2610110925465593](../../decisions/adrs/2610110925465593-decisions-are-one-flat-file-each-and-the.md),
which supersedes
[adr-2609151138420062](../../decisions/adrs/2609151138420062-the-decisions-log-is-a-folder-of-minted.md)
and drops its symlink.

The approach, in plain words:

- **A sixth family on the existing seam.** `dcn` joins the record families the
  resolver, record-lint and the site already read. Its ids come from the adr-45
  mint (`internal/core/recordid`), its slugs are cut at `recordid.MaxSlugLen`
  (40), and one package, `internal/core/decisions`, owns the folder: the path,
  the writer and the reader.
- **Recording writes one new file and nothing else.** A verb mints the record
  (`abcd decide "<text>"`, decision 17). Nothing is appended anywhere,
  so two branches that each record a decision merge with nothing to reconcile.
- **History is read, never committed.** The verb's bare form and a site page
  list every decision in date order, read from the folder each time. No index
  file exists.
- **Readers move before the history does.** Every piece of code that reads or
  writes the log (ideate, ahoy, the repository lint, the implement brief, the
  lifeboat, the cold-reading assembler) is moved to the folder first. A managed
  repository's frozen `DECISIONS.md` is still read where it exists, and never
  written.
- **abcd's own log moves once, by a script that is not part of abcd.** A
  one-off Go program in the local tier splits the 523 entries (count at
  6f0589fc3) into records and rewrites every citation it can map with
  certainty. Its output is committed, and the program itself is not. The
  citations it cannot map go on a review list in the pull request, and a person
  resolves each one.
- **The gate goes with the file.** The decisions-append gate (DA001–DA004)
  refuses any change that deletes the log's lines, and its own contract says the
  change that graduates the log retires the gate in the same diff. So the gate's
  code and wiring leave in the migration change (decision 5).

How each acceptance criterion is met, numbered A1 to A8 in the order the intent
lists them:

| Criterion | Met by |
|---|---|
| A1, recording writes one file with an adr-45 id and appends to nothing | Step 2: the writer in `internal/core/decisions` mints through `recordid.Minter`; `TestRecordWritesOneFileAndNothingElse` checks `git status` shows exactly one new path. |
| A2, the listing and the site page show every entry in date order with its id, derived from the folder | Step 2 (the listing, `TestListingIsDateOrderedAndDerived`) and step 5 (the site page, `TestDecisionLogPageListsEveryEntry`, with more records than the development page's cap). |
| A3, `DECISIONS.md` gone, no live path to it, each dated citation points at its entry's file, and uncertain ones listed for review | Step 6: the script deletes the file and rewrites the citations it can map; the review list goes in the pull request; `TestNoLiveDecisionsLogPath` holds the tree to it from then on. |
| A4, two branches that each record a decision merge without conflict and without a merge attribute or driver | Step 2, `TestTwoBranchesRecordingMergeCleanly`, a real git test with no `.gitattributes`. Step 6 removes the union line from abcd's own `.gitattributes`. |
| A5, a managed repository's existing `DECISIONS.md` untouched by install, new decisions to the folder, no merge attribute | Step 3, `TestInstallLeavesAFrozenLogByteForByte` and `TestInstallScaffoldsNoMergeAttribute`. |
| A6, adoption with no log gets the folder and no single file | Step 3, `TestInstallSeedsTheDecisionsFolder`. |
| A7, the gate gone from code, CI, preflight and the conventions | Step 6 deletes it with its wiring; `TestNoLiveDecisionsLogPath` also refuses `decisions-append` and `lint-decisions` on the live surfaces. |
| A8, every entry its own file with date and text verbatim, rendered in date order, count matching | Step 6: the script's round trip (each record reproduces its entry byte for byte), its count check, and a diff of the listing against the old file. The order part waits on open question 2. |

## Design

### The record

A decision record is one file, `.abcd/work/decisions/dcn-<id>-<slug>.md`:

```markdown
---
id: dcn-<yymmddHHMMSS><rrrr>
slug: <at most 40 characters>
date: <YYYY-MM-DD>
---
<YYYY-MM-DD> — <the decision, in the words it was recorded in>
```

- **The id** is the adr-45 grammar, `dcn-` plus a twelve-digit UTC stamp plus
  four random digits. The verb mints it through `recordid.Minter`, retrying a
  clash the same way `decide` does (`mintRetryBudget`, exclusive create under
  the store's mint lock).
- **The date** comes from the stamp, never from a second clock reading.
  `decide`'s `dateFromStamp` moves into `recordid` so both writers share it.
- **The slug** is `recordid.Slug(text, recordid.MaxSlugLen)` over the text
  after its leading date and separator. A text with nothing slug-able in it is
  refused, as `decide` refuses such a title.
- **The body** starts with the date, as every entry of the old log did. A
  migrated record's body is the entry exactly as written. The verb writes
  `<date> — <text>`, so a listing prints every body the same way and each file
  reads like a line of the old log.
- **No title heading.** An entry never had one. The site and the listing show
  the body itself.
- **Immutable, like every other record.** A correction is a new record. Nothing
  enforces this mechanically, and adr-2609151138420062 already named that as
  the cost of retiring the gate: if it needs enforcing, it needs it for every
  family, not just this one.

### The family in the shared code (step 1)

`dcn` is added everywhere the families are listed, so a decision is a record
like any other before anything writes one:

- `internal/core/recordid`: `familyRoots` gains `{"dcn", ".abcd/work/decisions"}`
  through a new constant `DecisionsRelDir`. `CitedIDRe` and `citedIDPartsRe`
  admit `dcn`.
- `internal/core/lint`: `recordStores` gains a flat `dcn` store (no buckets,
  `fileFamily: "dcn"`, filename `dcn-<N>-<slug>.md`, required fields `id`,
  `slug` and `date`, `readerFailsClosed`). The handle patterns in `schema.go`,
  `crossstore.go` and `prosecitations.go` admit `dcn`, so a prose citation of a
  decision that no file holds is a blocker, exactly as it is for an issue. A
  record whose `date` disagrees with its id's stamp, or with the date its body
  starts with, is a finding.
- `.abcd/record-lint.json`: `dcn` joins `record_stores` in `record_schema`,
  `prose_citation_resolves` and `context_citation_currency`.
- `internal/core/site/recordjson.go`: `bodyHandleRe` admits `dcn`, so a body
  mention of a decision becomes a mention edge.
- `internal/core/record`: `abcd dcn-<id>` says what the record is and where it
  sits, the way `abcd iss-<id>` does.
- `internal/core/decisions` (new): `RelDir`, `Parse` (one file into a
  `Decision`) and `List` (every record in the folder, ordered by id). `List`
  reads an absent folder as an empty list.

The prefix is `dcn`, not `dec`: the lifeboat already keys a line of a frozen
log as `dec-L<line>` (`internal/core/lifeboat/graveyard.go`), and a distinct
prefix keeps the two apart for a reader as well as for the grammar (decision
16). Step 4 keeps both kinds of key (decision 14).

### Recording and listing (step 2)

`decisions.Record(repoRoot, text)` is the one writer. The verb's front door and
ideate both call it (one-canonical-primitive). It:

- resolves the repository root, as `decide` does, and refuses outside a
  checkout;
- refuses when the shared tier `.abcd/work/` does not exist, naming
  `ahoy install`. It creates `decisions/` inside an existing shared tier,
  which is abcd's own namespace;
- mints, cuts the slug, and writes the file by exclusive create;
- touches no other path. That is A1, and A4 follows from it.

The listing is the verb's bare form, `abcd decide` with no operand (decision 17). It
prints every record newest first (decision 21), one block
per record: the id, then the body. `--json` returns `[{id, date, slug, path,
text}]`. It reads only the folder. Where a frozen `.abcd/work/DECISIONS.md`
also exists (a managed repository adopted before this change), it ends with one
line naming that file as earlier history it does not list. It writes nothing.

`TestTwoBranchesRecordingMergeCleanly` is the A4 proof. It sets up a temporary
repository with no `.gitattributes` and two branches cut from one base, records
one decision on each through `Record`, merges both into a third branch, and
asserts: both merges exit 0, both files are present, and
`git check-attr merge` reports the attribute unspecified for both paths.

### Writers and adoption move to the folder (step 3)

- **ideate** (`internal/core/ideate`). Today it appends a dated pointer line
  to the log under an advisory lock, and refuses with `MissingDecisionsError`
  when there is no log. It now calls `decisions.Record` with the same pointer
  text, the lock and the read-modify-write go, and `MissingDecisionsError` goes
  with them (the writer's own refusal covers a missing shared tier). The JSON
  `DecisionPath` names the new record. `commands/ideate.md` searches the folder
  for an earlier entry instead of the file, and
  `.abcd/development/brief/04-surfaces/18-ideate.md` says the same.
- **ahoy** (`internal/core/ahoy`). The shared tier's seed becomes
  `.abcd/work/decisions/README.md`: a few lines on what the folder holds and
  which command records and lists it. It is static, and recording a decision
  never edits it. The gap `skeleton.decisions_ledger_missing` becomes
  `skeleton.decisions_folder_missing`, raised when the shared tier has no
  decisions folder. The install summary line says so. An existing
  `DECISIONS.md` is never opened for writing. Nothing writes `merge=` into
  `.gitattributes`: the one attribute ahoy writes is the hooks' EOL pin
  (`banlist_scaffold.go`), and a test holds it there. `docs/how-to/install.md`
  changes in the same step (spec-moves-with-the-surface). So does whichever
  page then carries the adoption workflow: `commands/prepare-this-repo.md`, or
  `commands/ahoy.md` if the verb split's step 4
  ([spc-2610100613109045](../closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md))
  has already folded it there.
- **The repository lint's decision-durability rule**
  (`internal/core/repolint/rule_decisions.go`). It passes when
  `.abcd/work/decisions/` is committed. A frozen `DECISIONS.md` with no folder
  is a warning naming `ahoy install`. The sharper message for decisions kept
  only in the local tier stays. The brief's rule row
  (`.abcd/development/brief/04-surfaces/16-lint.md`) changes with it.

### Readers move to the folder (step 4)

Each of these reads the folder, plus the frozen file where one exists, through
`decisions.FrozenLog`. That is the one constant left naming the old path, and
it is documented as a managed repository's frozen history, read-only
(decision 7).

- **The implement brief** (`internal/core/implement/loop/brief.go`,
  `issuebrief.go`). "Entries naming the intent or spec" become the records,
  listed at the base ref through `at.list` on the folder, whose body names the
  id, followed by any frozen-file entries naming it. `DecisionsLogRel` goes.
  The brief's section heading names the folder.
  `.abcd/development/brief/04-surfaces/34-build.md` changes with it.
- **The lifeboat** (`sources_native.go`, `graveyard_abandoned.go`,
  `graveyard.go`). The folder becomes a native source next to the frozen file.
  A rejected option found in a record is keyed by the record's id. One found in
  the frozen file keeps `dec-L<line>`.
- **The cold-reading assembler** (`internal/core/reading/include.go`) lists
  `.abcd/work/decisions/` among the paths absent from the positive walk,
  replacing the file. `.abcd/development/readings/README.md` and the evals'
  fixtures (`evals/coldreading_oracle_test.go`, `coldreading_fixture_test.go`,
  `coldreading_coverage_test.go`) follow.
- **The scribe's boundary** (`internal/core/scribe/fixture_test.go`,
  `internal/core/lint/scribecontract_test.go`). The working-material sentinel
  and the path case move to a decision record.
- **The sources ledger's `--decision` flag** (`internal/surface/cli/source.go`,
  `commands/consult.md`). The help text says a decision id, an ADR or intent
  id, or free text. `docs/reference/cli/commands.md` is regenerated. If the
  verb split's step 5 has renamed `source` to `library` by then, this edit
  lands on the renamed file.
- **Tests that used the log only as an example of a tracked file**
  (`internal/gitutil/gitignore_test.go`, `internal/core/ahoy/gitignore_test.go`,
  `local_tier_test.go`, `prompt_help_test.go`,
  `internal/surface/cli/ahoy_prompt_help_test.go`, `lint_surface_test.go`) use
  the folder's README instead.

### The site page (step 5)

The site reads the record graph that `lint.LoadRecordGraph` builds, so once
step 1 lands the `dcn` records are nodes. Step 5 decides where they show:

- **Publication is opted into**, as the issue ledger's is. The folder sits in
  the working tier, and adr-32 makes publishing that tier a per-repository
  choice. `RecordOpts` gains `decisions`, and without it no `dcn` node reaches
  the export. abcd's `.abcd/site.json` sets it.
- **One page lists them all**, newest first (decision 21), grouped by month. Each month is a
  disclosure, and every entry is in the HTML whether its month is open or not.
  Each row is the id (linking the record's own page), the date and the full
  body. The page has no cap, because A2 says every entry. A test builds more
  records than `developmentDeckCap` and finds every one.
- **Each record gets its page** through the existing `recordPage`.
- **They stay off the development page and the relationship chart.** Both are
  about stores that move, and 523 one-line nodes would bury them. The
  dashboard gets one tile with the count, linking the page.
- The page is labelled "Decision log", at `record/decision-log/`; the ADR tile
  keeps its "decisions" label in `site-src/ui.json` (decision 19).

### The migration script (step 6)

The product thinker ruled that abcd's own migration is a one-off script, never
a verb or any code in abcd, and abcd never runs it in a managed repository;
that repository's owner may run it there by hand (decision 20).

- **Where it lives:** `.abcd/.work.local/scratch/decisions-migrate/main.go`,
  in the checkout's local tier. It is gitignored, so it is never committed and
  never shipped. It is a Go program so that it can call the canonical
  primitives instead of copying them: `recordid.Slug`, `recordid.MaxSlugLen`,
  the id validator and `decisions.Parse`. On 2026-10-11, `go list` resolved a
  package at that path importing `internal/core/recordid` with no error, so it
  runs as `go run ./.abcd/.work.local/scratch/decisions-migrate` from the
  checkout root. A dry run is the default, and `-apply` writes.
- **What it does with `-apply`:**
  1. Refuses unless the working tree is clean.
  2. Parses the log. The header is every line above the first top-level
     `- <YYYY-MM-DD>` bullet. An entry is one such bullet and every line after
     it, up to the next top-level bullet, minus trailing blank lines. Two
     entries hold a blank line followed by indented text, and it stays theirs.
     Any line below the header that is neither blank nor part of an entry makes
     the script refuse.
  3. Writes one record per entry (decision 3 sets the ids). The body is the
     entry with its leading `- ` and its two-space continuation indent removed,
     and nothing else changed. The header is not migrated: it describes the
     file and the gate, and both are retired. A record already in the folder
     (one recorded by the verb since step 2) is left alone, and an id clash
     makes the script refuse.
  4. Verifies before touching anything else. The number of records written
     equals the number of entries parsed. Every record, given back its prefix
     and indent, reproduces its entry byte for byte. The listing's `--json`,
     run through `go run ./cmd/abcd`, holds every migrated id in the expected
     order.
  5. Removes the log with `git rm`.
  6. Maps citations in every tracked text file outside the history set (see
     the next section) and rewrites each mapped one. It skips the records whose
     subject is the log itself, and prints their names in the report.
  7. Stages everything, never commits, and writes its report to
     `.abcd/.work.local/logs/decisions-migrate-<stamp>.md`: the entry count
     before and after, the round-trip verdict, the number of citations
     rewritten in each class, and the review list.
- **Run a second time** on its own result, it refuses with nothing to do,
  because the log is gone.
- **What the landing change commits** is the script's output (the records, the
  deletion of the log and the rewritten citations), each review item resolved
  by hand, and the hand changes listed in step 6. Nothing of the script itself.
- **How the change shows its work:** the pull request body carries the entry
  count before and after (523 and 523 at 6f0589fc3, more by landing), the
  round-trip verdict, the citation counts per class, and the review list with
  the resolution of each item. It also carries the script's source in a
  collapsed block with its sha256, so a reviewer can rerun it on the parent
  commit and get the same output.

### Mapping citations

A citation is mapped only when exactly one entry can be meant. Otherwise it
goes on the review list. Nothing is guessed (A3).

| Class | Example shape | Mapped when |
|---|---|---|
| Line number | the file's path followed by `:<line>`, often with a quote | The line falls inside one entry, and any quoted text appears in that entry. The gate kept old lines in place, but the header changed length over time, so the quote is the check. |
| Date | the file's name beside a date, or "the entry of that date" with the date nearby | The date has one entry. Of 66 dates, many have several (the most, 31), so this alone maps only some. |
| Date plus a key | as above, plus a ruling label (CJ1b, BZ2), a record id or a quoted phrase | Exactly one entry of that date carries every key. |
| Same commit | "the entry landing with this change" | The commit that added the citing line added exactly one entry (`git log -L` on the citing line, then that commit's diff of the log). |
| Link | a markdown link to the file | Mapped by the rules above when its text names an entry. A link to the file as a whole goes on the review list. |
| Names no entry | "recorded in" the file, with no date or key | Never. On the review list. |

A rewritten citation becomes the record's handle. In Markdown it is linked by
a relative path to the record's file wherever the original was a path or a
link. In Go source it is the bare handle, whether in a comment or a string; the
message in `lint.go` that cites an entry and the test in `lint_test.go` that
asserts it are rewritten together. The person resolving the review list picks
one of four outcomes for each item and records it in the pull request:

- map it to an entry, after reading the candidates;
- point it at the folder, where the sentence means the place and not an entry;
- leave it as history, where it narrates what the log was;
- reword the sentence.

### What is left as history

These keep their mentions word for word. Each is a verbatim record of a past
state, and a gate or a convention protects it:

- `CHANGELOG.md`, five lines in released sections. The changelog is derived,
  and `## [Unreleased]` stays empty.
- `.abcd/work/reviews/**`, 29 lines in nine receipts. The reviews charter's
  append-only rule (RD002) forbids editing them, and `internal/core/relink`
  already treats them as history.
- `.abcd/development/research/data/**`, four lines in raw run outputs.
- `internal/core/guard/testdata/corpus/repo-mined.txt`, one line in a mined
  corpus fixture.
- **Records whose subject is the log itself**: the intent, its two ADRs, this
  spec, iss-2609100507439414, and the issues about the gate
  (iss-2608271804494867, iss-2608291814575169). They talk about the log rather
  than citing an entry of it. Rewriting them would change what they say.

`TestNoLiveDecisionsLogPath` (step 6, a repository-tree test) enforces the line
between live and history. It reads `git ls-files` and refuses three things:

- the full path `.abcd/work/DECISIONS.md` on any live surface: code, `cmd/`,
  `evals/`, `commands/`, `agents/`, `docs/`, the root prose files, `.github/`,
  `scripts/`, the `Makefile`, `.gitattributes`, `.gitignore`, the `.abcd/*.json`
  configurations, the brief and the principles;
- a markdown link to `DECISIONS.md` in any file outside the history set;
- the dated-citation shapes in any file outside the history set.

On the live surfaces it also refuses `decisions-append` and `lint-decisions`
(A7). Two places are exempt: the `internal/core/decisions` package, which holds
`FrozenLog`, and lines that say "retired" or "frozen", the same per-line
allow-word convention record-lint's `banned_tokens` uses. Test fixtures that
build a frozen log name it through `decisions.FrozenLog`.

### Every reader, and its disposition

At 6f0589fc3, `git grep -c DECISIONS` outside the log itself counts 529 lines
in 300 files. Grouped by what happens to them:

| Reader | Lines (files) | Disposition | Step |
|---|---|---|---|
| ideate: the writer, its lock, its refusal, `redact.go`'s comment, `record_test.go` | 3 (3) | Code: writes through `decisions.Record` | 3 |
| ahoy: the seed, its gap, the install summary, the `apply.go` comment | 3 (3) | Code: seeds the folder | 3 |
| `repolint/rule_decisions.go` and its tests (`rules_test.go`, `rule_refusal_test.go`) | 15 (3) | Code: folder, or a frozen-file warning | 3 |
| `repolint/rule_docs.go`, `rule_layout.go` (comments citing an entry) | 2 (2) | Script | 6 |
| implement loop: `brief.go`, `issuebrief.go`, `brief_test.go` | 3 (2) | Code: reads the folder plus the frozen file | 4 |
| lifeboat: `sources_native.go`, `graveyard_abandoned.go`, `graveyard.go`, `synthesis_types.go`, and the tests `graveyard_abandoned_test.go`, `graveyard_notices_test.go`, `embark_test.go` | 18 (7) | Code: folder plus frozen file; fixtures use `FrozenLog` | 4 |
| the cold-reading assembler, its tests, the readings README, the three evals fixtures | 6 (6) | Code and fixtures: the folder replaces the file | 4 |
| scribe: `fixture_test.go`, `scribecontract_test.go` | 3 (2) | Fixtures: a decision record | 4 |
| `surface/cli/source.go` flag help, `commands/consult.md`, `docs/reference/cli/commands.md` | 3 (3) | Code and page; the reference is regenerated | 4 |
| tests using the log as an example tracked path (gitutil, ahoy, cli) | 11 (6) | Fixtures: the folder's README | 4 |
| the gate: `lint/decisionsappend.go` and tests, `cmd/record-lint` (`decisionsappend.go`, its test, `main.go`), `Makefile`, `.github/workflows/ci.yml` | 17 (6) | Deleted with the log | 6 |
| `relink/relink.go` and test, `cmd/record-slug-rename/main.go` and test (the append-only maps) | 7 (4) | Code: the entry goes (the path no longer exists); decision records are relinked like any record | 6 |
| `.gitattributes` (the union line and its comment) | 2 (1) | Removed | 6 |
| conventions: `AGENTS.md`, `.abcd/README.md`, `.gitignore`, `.github/CONTRIBUTING.md` (gate list) | 4 (4) | Hand: name the folder and the verb; drop the gate | 6 |
| `.abcd/rules.json`, `.abcd/record-lint.json`, `.abcd/prose-citations-baseline.json` (each cites an entry) | 3 (3) | Script | 6 |
| `.abcd/work/CONTEXT.md` (cites an entry) | 1 (1) | Script | 6 |
| the brief: `00-meta.md`, `02-constraints/04-naming.md`, `04-surfaces/24-decide.md`, `05-internals/04-universal-patterns.md` | 4 (4) | Script, or hand for the patterns table, which names the file as a place | 6 |
| the brief: `05-internals/06-lint.md` (the gate's description), `glossary/core/ledger.md` | 2 (2) | Hand: with the gate's removal and the shape | 6 |
| the brief: `16-lint.md`, `18-ideate.md`, `34-build.md` | 5 (3) | Hand, with the behaviour that changes | 3, 4 |
| `commands/ideate.md`, and `commands/prepare-this-repo.md` or its successor | 8 (2) | Hand, with the behaviour that changes | 3 |
| `docs/how-to/install.md` | 1 (1) | Hand, with ahoy | 3 |
| principles: `guidance-carries-its-evidence-and-its-purpose.md` (cites an entry), `one-writer-per-file.md` (narrates the conflict) | 2 (2) | Script; review list | 6 |
| code comments citing an entry by date or ruling (guard, intent, issueschema, launch, layered, oracle, rules, update, condition, decide, surface/cli `decide.go` and `sentences_test.go`, lint `config.go`, `lint.go` with `lint_test.go`) | 29 (24) | Script; any it cannot map, on the review list | 6 |
| `intent/prepass_test.go`, `intent/questions_test.go`, `launch/commandladder_test.go`, `launch/gates_test.go` (fixture prose naming the file) | 5 (4) | Hand: fixture text that names no live path | 6 |
| `internal/adapter/scanner/network.go` (`DECISIONS.local.md`, an unrelated file name) | 1 (1) | Untouched: not this file | none |
| records: intents, specs, plans, research notes, ADRs, issues, roadmap, readings | 342 (194) | Script for citations; review list for the rest; subject records left as history | 6 |
| history set (changelog, reviews, research data, the guard corpus) | 39 (15) | Left verbatim | none |

The counts are a census at one commit. The script's report at landing is the
authoritative tally, and a line that appears after the census is caught by the
rerun (next section) or, once step 6 lands, by `TestNoLiveDecisionsLogPath`.

### Concurrency at landing

Other sessions append to the log, and cite it, while this is built. So step 6's
generated part is never trusted from the day it was first run:

1. Step 6 is built and reviewed on its branch like any change. Its hand commits
   and its generated commit are kept separate.
2. At landing, the session cutting it announces a window: on the forge, in the
   peer note, and to any peer session found, after checking
   `go run ./cmd/abcd peers`. The announcement asks peers not to merge a change
   that touches the log until step 6 is in.
3. On a fresh default branch, the hand commits are rebased, the generated
   commit is dropped, and the script is rerun with `-apply`. Its ids are
   deterministic (decision 3), so every entry the first run saw gets the same
   file again. The only new output is for entries and citations added since.
   The review list is diffed against the reviewed one, and each new item is
   resolved before the merge.
4. The change merges as the next one in, with the tree proven clean
   immediately before the act.
5. Afterwards, a change still in flight that appends to the log hits a
   modify/delete conflict, which is loud: its author records the decision
   through the verb instead. A change that cites the old path or a dated entry
   fails `TestNoLiveDecisionsLogPath` in CI.

## Steps

1. The `dcn` family is a record family: resolver, record-lint, the site's mentions, the record dispatch and the folder reader
   - packages: internal/core/recordid (resolve.go, a DecisionsRelDir constant, dateFromStamp moved in from decide), internal/core/decide (calls the moved helper), internal/core/lint (schema.go, crossstore.go, prosecitations.go), .abcd/record-lint.json (record_stores in record_schema, prose_citation_resolves and context_citation_currency), internal/core/site (recordjson.go bodyHandleRe), internal/core/record (dispatch for a dcn id), internal/core/decisions (new: RelDir, Parse, List)
   - tests: TestResolverFindsADecisionRecord, TestRecordSchemaRequiresDecisionFields (a record missing date, and one whose date disagrees with its stamp or body, each a finding), TestProseCitationOfAnAbsentDecisionIsABlocker, TestListOrdersByIDAndReadsAnAbsentFolderAsEmpty, TestRecordDispatchDescribesADecision, each watched fail first; record-lint clean on the tree, which holds no dcn record yet
   - the slug cap tree test (recordid/slugcap_tree_test.go) walks familyRoots and fails on a store holding no records, so it would break here: it exempts the `dcn` store from that one assertion until step 6, which removes the exemption once the folder holds the migrated records
2. Recording a decision writes one file, and the bare verb lists them
   - packages: internal/core/decisions (Record, the mint lock, the listing's text and JSON forms), internal/surface/cli (the recording verb, its bare listing and its help placement, per open question 1), commands (the verb's page, per open question 1), .abcd/development/release/surface.json and docs/reference/cli/commands.md regenerated, .abcd/development/brief/04-surfaces (the verb's chapter, and README row), .abcd/rules.json (DOGFOODING recall list, if the answer adds a verb)
   - tests: TestRecordWritesOneFileAndNothingElse (A1), TestTwoBranchesRecordingMergeCleanly (A4), TestListingIsDateOrderedAndDerived (A2, records created out of id order, listing in id order, no tracked file changed by listing), TestListingNamesAFrozenLog, TestRecordRefusesWithoutTheSharedTier, TestRecordRefusesAnUnslugableText, each watched fail first; the menu, parity, snapshot and dogfooding tests pass after go generate ./internal/surface/cli
3. ideate, ahoy and the repository lint move to the folder
   - packages: internal/core/ideate (ideate.go, record.go, redact.go comment), commands/ideate.md, internal/core/ahoy (record_tiers.go, install_summary.go, apply.go comment), internal/core/repolint (rule_decisions.go), docs/how-to/install.md, commands/prepare-this-repo.md or commands/ahoy.md (whichever carries the adoption workflow when this lands), .abcd/development/brief/04-surfaces (16-lint.md, 18-ideate.md, 01-ahoy.md where it lists the seeds)
   - tests: TestIdeateRecordsItsPointerAsADecision (and the old missing-log refusal retired with its ruling in the commit), TestInstallSeedsTheDecisionsFolder (A6), TestInstallLeavesAFrozenLogByteForByte (A5, compared by hash), TestInstallScaffoldsNoMergeAttribute (A5), TestDecisionDurabilityPassesOnTheFolder, TestDecisionDurabilityWarnsOnAFrozenLogAlone, each watched fail first; TestEveryCommittedTierHasASeed passes with the new seed
4. The implement brief, the lifeboat, the cold-reading assembler, the scribe and the sources ledger read the folder
   - packages: internal/core/decisions (FrozenLog), internal/core/implement/loop (brief.go, issuebrief.go), internal/core/lifeboat (sources_native.go, graveyard_abandoned.go, graveyard.go), internal/core/reading (include.go), .abcd/development/readings/README.md, evals (the three cold-reading fixtures), internal/core/scribe (fixture_test.go), internal/core/lint (scribecontract_test.go), internal/surface/cli (source.go flag help), commands/consult.md, docs/reference/cli/commands.md regenerated, .abcd/development/brief/04-surfaces/34-build.md, and the six tests that used the log as an example tracked path
   - tests: TestBriefQuotesDecisionsNamingTheIntent (folder and frozen file both), TestIssueBriefQuotesDecisionsNamingTheIssue, TestLifeboatReadsTheDecisionsFolder, TestGraveyardKeysARecordByItsID, each watched fail first; the existing frozen-log lifeboat tests pass on FrozenLog; the cold-reading evals lane (make evals-cold-reading) and the reading tests pass with the folder excluded
   - if the non-test diff passes about 800 lines, the lifeboat moves to a step of its own
5. The site lists every decision on one page
   - packages: internal/core/site (manifest.go RecordOpts, recordjson.go, explorer.go the page, its route, its navigation entry and the dashboard tile; development.go and timeline.go keep dcn nodes out), site-src/ui.json (the page's label, per open question 3), .abcd/site.json (the opt-in)
   - tests: TestDecisionLogPageListsEveryEntry (more records than developmentDeckCap, newest first, each with its id and full text), TestDecisionsAreOptedInto (absent without the switch), TestDevelopmentPageAndChartLeaveDecisionsOut, each watched fail first; make site-render clean, and the export built twice still identical
6. abcd's log becomes records, every citation is rewritten, and the gate and the union line go
   - packages: .abcd/work/decisions (generated by the script), .abcd/work/DECISIONS.md (deleted), every tracked file holding a citation (generated, plus hand resolutions of the review list), .gitattributes, AGENTS.md, .abcd/README.md, .gitignore, .github/CONTRIBUTING.md, .github/workflows/ci.yml (the two gate steps and the Go setup only they use), Makefile (lint-decisions and its place in preflight), cmd/record-lint (the decisions-append mode, its file and test), internal/core/lint (decisionsappend.go and its test deleted), internal/core/relink and cmd/record-slug-rename (the append-only entries and their fixtures), .abcd/development/brief/05-internals/06-lint.md, .abcd/development/brief/glossary/core/ledger.md, the four fixture-prose tests, internal/core/recordid (the slug cap test's `dcn` exemption removed), a new repository-tree test file for TestNoLiveDecisionsLogPath
   - tests: TestNoLiveDecisionsLogPath (A3, A7), watched fail on the tree before the script runs and pass after; the script's round trip, count check and listing diff, reported in the pull request (A8); go test ./... and record-lint, docs-lint and site-render clean; make preflight runs eight gates, with no lint-decisions
   - the gate leaves here, not after: its contract refuses any change that deletes the log's lines and requires the change that graduates the log to retire it in the same diff (decision 5)
   - lands by the protocol under Concurrency at landing
   - the script lives at .abcd/.work.local/scratch/decisions-migrate and is never committed; the pull request body carries its source, its sha256, the counts, the round-trip verdict and the review list with each resolution
   - the non-generated diff is the hand edits, the tree test and the review resolutions; the deleted gate files (about 1,700 lines) are pure deletions and are counted apart from it
7. The docs-fidelity review and the close
   - packages: .abcd/development/intents (the intent moves to shipped by the close), .abcd/work/issues (iss-2609100507439414 resolved)
   - tests: a docs review recorded for HEAD with go run ./cmd/abcd docs fidelity record; then go run ./cmd/abcd spec close spc-2610110931381316, with a Delivers: itd-2609151138388536 trailer (the intent declares impact: breaking, so no --impact) and Resolves: iss-2609100507439414

## Footprint

- packages: internal/core/recordid, internal/core/decide, internal/core/decisions (new), internal/core/lint, internal/core/record, internal/core/site, internal/core/ideate, internal/core/ahoy, internal/core/repolint, internal/core/implement/loop, internal/core/lifeboat, internal/core/reading, internal/core/scribe, internal/core/relink, internal/surface/cli, internal/gitutil (a test), cmd/record-lint, cmd/record-slug-rename, evals, commands (the verb's page, ideate.md, consult.md, prepare-this-repo.md or ahoy.md), site-src/ui.json, .abcd/site.json, .abcd/record-lint.json, .abcd/rules.json, .abcd/README.md, .abcd/development/release/surface.json, .abcd/development/brief (04-surfaces, 05-internals, glossary, and the chapters the script rewrites), .abcd/development/readings/README.md, .abcd/work/decisions (new), .abcd/work/DECISIONS.md (deleted), every record the script rewrites, docs/how-to/install.md, docs/reference/cli/commands.md, AGENTS.md, .gitattributes, .gitignore, Makefile, .github/workflows/ci.yml, .github/CONTRIBUTING.md
- tests: TestResolverFindsADecisionRecord, TestRecordSchemaRequiresDecisionFields, TestProseCitationOfAnAbsentDecisionIsABlocker, TestListOrdersByIDAndReadsAnAbsentFolderAsEmpty, TestRecordDispatchDescribesADecision, TestRecordWritesOneFileAndNothingElse, TestTwoBranchesRecordingMergeCleanly, TestListingIsDateOrderedAndDerived, TestListingNamesAFrozenLog, TestRecordRefusesWithoutTheSharedTier, TestRecordRefusesAnUnslugableText, TestIdeateRecordsItsPointerAsADecision, TestInstallSeedsTheDecisionsFolder, TestInstallLeavesAFrozenLogByteForByte, TestInstallScaffoldsNoMergeAttribute, TestDecisionDurabilityPassesOnTheFolder, TestDecisionDurabilityWarnsOnAFrozenLogAlone, TestBriefQuotesDecisionsNamingTheIntent, TestIssueBriefQuotesDecisionsNamingTheIssue, TestLifeboatReadsTheDecisionsFolder, TestGraveyardKeysARecordByItsID, TestDecisionLogPageListsEveryEntry, TestDecisionsAreOptedInto, TestDevelopmentPageAndChartLeaveDecisionsOut, TestNoLiveDecisionsLogPath; the TestDecisionsAppend* suite retired with the gate; the migration script's round trip, count check and listing diff; record-lint, docs-lint, site-render and the cold-reading evals lane clean

## Decisions (technical facilitator, 2026-10-11)

Each of these was settled while writing the spec, because the record leaves
one defensible answer. Each is folded into the design above.

- **1. The record's shape.** Frontmatter `id`, `slug` and `date`. The body
  starts with its date, written `<date> — <text>` by the verb and verbatim for
  a migrated entry. No heading. Grounds: the only form that keeps a migrated
  entry verbatim when separators vary (487 entries follow the date with an em
  dash, and 36 with a colon or a parenthesised tag and a colon). Writing new records the same way keeps the
  listing uniform.
- **2. One package owns the folder.** `internal/core/decisions` holds the path,
  the writer, the reader and `FrozenLog`. The verb and ideate both call its
  writer, and `dateFromStamp` moves to `recordid` for `decide` and it to share.
  Grounds: one-canonical-primitive.
- **3. Migrated ids are deterministic.** Each takes the adr-45 grammar with its
  stamp fixed to the entry's date at `000000`, and its four digits set to the
  entry's place, in file order, among the entries of that date (from `0001`).
  A rerun gives every entry the same id. An entry appended later, even one
  dated earlier, gets the next number for its date, and no earlier id moves.
  Record-lint's uniqueness rules check the ids like any others. Grounds: the
  landing protocol needs a rerun to reproduce the reviewed output. A random
  suffix would change every id each time. Decision 18 fixes the
  stamp's date: each entry's own date.
- **4. The slug comes from the text after the date.** Cut at 40 by
  `recordid.Slug`. The script refuses an entry with nothing slug-able, and
  none exists at 6f0589fc3. Grounds: the date is in the id already.
- **5. The gate goes in the migration change, not after it.** The brief for
  this spec asked for the gate's retirement after the migration. That order
  cannot land green. DA002 refuses any change that removes the log's lines, so
  step 6 fails while the gate runs. The gate's own contract
  (`decisionsappend.go`, under "REDACTION AND GRADUATION") says the change that
  graduates the log "also adjusts or retires this gate in the same diff". The
  alternative, retiring it before the migration, would disable it while the
  shape is not yet in force, which is what
  adr-2610110925465593 rules out ("retired by the shape"). So step 6 carries
  the deletion of the gate, its CI steps, its Makefile target and the
  conventions that name it.
- **6. What stays as history, and what the tree test polices.** The history
  set is the changelog, the review receipts, the raw research data, the guard
  corpus, and the records whose subject is the log. The tree test polices the
  full path on live surfaces, and links and dated citations everywhere outside
  history. Grounds: A3 asks that no file name the path as live, and a past
  narration is not live. The review receipts cannot be edited (RD002).
  Rewriting the ADRs about the log would change what they record.
- **7. A managed repository's frozen log is read, never written.** The lifeboat,
  the implement brief, the repository lint and the listing's closing line read
  it through the one constant `decisions.FrozenLog`. Nothing writes it. Grounds:
  the product thinker ruled it stays as frozen history, and those readers
  already served it. Dropping it would hide a team's history from the tools
  that showed it before.
- **8. The site page is opted into, uncapped, and separate.** The page is
  opted into like the issue ledger, because the folder is in the working tier
  (adr-32). It lists every entry, newest first, grouped by month. Decisions
  stay off the development page and the relationship chart, and the dashboard
  gets a tile. Grounds: A2 says every entry; the working-tier rule already
  exists; and 523 nodes would swamp pages built for stores that move.
- **9. The listing is full text and reads only the folder; decision 21 sets its order, newest first.** It
  ends with a line naming a frozen log where one exists. Grounds: the old file
  read oldest first, and the intent's mechanism says the listing fails if a
  reader has to open the files to answer what the file answered.
- **10. The writer refuses without a shared tier**, and creates
  `decisions/` inside one that exists. Grounds: adr-2609091248200336. The
  shared tier is abcd's namespace once `ahoy install` has laid it, and outside
  one there is nothing to commit the record to.
- **11. The ahoy gap is renamed** `skeleton.decisions_folder_missing`, and the
  old id goes with no alias (adr-40). The seed is a static README. Grounds:
  the gap's name states what is missing, and the old name states a file that
  is no longer seeded.
- **12. The decision-durability rule** passes on a committed folder, and warns
  on a frozen log alone, naming `ahoy install`. Grounds: the rule asks whether
  decisions survive a clone, and a repository that has not yet got the folder
  has somewhere to put them only after install.
- **13. Decision records are relinked like other records.** The append-only
  entries in `relink` and `record-slug-rename` go in step 6. Grounds: the file
  they protected no longer exists, and a decision record is under the same
  supersede-not-edit convention as every family a relink already rewrites.
- **14. The lifeboat keys both kinds.** A record is keyed by its id, and a
  frozen line by `dec-L<line>`. Grounds: a record id is digits only, so the
  two never collide, and the frozen file's existing keys stay stable for a
  re-plan.
- **15. How a citation is rewritten.** It becomes the handle, linked by
  relative path in Markdown where the original was a path or link, and bare in
  Go. A sentence that means the log as a place is rewritten to the folder by
  hand. Grounds: step 1 makes a decision handle a checked citation, so a
  wrong one is a finding rather than a dead link.

## Decisions (product thinker and technical facilitator, 2026-10-14)

Settled in the planning interview of 2026-10-14, which reconciled this spec
with the product thinker's later rulings. Each is folded into the design above.

- **16. The family's prefix is `dcn`.** The technical facilitator's ruling. A
  distinct prefix keeps a decision record apart from the lifeboat's
  `dec-L<line>` keys for a reader, not only for the grammar.
- **17. `abcd decide` records, lists and mints.** The facilitator's ruling:
  `abcd decide "<text>"` (and `/abcd:decide`) records a decision,
  `abcd decide` with no operand lists them, and `abcd decide --adr "<title>"`
  mints an ADR, which bare `abcd decide "<title>"` did before. Decisions
  outnumber ADRs about eight to one, so the common act takes the short form.
  The cost was accepted knowingly: an older page or the published binary of
  an earlier release still files an ADR for the bare form until both catch
  up, so the docs sweep lands in the same change. Settles open question 1 of
  2026-10-11.
- **18. Migrated entries take date order.** The product thinker's ruling. Each
  migrated id carries its entry's own date, the 48 entries written out of
  date order move to their dates, and within one date the file's order holds.
  The pull request lists the 48 that move. Settles open question 2.
- **19. The page is the "Decision log".** The product thinker's ruling: the
  page is labelled "Decision log", at `record/decision-log/`, and the ADR tile
  keeps "decisions". Settles open question 3.
- **20. The owner may run the script on their own repository.** The product
  thinker's ruling: abcd's install never touches a managed repository's
  existing log, and the person who owns that repository may convert it with
  the same one-off script, by hand. The script stays out of abcd's shipped
  code.
- **21. The listing and the page run newest first.** The product thinker's
  ruling of 2026-10-15, amending decision 9: the record can hold years of
  decisions, and the recent ones are the ones a reader comes for. The page's
  newest month comes first and opens, and the terminal listing prints the
  newest record first in both its forms. Every other reader keeps id order.

## Open Questions

None. The three questions of 2026-10-11 are settled by decisions 17, 18 and 19.
