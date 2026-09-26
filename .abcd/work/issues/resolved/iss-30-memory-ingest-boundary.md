---
schema_version: 1
id: "iss-30"
slug: "memory-ingest-boundary"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "2026-07-08 multi-agent review"
found_at: "internal/core/memory/ingest.go"
resolution: "Remainder drained: detectors added for the URL-ingest success path, the content-type matrix, PDF extraction via the injectable PDFExtractor seam, and the YAML block-scalar / double-quoted-escape parser cases. No new dependency. The parser detectors found and fixed TWO real defects in yaml.go — an empty block scalar swallowing the rest of the document, and dumpString not quoting a carriage return — each of which could strip a written page's provenance silently."
impact: internal
---

memory ingest input-boundary defects: HTTP status is never checked so 404/500 error pages are silently ingested as source content (internal/core/memory/ingest.go:558-575); tilde expansion mangles ~user paths into home+user concatenations (ingest.go:579-584); a --keep-original failure after the page write reports total failure although pages and registry were durably mutated (ingest.go:301-311); CRLF pages are accepted by parseFrontmatter but rejected by splitFileFrontmatter so hashes and summaries silently degrade (yaml.go:558-591); the URL-ingest success path, content-type handling, PDF extraction, and original-storage are untested, as are YAML block scalars and double-quoted escapes. Detector: an ingest-boundary test suite — fetch status matrix, content-type matrix, CRLF round-trip, tilde cases, partial-failure reporting, parser-parity cases. Acceptance corpus: the six instances above.

---

**Progress (2026-07-12, /abcd:run burst 2 — partial, issue stays OPEN):** two
instances drained behind their detectors:

- `--keep-original` partial-failure reporting — FIXED. A `storeOriginal` failure
  after the durable page+registry write now records `IngestResult.KeepOriginalError`
  and returns the successful ingest (both the `ingested` and `registry_only`
  paths); the CLI renders a warning + non-zero exit. The message is path-safe
  (strips `*os.PathError`/`*os.LinkError` paths; repo-relative `sourcesRelPath`).
  Tests: `TestIngestKeepOriginalFailureStillReportsIngest`,
  `TestKeepOriginalErrorMessageNoPathLeak`, `TestMemoryIngestKeepOriginalPartialFailure`.
- CRLF parser-parity — FIXED. `splitFileFrontmatter` normalises line endings like
  `parseFrontmatter`/`frontmatterKeyLine`, so CRLF documents split identically to
  their LF form. Test: `TestSplitFileFrontmatterCRLFParity`.

HTTP-status and tilde-expansion instances were addressed earlier (PR #38).

**Remainder (kept this issue open until 2026-08-01):** the untested ingest surfaces named in the
detector — URL-ingest success path, content-type matrix, PDF extraction, and
YAML block-scalar / double-quoted-escape parser cases. PDF extraction coverage
may require a dependency; assess against the no-new-dep STOP before adopting.

**Closed (2026-08-01, v0.5.0 item C7):** the remainder is drained and the
acceptance corpus is complete. The four untested surfaces now have detectors in
`internal/core/memory/ingest_boundary_test.go` and
`internal/core/memory/yaml_boundary_test.go`, both driven through the existing
injectable seams — `Fetcher` for the URL path, `PDFExtractor` for PDF text — so
the no-new-dep STOP never fired: `go.mod` is unchanged, and the nil-extractor
refusal is itself one of the covered behaviours rather than a reason to adopt a
parser. Covered: the URL-ingest success path (requested vs final URL, headers
reaching licence detection, kept-original extension from the URL path); the
content-type matrix (`text/*` prefix, the three-entry allowlist and a near-miss
outside it, `application/pdf`, the non-text rejection, and a lying `text/plain`
whose bytes still fail the decode); PDF extraction on the fetched path and both
local sniffs (`.pdf` extension, `%PDF-` magic) across nil/error/empty/success
extractors; and the parser cases — literal `|` block scalars (de-indent,
interior and trailing blank lines, opaque content, block as last key) with the
unsupported `>`/`|-`/`|+` indicators pinned as loud refusals, plus the full
`unescapeDoubleQuoted` escape set, both error paths, and both call sites.
Original-storage — the fourth instance-5 sub-surface — was already covered by
`TestIngestKeepOriginalWritesSourceCanonically` and
`TestIngestKeepOriginalFailureStillReportsIngest` in
`internal/core/memory/memory_test.go`, and the new URL and PDF cases add the
extension-derivation and original-bytes assertions those did not reach, so the
completeness claim rests on both. Every case was mutation-checked against a
targeted reversion of the behaviour it claims to pin.

**Correction (2026-08-01, same day):** the closure above originally claimed "no
production behaviour was found to diverge from its documentation, so this
increment is test-only". That was WRONG, and pre-PR review caught it: the new
parser detectors, once written against the real writer/reader boundary rather
than the parser's internals, reproduced TWO defects in
`internal/core/memory/yaml.go`, both fixed here.

- `collectBlockScalar` took the block's indent from the first non-blank line
  without requiring it to exceed the indent of the `key: |` line that opened the
  block. A `|` with no INDENTED body therefore read the next unindented line as
  content and kept consuming — the rest of the document, other keys included,
  disappeared into the value with NO error. Fix: an unindented first line ends an
  EMPTY block and is left for the top-level parse. Detectors:
  `TestParseBlockScalar/empty_block_does_not_swallow_the_next_key` (+ its
  blank-line sibling) and `TestParseBlockScalarEmptyBlockKeepsLaterKeys`.
- `dumpString` triggered quoting on `\n`/`\t` but not `\r`, so a bare carriage
  return was written raw into the YAML region. `parseFrontmatter` normalises
  `\r` to `\n` BEFORE splitting, so a value carrying `\r---` re-read as an early
  frontmatter terminator and every key below it fell into the page body — a
  distiller-supplied `recall` string was enough to make `Ingest` report success
  while writing a page that reads back with no `source` block and no source
  hashes, the state the repair path treats as an orphan. Fix: `\r` joins the
  quote trigger (`doubleQuote` already escaped it). Detectors: the file-boundary
  call site added to `TestDoubleQuoteEscapeRoundTrip` (the pre-existing `\r`
  subjects passed only because they never crossed `dumpFrontmatter` →
  `joinFileFrontmatter` → `parseFrontmatter`) and
  `TestIngestDistilledControlCharsKeepProvenance`, which drives the whole chain
  through `Ingest`. Both were watched fail before the fix and pass after.

So the increment is NOT test-only: it carries two production fixes and a
CHANGELOG entry under Fixed.
