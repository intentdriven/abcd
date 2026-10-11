---
id: spc-2610021456200324
slug: the-consistency-pass-runs-where-records
intent: itd-2609291923559186
origin: researcher-authored
production_mode: hand-written
---
# The consistency pass at commit time: a stable finding key, cross-document lint codes, chunked review, and a pre-commit gate

## Summary

This spec delivers
[itd-2609291923559186](../../intents/planned/itd-2609291923559186-the-consistency-pass-runs-where-records.md):
the five follow-ups itd-48 left to the consistency pass (A1, A2, A3, A4, G1),
planned together under ruling BT2 and tracked until now by
iss-2609260926323349.

Four pieces land in the order decision 7 and decision 6 set:

- **G1** gives every finding a deterministic key, `cfk-` and sixteen hex
  digits, derived from the class, the two ends and their collapsed quotes. The
  key is written into the issue a finding files (`finding_key:`), and a finding
  whose key a resolved or won't-fix record carries is marked `addressed` and
  files nothing.
- **A1** adds three record-lint rules at warn, read out of the `record_schema`
  scan as `stale_edge` and `edge_cycle` already are. They name a `builds_on` the
  target never mentions back, a `routed_from` the source never acknowledges, and
  a `related_issues` entry the issue does not link back.
- **A4** measures the assembled corpus against a request limit, reports an
  overflow with its size, and reviews an overflowing corpus in chunks chosen so
  that every pair of documents shares at least one chunk. One report covers every
  chunk.
- **A3 and A2** together are one verb, `abcd intent consistency precommit`, run
  as a gate in the core-owned managed pre-commit gates. It runs record-lint's kind
  rules and the A1 rules over the whole corpus, and runs the model pass over the
  staged records and their neighbours wherever the owner's own machine
  configuration routes the auditor to a provider. It then acts on what the commit
  introduced as the repository chose at setup: warn, stop or automatic. Where no
  route serves, the records are marked "deep check owed" on the status board and
  the commit proceeds. This piece waits for the managed gates (iss-84 with itd-62,
  ruling J23).

Everything below is grounded in the code as it stands:
`internal/core/intent/consistency.go` (emit and ingest),
`internal/core/capture/consistency.go` (the filer), `internal/core/lint`
(`record_schema`, `edges.go`, `config.go`), `internal/core/ahoy` (the value
questions and their gaps), `internal/core/oracle` (routes, `Admitted`, ruling
DR5 in `selfcontained.go`) and `internal/surface/cli/board.go`.

## Scope

In:

- a new leaf package, `internal/core/record/findingkey`, which derives the key;
- `internal/core/intent`: the key on every finding and row, the `addressed`
  state, the `staged` scope, overflow measurement and chunked emit and ingest;
- `internal/core/capture`: the key lookup across `open/`, `resolved/` and
  `wontfix/`, `finding_key` on the filed record, a dry classification for the
  gate, and `Link` taught to append `related_intents`;
- `internal/core/issueschema`: `finding_key` joins `Known`;
- `internal/core/lint`: the three A1 rules, `finding_key`'s shape check in
  `record_schema`, and `related_issues` and `routed_from` read by the scan;
- `.abcd/record-lint.json`: the three rules at warn;
- a new package, `internal/core/consistencygate`: the gate's decisions;
- `internal/core/ahoy`: the setup question and its gap;
- `internal/surface/cli`: the verb, the multi-payload ingest, and the board line;
- the plugin pages `commands/intent.md` and `commands/ahoy.md`;
- the regenerated surface snapshot and command reference.

Out, as the intent draws it: Role 1 and Role 3 beyond its scheduling; any change
to the classes or the rubric (`consistencyClassText` and
`consistencyRubricRules` are not edited: a chunk's manifest is that chunk's,
so the existing rule that each end is a manifest document already bounds a
chunked finding); and the managed pre-commit gates themselves, which this spec plugs
into and does not build.

## Approach

### G1: one key, derived once

`findingkey.Derive(class string, a, b End) string`, where `End` is
`{Path, Quote string}`, is the only derivation. Its steps:

1. Each end's path becomes its **identity path**. Under a record store whose
   folder is the record's status (the intent buckets, the issue status
   folders, `specs/open` and `specs/closed`), the status segment is dropped:
   `.abcd/development/intents/planned/itd-N-slug.md` becomes
   `.abcd/development/intents/itd-N-slug.md`. Any other path, such as a brief
   page, is kept as it is.
2. Each quote is collapsed with `strings.Fields`, the same `collapseSpace` the
   ingest already applies.
3. The two ends are ordered by identity path and then by quote, which is the
   order `validateConsistencyFindings` already sorts them in.
4. The result is `"cfk-"` followed by the first sixteen hex digits of the
   SHA-256 of `abcd/finding-key/v1`, the class, and the two ordered
   (path, quote) pairs, with NUL between every field.

The package sits under `internal/core/record` beside `match` because both
`intent` and `lint` call it, and `lint` cannot import `intent`.

`validateConsistencyFindings` replaces its local
`f.Class + "\x00" + k0 + "\x00" + k1` with the derived key, so the duplicate
check and the row key are one computation. `ConsistencyFinding` gains
`Key string` (`json:"key"`).

**`ConsistencyRow`** gains `State` (`filed` | `linked` | `addressed`) and
`AddressedBy` (the record's id and its folder). `Linked` stays, so `--json`
consumers keep working. `ConsistencyIngestResult` gains
`Addressed []string`. `ConsistencyFiling` gains `Addressed bool`, and its
`IssueID` names the record either way.

**The rendered report** gains a key column. The ledger cell reads
`iss-N (filed)`, `iss-N (already open)` or `iss-N (addressed: resolved)` /
`(addressed: wontfix)`. Each finding's detail line carries `key: cfk-…`.

**The filer** in `capture.IngestConsistency` looks a finding up in this order:

1. Every record in `open/`, `resolved/` and `wontfix/` whose `finding_key`
   equals the key. The ledger is read once per ingest, as `loadOpen` reads it
   now. An open record wins and links. Failing that, a resolved or won't-fix
   record makes the finding `addressed`: nothing is filed and nothing is
   written to that record.
2. Failing that, `openRecordHolding`, the existing prose match. It still links
   a record filed before keys existed. Linking writes nothing, as it does
   today, so no key is back-filled into a legacy record.
3. Failing that, the finding is filed. `CaptureRequest` gains `FindingKey`,
   which capture writes as frontmatter `finding_key:` and refuses unless it
   matches `^cfk-[0-9a-f]{16}$`.

`check` skips a finding that is linked or addressed, as it already skips a
linked one. The filing-time match of itd-2609212137116617, which
adr-2609300821558671 extended, keeps working as it does. The key does not
replace it: the key joins exactly one contradiction to itself, and the match
compares prose.

**`issueschema.Known`** gains `finding_key`. Without it the ledger reader would
refuse a keyed record and drop it from every capture surface. `record_schema`
reports a `finding_key` whose value is not the key's shape, so the
committed-ledger gate refuses what capture refuses.

### A1: three cross-document rules at warn

The rules are declared as `stale_edge` and `edge_cycle` are: constants in a new
`internal/core/lint/reciprocal.go`, a `knownRules` entry each in `config.go`,
and one call each in `lint.go` beside the edge rules, because they read the same
store scan, which straddles `cfg.Roots`. They are armed in
`.abcd/record-lint.json` with `"severity": "warn"`, so a tree that passes
record-lint today keeps passing (decision 8). The scan already parses
`recordParsedFields`. `related_issues` and `routed_from` join the fields parsed
for the graph and are never resolved by `record_schema`, which keeps its own
findings unchanged.

- **`builds_on_reciprocal`**: a record's `builds_on` names a record in the
  corpus whose file, frontmatter or body, never names the source by its id as a
  whole token (the token test `capture.namesID` applies). The finding names
  both files and the rule. A target that does not resolve is `record_schema`'s
  finding and is skipped here.
- **`routed_from_acknowledged`**: a `routed_from` entry is `<record>:<item>`.
  The rule judges an entry only when its record half resolves in this corpus
  and that record carries the item label as a whole token, which proves it is
  the record that did the routing. It then reports that record when it never
  names the receiving record back.
- **`related_issues_backlink`**: an intent's `related_issues` names an issue
  whose `related_intents` does not list that intent. The finding names both
  files.

Every A1 finding also carries its key: `findingkey.Derive` with the rule id as
the class, the two files as the ends and empty quotes. The gate needs the key
to name a finding and to tell a new finding from one already standing. The
record-lint text renders it at the end of the message, so the line reads the
same with or without the gate.

**Measured on the tree this spec was written against:**

| Rule | Edges | Findings |
|---|---|---|
| `related_issues_backlink` | 46 | 0 |
| `routed_from_acknowledged` | 5 | 0 |
| `builds_on_reciprocal` | 184 | 136 |

The `routed_from` entries are all spellings from the predecessor store, which
the local spc-33 does not carry, so all five are skipped. The 136 targets of
`builds_on_reciprocal` never name their follower. They arrive as warnings and
fail nothing (see the open point).

### A4: measure, report, and read in chunks

**The limit** is `consistency.max_request_bytes`, read through `layered.Config`
(the repository file, then `~/.abcd.noindex/config.json`, then a bundled 524288). The
intent package claims the `consistency` namespace and its keys
(`max_request_bytes`, and `on_finding` below), so a misspelt key is refused
naming its file. A value below 65536 is refused, and so is a value above
1048576. The upper bound is `maxAgentPromptBytes`, the guarded-read cap
`sendRequest` already puts on every input file it sends to a provider. A corpus
file above that cap cannot be dispatched at all.

**The measure** is the byte length of `consistencyCorpusText` for the scope: the
exact file the reviewer reads, so there is no tokenizer and no estimate. Today
the brief alone is about 1.25 MB. The full corpus is already over the 1 MiB cap,
so a provider-routed corpus run refuses before it sends anything, and A4 is live
rather than dormant.

**When the corpus fits**, the emit is exactly what it is today: one request, one
corpus file, and `overflow` omitted from the result and the report
(criterion 8, second half).

**When it overflows**, the emit groups the documents in path order, filling each
group up to half the limit with each document's framing counted. It then writes
one chunk for every pair of groups. Every pair of documents therefore shares at
least one chunk, and a contradiction between two documents is not lost at a
chunk boundary. A scoped run (`itd-N`, or the gate's `staged` scope) pins the
scope's documents into every chunk instead and pairs them with each group of the
rest, the groups sized to what the scope leaves free. A single document larger
than the space a chunk leaves is reviewed in no chunk. The emit result and the
report name it as unreviewed, with its size.

Each chunk is an ordinary request and corpus pair:

- its receipt is derived from the scope, the whole corpus's digest and
  `chunk i of n`;
- its prompt names the chunk and the emit's receipt (the receipt of the
  unchunked corpus), so `prompt_hash` binds both;
- its manifest lists only that chunk's documents, so the existing rubric rule
  ("each end is a document in the corpus manifest") confines a finding to the
  chunk with no change to the rubric.

The emit result gains `overflow: {corpus_bytes, limit_bytes, groups, chunks,
unreviewed}` and lists every chunk's request and corpus path.

**The ingest** takes `--findings-json` once per chunk and refuses, writing
nothing, unless it holds exactly one valid payload for every chunk of one emit.
That refusal is what makes "the report covers every chunk" checkable. Findings
from different chunks that share a key are one finding, and the first chunk's
copy is kept. The report's header states the overflow (corpus bytes, limit,
groups, chunks, unreviewed documents) and lists every chunk's receipt.

**On a provider route**, the front door sends the chunks one after another
through `sendRequest` and ingests them together. On the harness, the request
block lists every chunk for the host to run.

### A3 and A2: one gate, `abcd intent consistency precommit`

The gate is a verb, so the managed pre-commit gates register and run it as they
run any gate. This spec fixes the verb's behaviour and its exit codes. The
registration, the hook file and the ordering among gates belong to the managed
gates (decision 6), which is why this step is blocked on them.

The decisions live in `internal/core/consistencygate`, which imports `intent`,
`capture`, `lint` and `oracle` and is imported by none of them. It never
prints. The front door in `internal/surface/cli` performs the dispatch, as
`newIntentConsistencyCommand` does today.

**1. Setting.** `consistency.on_finding` is read from the repository layer only.
A machine-layer value is refused naming the file, because the choice is the
repository's (criterion 4). An absent value is `warn`.

**2. Staged set.** The gate reads `git diff --cached --name-only -z`. It judges
the working tree, as record-lint does. The model pass's report is therefore
always marked `dirty` against HEAD, which is true of a commit not yet made.

**3. Kind rules and A1 over the whole corpus** (criteria 5, 6 and 7). The gate
calls `lint.Lint` with the repository's record-lint configuration, narrowed to:

- `intent_lifecycle`, which carries the kind-and-shelf rule of itd-34;
- `record_schema`, which carries the bundle leg of itd-34;
- the three A1 rules.

Each finding keeps record-lint's own severity in the output.

**4. Introduced.** An A1 finding belongs to this commit when one of its ends is
staged and its key is absent from the same rules run over HEAD. For that run,
the gate puts HEAD's record stores into a scratch directory with `git archive`.
The directory sits under the local tier, is removed afterwards, and no live
worktree is touched. A finding already standing at HEAD is reported as standing
and never stops or fixes anything. Without HEAD (a first commit), every finding
is the commit's.

**5. Route.** The model pass runs only where
`oracle.Resolve("intent-auditor", …)` gives a provider leg (`OnProvider`) whose
target comes from `~/.abcd.noindex/config.json` and which `APIConfig.Admitted` accepts.
Each of the following is "no route", and the reason is named:

- a harness route;
- a repository route;
- a route `Admitted` refuses, DR5's refusal among them;
- a provider `FellBack` reports unreachable.

No model is called for any of them (criterion 5).

**6. Model-pass scope** (criterion 6). The seeds are the staged documents the
pass's corpus holds: brief pages, and intents outside `superseded/`. The
neighbours are the corpus documents one edge away from a seed in either
direction in `lint.LoadRecordGraph`. The emit gains a scope, `staged`, spelled
`staged-<12 hex of the seed set's digest>` so that `requestScopeRe`, the report
directory name and the receipt all accept it. Its corpus is the seeds plus the
neighbours, and every finding must have an end in a seed, as an `itd-N` scope
requires today. A commit that stages no corpus document runs no model pass and
owes none. A4 chunks this scope like any other.

**7. Mode.** The gate acts only on what the commit introduced. An A1 finding
counts when it is introduced as in step 4. A model finding counts when its row
would be `filed`, meaning no record holds its key and no open record holds its
prose. A linked or addressed row is printed as known.

- **warn**: every finding is printed with its rule or class, both ends and its
  key, and the gate exits 0. The model pass's ingest runs dry: it validates and
  classifies the rows and writes neither an issue nor a report (criterion 2).
- **stop**: the same report. The gate exits 1, refusing the commit, when
  anything the commit introduced is present, naming each finding by its key.
  The dry ingest writes nothing (criterion 2).
- **automatic** (criterion 3):
  - a `related_issues_backlink` finding is fixed through `capture.Link`, which
    gains `RelatedIntents`, appended through the validated single-field rewrite
    it already makes for `blocked_by`. The issue is then staged;
  - every judgement finding is ingested for real, so each issue is filed with
    its `finding_key` and the report is written to the reviews shelf. Both are
    staged;
  - each change is noted, both printed and as a `Consistency-fix:` trailer
    line handed to the managed gates' `prepare-commit-msg` stage, so the commit
    message records what the gate added;
  - an A1 finding with no field to write the back-link into (`builds_on`,
    `routed_from`) is printed as under warn;
  - the gate exits 0.

**8. Owed.** When the staged set holds corpus documents and no route serves,
the gate writes `.abcd/.work.local/consistency/owed.json`: each document's
path, the commit it was owed at, and the reason. The status board's
`boardOutput` gains `consistency_owed`, rendered as one line ("deep check owed:
N records — <paths>; run `abcd intent consistency`"). The line is omitted when
nothing is owed, and it lives on the board only, not in `statusblock.Block`.
An ingest whose corpus included an owed path clears that entry. The gate exits
0 on this branch, so a judgement it could not obtain never refuses a commit
(decision 3, criterion 5).

**Output.** Every source reports in one run, in one envelope (criterion 7):

- `lint` holds the kind-rule and A1 findings, each with its key and
  `introduced` or `standing`;
- `pass` holds `ran` or `owed`, with the reason, the report path and the rows;
- `mode` and `action` say what the gate did.

`--json` emits the same envelope. A fault, such as a tree the scan cannot read
or a configuration refusal, exits 2 and leaves the managed gates' fail-closed
contract to decide. Exit 1 is only the stop refusal.

**Setup.** `ahoy` gains the value question `consistency_hook`, with:

- a `promptHelp` entry giving the three answers in plain words;
- a choices set `warn | stop | automatic`;
- the flag `--consistency-hook`;
- the gap `config.consistency_hook_missing`, raised in a managed repository
  whose `.abcd/config.json` lacks `consistency.on_finding`.

`apply` writes the answer there (criterion 4). It ships with this step, so no
repository is asked about a gate it cannot yet run.

### The surfaces

`commands/intent.md` documents:

- the key and `addressed` (step 1);
- the overflow and the chunked ingest (step 3);
- the `precommit` verb, its modes and the owed line (step 4).

`commands/ahoy.md` documents the new question. The surface snapshot and the
generated command reference are regenerated in each step that adds a flag or a
verb.

## How each acceptance criterion is met

1. `builds_on_reciprocal` and `related_issues_backlink` read both records from
   the `record_schema` scan and name the rule and both files. No model runs.
2. Under stop, an introduced finding exits 1 naming its key. Under warn, it is
   printed by its key and the gate exits 0. The dry ingest writes nothing in
   either case.
3. Under automatic, the gate adds the missing `related_intents` entry through
   `capture.Link` and stages it with a printed note and a trailer. It files the
   judgement finding with its `finding_key`, stages both, and exits 0.
4. The `consistency_hook` question is asked by `ahoy` (gap, prompt help, flag)
   and the answer is recorded as `consistency.on_finding`.
5. With no admitted machine-layer provider route, no model is called, the A1
   and kind rules run, the gate exits 0, and `owed.json` feeds the board's
   "deep check owed" line naming the records.
6. With a route, the `staged` scope's corpus is the staged corpus documents plus
   their one-hop graph neighbours, and the lint half runs over the whole
   corpus.
7. Kind rules, A1 and the pass report in one invocation and one envelope.
8. The emit measures `consistencyCorpusText` against the limit. On overflow it
   reports the size and the chunk plan, and the ingest refuses unless every
   chunk is present. A corpus that fits emits one request with no overflow.
9. `findingkey.Derive` returns the same key from the same class, identity paths
   and collapsed quotes on any run. Capture writes it as `finding_key`, and the
   second run's row carries it.
10. The key lookup finds a resolved or won't-fix record carrying the key. The row
    is `addressed`, naming that record, and nothing is filed.

## Settled here, not by the intent

These are the facilitator's design calls. None changes a criterion.

- **"Both end paths"** is read as each end's path with its status folder
  dropped. This follows the spirit of the letter rather than the letter: with
  the folder in the key, a contradiction would change key when its intent moved
  from `drafts/` to `planned/` or its issue to `resolved/`. That would break the
  intent's promise that the same contradiction carries the same key on every
  run.
- **Key format and home**: `cfk-` and sixteen hex digits, in
  `internal/core/record/findingkey`. A1 findings are keyed by the same function,
  with the rule id as the class and empty quotes.
- **Key lookup precedence**: an open record with the key links. A resolved or
  won't-fix record makes the finding addressed. The prose match still links
  legacy records. Linking never back-fills a key.
- **`builds_on` reciprocity** means the target names the source anywhere in its
  file.
- **`routed_from` reciprocity** is judged only when the source resolves here
  and carries the item label. The predecessor store's entries are therefore not
  misjudged against this store's spc-33.
- **`related_issues` reciprocity** means the issue's `related_intents` lists
  the intent.
- **The request limit** is a byte measure of the assembled corpus file, set by
  `consistency.max_request_bytes` (default 512 KiB, bounded by the existing
  1 MiB send cap).
- **Chunking** covers every pair of groups, so no pair of documents goes unread
  together. A scoped run pins its scope into every chunk. A document too large
  for any chunk is named as unreviewed. The ingest demands every chunk.
- **The gate's model-pass corpus** is the staged brief pages and intents plus
  one hop over the record graph. A staged issue, spec or ADR seeds nothing,
  because the pass's corpus and judgement are itd-48's.
- **"Introduced"** means:
  - for an A1 finding, an end is staged and its key is absent at HEAD;
  - for a model finding, its row would be filed.

  Standing findings never stop or fix anything.
- **A model route** is a provider leg from the machine layer that `Admitted`
  accepts. A harness route, a repository route, a refused route and an
  unreachable provider all mean owed, with the reason named.
- **An unanswered repository runs as warn**, as decision 8 and the additive
  impact require. `on_finding` is read from the repository layer only.
- **Under automatic**:
  - only a back-link with a field to hold it is written, through `capture.Link`;
  - other A1 findings are printed;
  - the filed issue and the report join the same commit;
  - the note is printed and handed on as a `Consistency-fix:` trailer.
- **Warn and stop run the ingest dry**, so neither writes an issue or a report.
- **"Kind rules"** are `intent_lifecycle` and `record_schema`'s bundle leg. The
  setting, not record-lint's severity, decides whether the gate refuses.
- **The owed list** lives in the local tier and clears when an ingest's corpus
  includes the path. Its board line is on `boardOutput` only.
- **The verb** is `abcd intent consistency precommit`, with exit 0 to proceed,
  1 only for the stop refusal, and 2 for a fault.

## Open point

_None open._ Both points were settled on 2026-10-02:

- **Ruling DR5 and the gate's model pass** needed no question, because the record settles it. DR5 keeps file-reading agents off a provider with a key unless the owner's personal list admits it, and DR5b-4 treats every remote provider the same way. So the gate's model pass runs where the owner's own settings allow it: on a local model, or on a provider the owner lists once that list carries bundles. Everywhere else the commit is marked "deep check owed" with DR5 named as the reason.
- **The 136 `builds_on` arrivals** were ruled by the technical facilitator: only new ones are flagged. The one-way edges that exist when the rule lands are recorded once as a baseline, and the rule flags only an edge added after it. The baseline is the shared, shrink-only warnings baseline that ruling BT3 plans (H5); until that ships, the rule's own exemption list holds the 136, and it never grows.

## Footprint

- packages: internal/core/record/findingkey, internal/core/intent, internal/core/capture, internal/core/issueschema, internal/core/lint, internal/core/consistencygate, internal/core/ahoy, internal/surface/cli, .abcd/record-lint.json, commands/
- tests: the key over reordered ends, whitespace-variant quotes and a record moved between status folders; the same finding ingested twice carrying one key and `finding_key` on the filed record; a resolved and a won't-fix keyed record making the row addressed with nothing filed; a legacy open record still linked by the prose match; `finding_key` admitted by the ledger reader and a malformed one refused by capture and `record_schema`; the three A1 rules over a one-way `builds_on`, an unacknowledged `routed_from`, a predecessor-store `routed_from` skipped, and a missing `related_intents` back-link, each at warn naming both files and its key; a corpus over the limit reported with its size and chunked so every pair of documents shares a chunk, an ingest missing a chunk refused, and a corpus under the limit emitted once with no overflow; the gate under warn, stop and automatic with an introduced back-link and an introduced judgement finding; a standing finding never stopping a commit; no route, a harness route, a repository route and a DR5-refused route each giving owed with no model call and exit 0; the staged scope's corpus as seeds plus one-hop neighbours; one envelope carrying lint and pass together; the ahoy question, its gap and its flag; the board's owed line and its absence from the site block

## Steps

1. G1: the stable finding key and the addressed state
   - criteria: 9, 10
   - packages: internal/core/record/findingkey, internal/core/intent, internal/core/capture, internal/core/issueschema, internal/core/lint, internal/surface/cli, commands/
   - tests: `Derive` is stable over reordered ends, collapsed whitespace and a status-folder move, and differs by class; the duplicate check in `validateConsistencyFindings` uses the key; the same payload content ingested in two runs gives one key and the filed record carries `finding_key`; keyed resolved and won't-fix records make the row `addressed` naming them, with nothing filed and the record untouched; an open keyed record links; a legacy open record links by prose with nothing written; `issueschema.Known` admits `finding_key` and capture and `record_schema` refuse a malformed value; the report renders the key column and the addressed cell
2. A1: the cross-document lint codes
   - criteria: 1
   - packages: internal/core/lint, .abcd/record-lint.json
   - tests: `builds_on_reciprocal`, `routed_from_acknowledged` and `related_issues_backlink` each fire on a fixture pair naming the rule and both files with the key, and stay silent on a reciprocated pair; an unresolved target is left to `record_schema`; a `routed_from` whose source lacks the item label is skipped; the rules are known to `validateRuleNames` and land at warn, so record-lint over this tree exits as before
3. A4: overflow measurement and chunked review
   - criteria: 8
   - packages: internal/core/intent, internal/surface/cli, commands/
   - tests: a corpus under the limit emits one request with no `overflow`; over the limit, the result and the report state corpus bytes, limit, groups and chunks, and every pair of documents shares a chunk; a scoped run pins its scope into every chunk; an oversize document is named as unreviewed; the ingest refuses a chunk set missing one chunk or mixing two emits, writing nothing; cross-chunk findings sharing a key are one row; `consistency.max_request_bytes` is read through the layered resolver and values outside 64 KiB–1 MiB are refused
4. A3 and A2: the pre-commit gate and its setup question
   - blocked: waits for the core-owned managed pre-commit gates (iss-84 with itd-62, ruling J23); steps 1 to 3 do not
   - criteria: 2, 3, 4, 5, 6, 7
   - packages: internal/core/consistencygate, internal/core/intent, internal/core/capture, internal/core/lint, internal/core/ahoy, internal/surface/cli, commands/
   - tests: stop refuses an introduced finding by its key and warn prints it and proceeds, neither writing an issue or a report; automatic adds the `related_intents` back-link through `capture.Link`, files the judgement finding with its key, stages both with the note, and proceeds; a finding standing at HEAD never stops; no route, a harness route, a repository route, a DR5-refused route and an unreachable provider each call no model, exit 0 and write the owed list, which the board renders and a later ingest clears; with a local route the staged scope's corpus is seeds plus one-hop neighbours and the lint half covers the whole corpus; one envelope carries kind rules, A1 and the pass; `ahoy` raises `config.consistency_hook_missing`, explains the three answers, honours `--consistency-hook` and writes `consistency.on_finding`; a machine-layer `on_finding` is refused; the surface snapshot and command reference regenerated
