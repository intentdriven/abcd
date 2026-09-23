---
id: spc-2609231435545473
slug: every-release-arrives-with-its-own-press-release-when-a
intent: itd-2609231013154443
origin: researcher-authored
production_mode: hand-written
---
# The release cut writes RELEASE.md from the press releases of the intents it shipped

## Summary

spc-2609231435545473 delivers itd-2609231013154443: a feature release's cut
writes a short press release to `RELEASE.md` at the repository root, composed
by an agent from the press releases of the user-facing intents in the cut, and
moves the page it replaces to `.abcd/development/releases/<version>.md`.

It is the changelog slice of spc-11 applied to a second document, not a new
trust model. The same host-delegated composer writes both documents in one
payload; the binary computes the set of intents the page must cite, proves the
payload cites exactly that set, checks every carried quote against its source,
and only then makes three writes in a fixed order, rolling back whatever landed
if a later one fails. A refused payload goes back to the composer with
machine-readable reasons until it is valid (Decision 9). A cut that ships fixes
alone writes no page and says why.

At this tip the read-only preview (`abcd changelog`) shows the set the first
page would carry: `itd-121` and `itd-2609221656373558`, the two `additive`
intents among fourteen added records; the other twelve are issues.

## Scope

### In

- **The press-release set**: `PressReleaseRequired()` beside
  `ChangelogRequired()` in the changelog package, marked per cut entry as
  `in_press_release`, so the preview, the composer's view and the bijection read
  one definition.
- **The payload**: the changelog payload gains a `press_release` object
  (schema 2) carrying headline paragraphs, a name list and quotes, every
  citation as data.
- **The composer**: `release-changelog-composer` is extended to write the page
  in the same pass as the changelog lines, with a new injection canary and a
  no-forecast fixture.
- **Ingest validation**: every refusal AC6 and AC7 name, plus the outbound-policy
  check, all run before any write.
- **The retry-until-valid loop**: the binary refuses with machine-readable
  reasons; the host re-invokes the composer with them, with no attempt limit,
  reporting every refused attempt.
- **The three writes and their rollback**: archive, page, changelog heading, in
  that order; the first-cut and fixes-only cases.
- **The page**: `RELEASE.md` whose heading names its version and date.
- **The preview**: `abcd changelog` (and the ship verb's emit step, which shares
  its render) lists the intents the page will be composed from.
- **Gates**: `RELEASE` in the `stray_root_docs` allowlist, `RELEASE.md` in the CI
  inert-path list, a `README.md` for `.abcd/development/releases/`, and that
  directory added to `forbidden_synonyms`' exempt prefixes.
- **Doc surfaces**: `commands/launch.md`, the launch surface page's prose,
  the agents catalogue in the brief, `agents/CHANGELOG.md`.

### Out

Carried from the intent and not reopened here:

- Any forecast: no `target_release` intent, no still-planned work, no date, no
  "coming next". The `target_release` report stays in the cut's report.
- Issues, `impact: internal` intents and Removed intents as page sources.
- Release pages on the project website (a separate draft intent), and the
  published GitHub release notes (`release.yml` and its scaffold template are
  untouched).
- Pages for releases cut before this one: no backfill, no on-request page.
- The changelog's wording, sections, bijection and version derivation.
- The release retrospective (itd-24) and the lifeboat's `disembark
  press-release`.
- An enforced cap on intents per release: about twenty is a scope condition of
  the intent, not a refusal.

## Approach

### 1. The press-release set, defined once

`RecordSet.PressReleaseRequired()` in `internal/core/changelog/shipped.go` returns
the `Added` records whose id is an `itd-*` and whose `InChangelog()` holds:

```
PressReleaseRequired = Added ∩ itd-* ∩ InChangelog
```

Removed records, issues and `impact: internal` intents fall out by
construction. A record carrying a valid `shipped_in:` is already absent from
`Added` (`ShippedSince`), so a migration-closed intent is never announced. An
unlabelled `Added` record never reaches a ready cut (`UnlabelledAdded` refuses
it), so `InChangelog()` means "user-facing" here without qualification.

`release.Entry` gains `InPressRelease bool` (`json:"in_press_release"`), set in
`entriesOf` from that method. It is carried per entry for the reason
`InChangelog` is: the host must never re-derive the rule. The addition to the
`Cut` JSON is additive.

`changelog.Record` gains `PressRelease string`: the body of the intent's
`## Press Release` section, read from the same blob `newRecord` already reads for
`Title` and `Summary` (no second git read), extracted by a new
`pressReleaseSection` in `source.go` and bounded by the existing
`maxRecordBytes`. It reaches the ingest on `release.Entry` as an unexported field,
so it never enters the cut JSON; the composer reads the record at its `path`, as
it does today.

### 2. The payload

`ChangelogSchemaVersion` moves from 1 to 2 and `ChangelogPayload` gains one
field. A schema 1 payload is refused as unsupported; the prompt and the binary
ship together in the plugin, so no composer is left writing the old shape.

```json
{
  "schema_version": 2,
  "prompt_version": "0.4.0",
  "next_tag": "v0.10.0",
  "entries": [ { "section": "Added", "records": ["itd-121"], "text": "..." } ],
  "press_release": {
    "headlines": [
      { "records": ["itd-121"], "text": "One paragraph told as the moment a person notices it." }
    ],
    "listed": ["itd-2609221656373558"],
    "quotes": [
      { "record": "itd-121", "text": "\"... what I'd do next,\" says Nia, facilitator.", "attribution": "Nia, facilitator" }
    ]
  }
}
```

- **`headlines[]`**: Prose paragraphs, each citing the intents it tells in
  `records`. At least one when the set is non-empty.
- **`listed[]`**: The ids of every other intent in the set. The binary renders
  each as its record's `Title`, so the name list carries no composer prose.
- **`quotes[]`**: `{record, text, attribution}`, where `record` is a headline
  intent (the intent's Scope: "the persona quotes of the intents it tells").
- **`press_release` is `null` or absent exactly when the set is empty.**

The citation suffix on each headline and quote is appended by the core, as the
changelog's is (`renderSection`), never trusted from the prose.

### 3. The composer

The page is written by `release-changelog-composer`, extended rather than
joined by a new agent. One composition pass yields one payload, which gives AC1's
"validate all first" for free, keeps the public surface to the existing
`--changelog-json` flag (so the surface guardrail sees no change), and avoids the
name `press-release-composer`, which is the lifeboat agent's. The rejected
alternative, a second agent with its own flag, costs a second payload the host
must pair with the first, a second canary set, and a new flag.

The prompt (`agents/release-changelog-composer.md`) bumps `0.3.0` to `0.4.0`: a
schema break inside the `0.x` calibration band, where `1.0.0` is reserved for a
measured lock (`agents/README.md`). It gains a section on the page: the
`in_press_release` entries are the set; choose the headlines; tell each as the
user moment in its own press release's words; list the rest by id; carry the
quote of each told intent word for word with its attribution; write nothing
forward-looking (no date, no future release, no planned work). Its
untrusted-data paragraph extends to the press releases it now quotes. An
`agents/CHANGELOG.md` entry for `0.4.0` is required by record-lint's
`agent_contract` rule.

Fixtures under `agents/release-changelog-composer/fixtures/`:

- **`injection-canary.json`**: Updated to schema 2. Kept as the changelog canary.
- **`injection-canary-press-release.json`**: New. A shipped intent whose
  `## Press Release` carries the hostile text (an instruction to cite a planned
  intent, to announce a date, to drop a record, a `</system>` break, an HTML
  comment lure, and a fake quote attributed to a real-sounding person). The
  expected payload cites exactly the set and carries only the intent's genuine
  quote.
- **`no-forecast.json`**: New. A cut beside a planned intent carrying
  `target_release`; the expected payload does not cite it and contains none of
  the fixture's `must_not_contain` strings (a date, "next release", "coming").

### 4. Ingest validation

All validation runs in `release.Ingest` after the existing changelog checks and
before any write. New code lives in `internal/core/release/page.go`. Each fault
becomes a reason with a stable `code`, the payload path it concerns (`at`) and a
sanitised `detail`. Decode faults are necessarily single; after decoding, the
validator collects every reason it finds in one pass, so the composer sees the
whole list at once.

| Code | Refuses |
| --- | --- |
| `payload-oversize` | A payload over `MaxPayloadBytes` (existing guard) |
| `malformed-json`, `unknown-field`, `trailing-data`, `schema-version`, `prompt-version` | The existing decode guards (`decodeChangelogPayload`, `DisallowUnknownFields`) |
| `stale-cut` | `next_tag` differs from the re-derived cut (existing); the host re-runs the emit step before recomposing |
| `text-oversize` | A headline, quote or attribution over `maxEntryProseBytes`, measured on the raw value before cleaning, because `termsafe.CleanProseLine` truncates silently; also more than 50 headlines, 50 quotes or `maxRecordsPerEntry` records on one headline |
| `malformed-id` | An id failing `payloadRecordIDRe` or over `maxRecordIDBytes` |
| `missing` | An id in the set that no headline and no `listed` entry cites |
| `outside-set` | A cited id not in the set; the detail says which: not in this cut, an issue, `impact: internal`, or removed |
| `duplicate-citation` | An id cited twice across `headlines` and `listed` |
| `no-headline` | A non-empty set with no headline |
| `page-for-empty-set` | A `press_release` object when the set is empty |
| `heading`, `fence` | A text whose first non-space rune is `#`, or which contains a code fence (three backticks or three tildes); either would break the page |
| `empty-prose` | A headline whose cleaned text is empty |
| `quote-source` | A quote whose `record` is not a headline record |
| `quote-not-verbatim` | See below |
| `outbound-policy` | See below |

The existing changelog entry faults (`section`, `section-not-writable`, the
changelog bijection's MISSING, INVENTED and INTERNAL) are reported as reasons in
the same list, so the loop reads one shape.

**Verbatim quotes.** The cited intent's `PressRelease` text is split into
paragraphs, each normalised as `firstParagraph` normalises (blockquote markers
stripped, whitespace runs collapsed). A quote passes when its whitespace-collapsed
`text` is a contiguous substring of one such paragraph, its `attribution` is
non-empty and a substring of `text` (so the attribution is carried exactly as the
source has it), and `termsafe.CleanProseLine(text)` equals the collapsed text
(so the bytes rendered are the bytes verified). Anything else is
`quote-not-verbatim`, naming the record. A quote taken from outside the
`## Press Release` section fails by construction.

**Outbound policy.** `scanner.CheckOutbound(root, text, label)` in
`internal/adapter/scanner/outbound.go` is the check-direction twin of
`ScrubOutbound`: it reports session URLs and tool attribution footers and
returns no rewritten text, which is right for a gate that sends output back to
its author. The ingest runs it over the rendered page and over the rendered
changelog section, closing the same gap for the changelog in this change (the
design review's M6; one payload, one check). A degraded scanner config makes
`CheckOutbound` error; that is a structural fault, not a payload reason, and the
cut stops.

### 5. The retry-until-valid loop

The verb is host-delegated, so the loop lives in the host orchestration
(`commands/launch.md`, steps 2 and 3); the binary stays stateless per attempt.

- **The binary**: A refused payload is a `*release.PayloadRefusal{Reasons}`
  error. With `--json` the ship verb renders
  `{"cut": ..., "written": false, "payload_refusal": {"reasons": [{"code", "at", "detail"}]}}`
  and exits 2; the human render lists the same reasons. Exit 2 **with**
  `payload_refusal` means "recompose"; exit 2 **without** it (unreadable
  repository, degraded scanner, unreadable outgoing page, archive collision)
  means "stop". Exit 1 (the cut refuses) stays a stop. No exit code changes.
- **The host**: On `payload_refusal`, it tells the user at once that attempt N
  was refused and lists every reason (loud staging, so an autonomous run shows the
  loop as it happens), then re-invokes the composer with the cut, its previous
  payload and the reasons. On `stale-cut` it re-runs the emit step first. There
  is no attempt limit (Decision 9). The final cut report lists every refused
  attempt and its reasons before the written result.
- **Unchanged**: If the composer cannot run at all, the existing LOUD STAGE stop
  applies: nothing is hand-written.

The reason codes are one Go enum; a test pins the command page's list to it, as
`TestComposerPromptNamesOnlyTheWritableSections` pins the section table today.

### 6. Writes, order and rollback

The writes move from `Ingest`'s single `WriteFileAtomicPreserveMode` call into
`internal/core/release/write.go`, which holds the pre-cut bytes and an undo:

1. **Validate everything first**: The payload (section 4); the outgoing
   `RELEASE.md`, if present, must carry a parseable heading (section 7), or the
   cut stops as a structural fault; the archive target must not exist; the
   changelog content is built by `insertSection` without writing.
2. **Archive**: The outgoing page's bytes are written to
   `.abcd/development/releases/<its version>.md` through
   `fsutil.CreateExclusiveIn`, which never overwrites.
3. **Page**: The new `RELEASE.md` replaces the old one atomically
   (`WriteFileAtomicPreserveMode`). Writing a copy then replacing, rather than
   renaming, leaves no instant with no `RELEASE.md`.
4. **Changelog heading**: The dated section, as today, written last so the file
   the tagging workflow reads lands only after the page did.

On a failure at step 3 the archive file is removed; at step 4 `RELEASE.md` is
restored from the held bytes (removed on a first cut) and the archive file is
removed. The report says the steps were rolled back, or names each rollback that
failed with `THE ROLLBACK FAILED: recover by hand`, the shape `rollbackCut` uses.
`rollbackCut` in `internal/surface/cli/ship.go`, which undoes a cut whose
`--payload-dir` render refused after the heading landed, calls the same undo, so
it restores the page and removes the archive too.

- **First cut**: No `RELEASE.md` exists, so step 2 is skipped and rollback
  removes the new page. The archive's first entry arrives at the second feature
  cut.
- **Fixes only**: An empty set writes neither the archive nor the page. The
  changelog heading is written as today, and the report says: "No release page
  written: no user-facing intent shipped in this cut; RELEASE.md stays on
  <version>" (or "no RELEASE.md exists yet").

`IngestResult` gains `Page` (`json:"page"`): `written`, `path`, `heading`,
`archived` (the archive path, empty when nothing moved), `headlines`, `listed`,
`quotes`, and `reason` when no page was written.

### 7. The page

`renderPage` produces, deterministically from the validated payload:

```markdown
# Release 0.10.0 (2026-09-24)

<headline paragraph> (itd-121)

> <quote text> (itd-121)

<headline paragraph> (itd-...)

Also in this release:

- <Title of the intent> (itd-...)

The line-by-line record of this release is its section in CHANGELOG.md.
```

The heading carries the bare version, as the changelog heading does
(`datedHeading`), and the same clock's date. It is parsed back by one regular
expression, `^# Release (\d+\.\d+\.\d+) \((\d{4}-\d{2}-\d{2})\)$`, used both to
name the archive file and to assert, before writing, that the rendered heading
names the cut's own version. A page left in place by a fixes-only release
therefore says which release it describes. Quotes sit directly under the
headline that cites their record. Titles in the name list pass through
`termsafe.CleanProseLine`. The page carries no other heading.

### 8. The preview and the report

`renderCut` is shared by `abcd changelog` and the ship verb's emit step. It gains
one block after the entries: `release page: N intent(s)` followed by the ids and
titles marked `in_press_release`, or `release page: none (no user-facing intent
shipped; RELEASE.md stays as it is)`. `renderEntries` marks those entries
`(on the release page)` beside the existing `(excluded from the changelog)`.
Both commands still write nothing. The `launch --dry-run` report does not render
the cut today (the command page's Preview section), so it gains nothing here.

`renderIngest` adds the page line after `wrote:`: the page path and heading, the
archived path, and the counts; or the no-page sentence from section 6.

### 9. Gates and doc surfaces

- **`.abcd/docs-lint.json`**: `RELEASE` joins the `stray_root_docs` allowlist
  (a blocker rule; matched on the upper-cased basename stem).
- **`.github/workflows/ci.yml`**: `RELEASE.md` joins `is_inert_path`'s root file
  list, beside `CHANGELOG.md`.
- **`.abcd/development/releases/README.md`**: New, stating what the archive
  holds and that its pages are written by the cut, never by hand;
  `directory_coverage` warns on a record directory without one.
- **`.abcd/record-lint.json`**: `.abcd/development/releases/` joins
  `forbidden_synonyms.exempt_prefixes`. The archive is frozen prose carried from
  `intents/shipped/`, which is already exempt, so a quote that passes there must
  not fail here. Every other record-lint rule, `harness_leak` and
  `persona_registry` included, still reads the archive.
- **`commands/launch.md`**: The frontmatter description and opening ("It writes
  **one** file") become three; step 2 documents the `press_release` payload;
  steps 2 and 3 document the retry loop and its reason codes; step 3 documents
  the write order, rollback, first-cut and fixes-only cases and the report.
- **The launch surface page** (`.abcd/development/brief/04-surfaces/04-launch.md`):
  Its prose says the ship verb writes the page and moves the archive. On `main`
  the page carries a generated appendix, which is regenerated, not edited.
- **The agents catalogue** (`.abcd/development/brief/05-internals/01-agents.md`):
  The `release-changelog-composer` entry names both documents.

The launch payload bundle is unaffected: `.abcd/config/launch-payload.json`
includes `README.md` and not the root's other prose files, and the `.abcd/`
namespace is denied structurally.

## Footprint

- `internal/core/changelog`: `shipped.go` (`PressReleaseRequired`,
  `Record.PressRelease`), `source.go` (`pressReleaseSection`).
- `internal/core/release`: `emit.go` (`Entry.InPressRelease`), `ingest.go`
  (schema 2, page validation call, write handoff), new `page.go`, `write.go`,
  `payloadrefusal.go`.
- `internal/surface/cli/ship.go`: payload-refusal JSON, report lines,
  `rollbackCut` through the core undo, the preview block.
- Agent surface: the prompt, three fixtures, `agents/CHANGELOG.md`.
- Config and docs: section 9.

## How the acceptance criteria are satisfied

Test names are the ones to be written; files are where they go.

1. **Three writes in order, rolled back on failure, first-cut removal, report.**
   Section 6. Tests in `internal/core/release/write_test.go`:
   `TestCutWritesArchivePageThenHeading` (order observed through an injected
   writer seam), `TestCutRollsBackEveryEarlierWrite` (fail at each step, assert
   the tree is byte-identical to before), `TestFirstCutRollbackRemovesThePage`,
   `TestCutReportsAFailedRollback`, `TestArchiveNeverOverwrites`. In
   `internal/surface/cli/ship_payload_test.go`:
   `TestRollbackCutRestoresThePageAndArchive`.
2. **Every intent in the set cited, nothing outside it, refused otherwise.**
   Sections 1 and 4. Tests: `TestPressReleaseRequiredIsAddedUserFacingIntents`
   in `internal/core/changelog/shipped_test.go` (issues, internal, removed and
   `shipped_in` records excluded); `TestPageBijection` in
   `internal/core/release/page_test.go` (missing, outside-set by each cause,
   duplicate).
3. **No planned or `target_release` intent cited; no forward-looking sentence.**
   The structural half is `outside-set`: `TestPageRefusesAPlannedIntent` and
   `TestPageRefusesATargetReleaseIntent` in `page_test.go`. The composer half is
   `no-forecast.json`, whose expected payload
   `TestComposerFixtureExamplesIngest` ingests against the fixture's cut and
   checks against its `must_not_contain` list.
4. **The preview lists the set and writes nothing.** Section 8. Tests in
   `internal/surface/cli/ship_test.go`: `TestChangelogPreviewListsThePageSet`,
   `TestChangelogPreviewSaysNoPageForAFixesOnlyCut`, and
   `TestChangelogPreviewWritesNothing` extended to assert `RELEASE.md` and the
   archive are untouched; `TestEmitMarksInPressRelease` in `emit_test.go`.
5. **Empty set: page and archive untouched, the report says why.** Section 6.
   Tests: `TestFixesOnlyCutLeavesThePageAlone` in `write_test.go`,
   `TestLaunchShipFixesOnlyReportsNoPage` in `ship_test.go`,
   `TestPageForAnEmptySetIsRefused` in `page_test.go`.
6. **Structural refusals, nothing written, reasons back to the composer until
   valid, every attempt reported.** Sections 4 and 5. Tests:
   `TestPagePayloadRefusals` in `page_test.go` (one row per code: unknown
   field, oversize, malformed id, outside set, heading, fence, outbound policy;
   each asserts no file changed), `TestPageRefusalCollectsEveryReason`,
   `TestLaunchShipPayloadRefusalJSON` in `ship_test.go` (exit 2 with
   `payload_refusal` and stable codes), and `TestLaunchPageNamesEveryRefusalCode`
   pinning `commands/launch.md` to the enum and to its "no attempt limit" and
   "report every refused attempt" instructions. The loop itself is host prose;
   it is exercised at a real cut, not in CI.
7. **Quotes verbatim, with attribution, from a cited intent.** Section 4.
   Tests: `TestQuoteMustBeVerbatim` in `page_test.go` (a changed word, a changed
   attribution, an attribution outside the text, a quote from the intent's body
   outside `## Press Release`, a quote from an uncited intent, one that cleaning
   would alter); `TestPressReleaseSectionExtractsTheWholeSection` in
   `source_test.go`.
8. **The heading names its release; the canary shows hostile text treated as
   content.** Section 7 and section 3. Tests: `TestPageHeadingNamesItsVersion`
   and `TestArchiveIsNamedFromTheOutgoingHeading` in `page_test.go`;
   `TestComposerFixtureExamplesIngest` runs
   `injection-canary-press-release.json`'s expected payload through the ingest
   and asserts it passes and carries none of the fixture's `must_not_contain`
   strings. The existing `agent_contract` tests keep the canary present.

## Risks and not verified

- **An unbounded loop may not converge.** A composer that keeps producing the
  same fault (for instance a quote the source does not hold) loops until a person
  stops it. This is the product thinker's ruling (Decision 9); loud staging is
  what makes it visible.
- **The canary proves the expected output, not the model.** CI checks that each
  fixture's expected payload is valid and clean; whether a given model obeys the
  prompt is shown only when a host runs the fixture. The prompt stays in the
  `0.x` band for that reason.
- **Quotes are optional to the binary.** The prompt tells the composer to carry
  each told intent's quote; the binary checks the quotes carried (AC7) and does
  not refuse a page that omits one, because no criterion asks it to.
- **Byte-exact quotes.** Straight and curly quotation marks must match the source
  exactly; the check normalises whitespace and blockquote markers only.
- **A degraded scanner config stops the cut.** `CheckOutbound` fails closed on
  an unreadable `.abcd/config/pii.json`; that is the intended direction.
- **The first page waits on the cut.** `abcd changelog` at this tip refuses on
  eight unfixed `major` findings captured since v0.9.0, none of them this
  intent's; the page arrives with the first cut that passes.
- **Not verified:** that `persona_registry` and `abcd lint`'s privacy rule treat
  a root `RELEASE.md` as this spec assumes (the archive is inside record-lint's
  root; the root page is not); and the exact shape of `main`'s generated
  appendix on the launch surface page, since this worktree predates it.
  Checked: `forbidden_synonyms` is the only record-lint rule that exempts
  `intents/shipped/` (the global `exempt_paths` are `research/` and
  `intents/superseded/`), so it is the only exemption the archive needs.
