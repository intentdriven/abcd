# Sources, citations, licensing, provenance and attribution: state of the art

Dated 2026-10-23; recency is assessed against that date. A state-of-the-art pass
over the open intents that handle where a source comes from, whether the record
may lean on it, and how credit travels: [itd-126](../../intents/drafts/itd-126-a-team-shares-one-bibliography-without.md)
(a team shares one bibliography without sharing anyone's corpus),
[itd-127](../../intents/drafts/itd-127-a-paper-is-reconstructed-from-the.md)
(a document is reconstructed from the provenance ledger),
[itd-163](../../intents/drafts/itd-163-a-cited-source-always-resolves-and-an.md)
(the reference-closure and acknowledgements-mirror gate),
[itd-164](../../intents/drafts/itd-164-a-source-is-licence-vetted-before-the.md)
(a source is licence-vetted before the record may lean on it),
[itd-2609292108089653](../../intents/drafts/itd-2609292108089653-ingesting-a-source-and-consulting-the.md)
(ingest and consult as abcd verbs, no keyword stubs),
[itd-26](../../intents/drafts/itd-26-loot-oss-vendor.md) (vendoring code with its
papers), [itd-171](../../intents/drafts/itd-171-a-decision-recorded-in-an-intent-an-adr.md)
(a decision points back to the conversation where it was reached) and
[itd-91](../../intents/drafts/itd-91-ai-attribution-preference.md) (a project
declares once how it wants AI acknowledged). Every source linked below was opened
in this pass; a source that could not be opened is named in words under "Not
verified" and carries no link. Three earlier notes already settle part of this
ground and are reused rather than redone: the [confidential-sources survey of
2026-07-08](2026-07-08-confidential-sources-provenance-sota.md) (CSL-JSON over
BibTeX, JSONL ledger with PROV vocabulary, grep over a converted corpus, Quarto
for the render), the [academic references baseline of
2026-08-16](2026-08-16-academic-references-baseline.md) (admission criteria for
the committed bibliography) and the [session-recording survey of
2026-07-30](2026-07-30-session-recording-sota.md) (transcripts out of the
repository, linked from commits by trailer).

Evidence tiers: [EVIDENCE] a measurement, a dataset or a live response from the
system itself; [CONSENSUS] several independent sources, or a published standard
and the tools that implement it; [CONTESTED] credible disagreement, both sides
given; [ANECDOTE/MARKETING] one source, a vendor's own claim, or a community
write-up with no method.

This is the proposer's half of [prefer-sota](../../principles/prefer-sota.md).
The independent fit-challenge is still owed at plan time; the "Local fit risks"
under each intent are its input, not its verdict.

## itd-126: a team shares one bibliography without sharing anyone's corpus

### Findings

- CSL-JSON requires `type` and `id`, forbids unknown top-level keys
  (`additionalProperties: false`), and reserves `custom` "to store additional
  information that does not have a designated CSL JSON field", preferred over
  `note` for key-value pairs. The corpus already keeps its `confidential`,
  `permission_status`, `keywords` and `aliases` there, so provenance of a shared
  entry (who shared it, from which repository, when) has a sanctioned home that
  renders in no citation style. [CONSENSUS] [CSL-JSON input schema](https://raw.githubusercontent.com/citation-style-language/schema/master/schemas/input/csl-data.json).
- Zotero, the most widely used shared-bibliography tool, resolves a concurrent
  edit at item level: "If an item has changed in multiple places in conflicting
  ways between syncs, you'll receive a conflict resolution dialog asking which
  version you'd like to keep"; non-conflicting changes merge automatically. A
  group library holds "separate copies" of items, and a change to the group copy
  does not reach a member's own copy until they drag it back. [CONSENSUS]
  [Zotero sync](https://www.zotero.org/support/sync),
  [Zotero groups](https://www.zotero.org/support/groups).
- Zotero's duplicate merge is the field-level model: duplicates are detected on
  title, DOI and ISBN, then year and creators; the person picks a master item and
  chooses, field by field, which version survives, and "citations and
  bibliographies aren't affected". That is the human-run shape of "two teammates
  share the same source with divergent metadata". [CONSENSUS]
  [Zotero duplicate detection](https://www.zotero.org/support/duplicate_detection).
- Better BibTeX keeps citation keys stable by generating them "in a
  deterministic way" against the keys already in the library and resolving a
  clash with "a letter postfix (a, b, c, etc)"; a changed key pattern touches only
  items edited afterwards. Key stability is treated as a property of the store,
  never recomputed per export. [CONSENSUS]
  [Better BibTeX citing](https://retorque.re/zotero-better-bibtex/citing/).
- A committed CSL-JSON file is an array; git merges it line by line, so two
  teammates appending entries in the same place conflict on the file even when
  the entries differ. Keeping entries sorted by `id`, one entry per block, and
  refusing a share whose key already exists with different content moves the
  conflict from git's textual merge to abcd's refusal, where it can name the
  key. (This is an inference from the format, not a measured finding.)

### Options weighed

1. Zotero group library as the team surface (Path 1). It shares corpora, not
   bibliography; file storage counts against one owner's quota; it is a hosted
   service and the corpus never travels is the whole point of the intent.
   Rejected.
2. A second exchange file beside `references.csl.json`. Already ruled out by the
   maintainer on 2026-08-16 (one committed bibliography); the admission criteria
   of the [references baseline](2026-08-16-academic-references-baseline.md) then
   apply to shared entries too.
3. Native `share` and `ingest` over the committed CSL-JSON, with provenance in
   `custom` and conflicts refused by key (Path 2).

### Recommendation

**Path 2.** `abcd source share` writes a public entry into
`references.csl.json` with a `custom.shared` block (`by`: a repository handle,
never a person's name or mail; `from_repo`; `shared_on`; the corpus key it came
from), sorted by `id`. Key ownership is first-sharer: a second share of an
existing `id` with different bibliographic fields is refused, printing the
field-level diff, and `--replace` is the deliberate overwrite, which is Zotero's
pick-a-master merge reduced to the one case abcd can decide mechanically. `abcd
source ingest` imports entries as public and keeps the `custom.shared` block, so
a team bibliography stays auditable from the corpus side. The seam is the
entry: it stays pure CSL-JSON, so a later move to a shared Zotero library or a
CFF `references` list is an exporter, not a migration. No new Go dependency.

### Local fit risks

- The baseline's admission criteria (primary sources, verified metadata, a
  resolver link) were written for hand-curated entries; a mechanical share must
  either enforce them or mark a shared entry as unverified
  ([references baseline](2026-08-16-academic-references-baseline.md)). This is
  the intent's third open question and needs the ruling before planning.
- [adr-41](../../decisions/adrs/0041-corpus-trust-boundary.md): the refusal of
  a confidential entry must name the gate and never echo the entry's strings;
  `custom.shared.by` must not become a path for a person's identity to enter a
  committed file (the attribution rules refuse personal paths and mails).
- [one-canonical-primitive](../../principles/one-canonical-primitive.md): the
  corpus package already parses CSL-JSON and preserves unknown fields
  byte-for-byte; share and ingest must use that reader, not a second one.

## itd-127: a document is reconstructed from the provenance ledger

### Findings

- Quarto project profiles still give the structural firewall the 2026-07-08
  note chose: profile config "is merged with the top-level `_quarto.yml`",
  content is shown or hidden with `.content-visible` plus `when-profile`, and a
  default profile can be declared. One caveat found this pass: "`metadata-files`
  are not resolved in profiles", so the public profile must name its filtered
  bibliography in the profile config itself. [CONSENSUS]
  [Quarto profiles](https://quarto.org/docs/projects/profiles.html).
- Pandoc, the engine under Quarto, reads bibliographies by extension: `.json`
  (CSL JSON), `.yaml` (CSL YAML), `.bib`, `.bibtex`, `.ris`; `nocite` adds
  entries without an in-text citation, and the wildcard builds "a bibliography
  with all the citations, whether or not they appear in the document".
  `--fail-if-warnings` exits non-zero on any warning. Whether a missing citation
  key raises a warning is not stated in the manual (see "Not verified").
  [CONSENSUS] [Pandoc manual](https://pandoc.org/MANUAL.html).
- Typst accepts only "A BibLaTeX `.bib` file" or "A Hayagriva `.yaml`/`.yml`
  file"; CSL JSON is not an input, though CSL styles are. A Typst-only chain
  would need a `.bib` (or Hayagriva) export of the already-filtered
  bibliography; the gates run before that export, so its cost is a second
  export format, not a lost gate. [CONSENSUS] [Typst bibliography](https://typst.app/docs/reference/model/bibliography/).
- W3C PROV-O (Recommendation, 30 April 2013) names the relations the ledger
  already implies: `wasDerivedFrom` ("a transformation of an entity into
  another"), `wasQuotedFrom` ("cites a potentially larger Entity ... from which
  a new Entity was created"), `hadPrimarySource`, `wasAttributedTo`
  ("the ascribing of an entity to an agent"), `wasInfluencedBy`. The earlier
  note's advice stands: borrow the vocabulary, not the format. [CONSENSUS]
  [PROV-O](https://www.w3.org/TR/prov-o/).
- No Go-native CSL processor (citeproc) exists that this pass could open. Go
  PDF generators do exist and are maintained (go-pdf/fpdf, "a PDF document
  generator with high level support for text, drawing and images"), but they
  would leave citation formatting to abcd; the citation-correct render engines
  are Pandoc and Quarto (Haskell binary; Quarto bundles Pandoc and Typst) or
  Typst. A citation-correct PDF or HTML render therefore means an external
  program, on the citeproc ground, which is exactly the intent's one open
  question. [CONSENSUS] [go-pdf/fpdf](https://pkg.go.dev/github.com/go-pdf/fpdf).

### Options weighed

1. Bundle or depend on a renderer (Path 1): a Go dependency does not exist for
   this; shelling to Quarto as a hard requirement breaks the "basics built in"
   rule.
2. Native generation of the inputs plus a deterministic post-render check, with
   the renderer as an opt-in adapter (Path 2).
3. Markdown only, no render (Path 3): abandons the PDF and HTML the press release
   promises.

### Recommendation

**Path 2**, shaped to the Pandoc and Quarto input contract. The binary does the
deterministic work with no dependency: walk the ledger, group claims by
decision, emit one markdown source with `@key` citations, emit the filtered
`public.csl.json` (every entry whose `custom.confidential` is false and whose
`permission_status` permits, and whose ledger line is `cited_publicly`), and run
the post-render check over the output's citation keys against both gates. The
render is an adapter in the sense of
[basics-built-in-adapters-bring-power](../../principles/basics-built-in-adapters-bring-power.md):
`abcd source render` looks for `quarto`, then `pandoc`, on PATH, passes the
generated bibliography with `--fail-if-warnings`, and refuses with the install
instruction when neither is present; the markdown and the filtered bibliography
are the basic form and are useful on their own. The seam is the generated
bibliography file plus the markdown; swapping Quarto for Typst later means a
`.bib` exporter and nothing else changes. No Go dependency, so no sign-off
needed; the adapter is a program the person installs.

### Local fit risks

- [adr-41](../../decisions/adrs/0041-corpus-trust-boundary.md): the filtered
  bibliography is a derived artefact of the corpus; it must be generated into
  the local tier and never committed with confidential strings, and the
  post-render check must run on the rendered text, not only on the keys, since
  a confidential title can enter prose without a citation.
- [adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md): Pandoc
  fetches a bibliography given as a URL; the adapter must pass a local path only.
- [loud-staging](../../principles/loud-staging.md): if the render adapter ships
  after the generator, the render verb must refuse loudly rather than silently
  emit markdown and claim a render.

## itd-163: a cited source always resolves; an adopted inspiration is always acknowledged

### Findings

- REUSE 3.3 (2024-11-14) is the closest published gate of the same shape: every
  covered file carries an `SPDX-License-Identifier`, the project "MUST include a
  License File for every license under which its Covered Files are licensed" in
  `LICENSES/`, and "MUST NOT include License Files for licenses under which none
  of the files in the Project are licensed". That is a bidirectional mirror
  checked, not generated: declarations and the licence set must close in both
  directions. [CONSENSUS] [REUSE specification 3.3](https://reuse.software/spec-3.3/).
- The Citation File Format's `references` key is the nearest standard record
  for "an influence with its licence and where it lives": a reference carries a
  required `type` from an enum that includes `software`, `software-code`,
  `software-container`, `website`, `blog`, `standard`, plus `license` (SPDX,
  several meaning OR), `license-url`, `repository-code`, `commit`, `url` and
  `date-accessed`. [CONSENSUS] [CFF schema guide](https://raw.githubusercontent.com/citation-file-format/citation-file-format/main/schema-guide.md).
- CSL-JSON itself has the `software` item type and the `custom` block, so an
  influence registry can be CSL entries rather than a second format; the
  2026-08-16 baseline kept "tool repositories" out of References as a curation
  rule, which is a rendering choice (which section), not a storage one.
  [CONSENSUS] [CSL-JSON input schema](https://raw.githubusercontent.com/citation-style-language/schema/master/schemas/input/csl-data.json),
  [references baseline](2026-08-16-academic-references-baseline.md).
- Pandoc's `nocite` with the wildcard is the "generated from the store" end of
  the design space: the bibliography lists every entry whether cited or not.
  REUSE's `reuse lint` is the "checked against the store" end. Both exist in
  mature tools; neither is the field's only answer. [CONSENSUS]
  [Pandoc manual](https://pandoc.org/MANUAL.html),
  [REUSE specification 3.3](https://reuse.software/spec-3.3/).
- Licensee, which GitHub runs, ignores README licence mentions by default
  ("there are reasons you might not want to use this") and reads only
  root-level licence files and `LICENSES/`: the one widely deployed detector
  declines to find declarations in prose, which supports the intent's choice to
  leave "a prose-named source that never enters the citation apparatus" to
  review. [CONSENSUS] [licensee customizing](https://github.com/licensee/licensee/blob/main/docs/customizing.md),
  [licensee: what we look at](https://github.com/licensee/licensee/blob/main/docs/what-we-look-at.md).
- Persistent-identifier practice for "a cited source always resolves": the
  Wayback availability API answers whether a capture exists for a URL
  (`archived_snapshots.closest.url`, `timestamp`, `available`), which is the
  cheapest way for the refresh verb, not the gate, to record a fallback
  location when a cited page dies. [CONSENSUS]
  [Wayback APIs](https://archive.org/help/wayback_api.php).

### Options weighed

1. Registry as a separate JSON or YAML file (CFF-shaped) beside the CSL file:
   a standard shape, but a second parser and a second mirror rule.
2. Registry as CSL-JSON entries of type `software` (and `webpage` or
   `standard` where apt) with `custom.abcd.kind: inspiration | tool` and
   `custom.abcd.adopted_in: <record id>`: one file, one reader, one closure
   rule over both sections.
3. One frontmattered markdown file per influence: matches the record's style
   but is the hardest to mirror-check and the easiest to let drift.

### Recommendation

**Path 2**, a native lint family shaped to REUSE's bidirectional closure, with
the registry stored as CSL-JSON entries (option 2) and the acknowledgements
sections checked against the store first, generated later. The closure rule for
the development tier scopes to reference-style links and `@key` tokens whose
target is `references.csl.json` or a key in it. Checking before generating is
[ratchet-not-big-bang](../../principles/ratchet-not-big-bang.md): the gate arms
on the backfilled tree; generation of the Inspirations and References sections
from the store is the next rung and removes drift for good, the way the
changelog already derives from records. The seam is the entry shape: a CFF
export of the `software` entries is a mechanical projection when a consumer
wants one. No Go dependency.

### Local fit risks

- The 2026-08-16 curation rule that tool repositories are Inspirations, not
  References, becomes a `custom.abcd.kind` field rather than a separate file;
  the fit-challenge should confirm the maintainer accepts registry entries in
  the same CSL file as the academic baseline.
- [one-canonical-primitive](../../principles/one-canonical-primitive.md): the
  spc-17 citation rules and the corpus package both read CSL-JSON; the closure
  rule must reuse one of those readers.
- Generating the acknowledgements sections changes how contributors edit credit
  (the intent's second open question); the check-first rung keeps hand editing
  legal until that ruling.

## itd-164: a source is licence-vetted before the record may lean on it

### Findings

- SPDX 2.3.1 separates the two unknowns the intent folds into one. Concluded
  licence `NOASSERTION` means the creator "has attempted to but cannot reach a
  reasonable objective determination", made no attempt, or intentionally gave
  no information; `NONE` means the creator "concludes there is no license
  available". Declared licence `NONE` means "the package contains no license
  information whatsoever". An absent field "implies an equivalent meaning to
  NOASSERTION", and the comments field is where "an explanation ... when the
  Concluded License is NOASSERTION" belongs. [CONSENSUS]
  [SPDX 2.3.1 package information](https://spdx.github.io/spdx-spec/v2.3.1/package-information/).
  SPDX 3.0.1 is the current specification and keeps the two states as the
  `NoAssertionLicense` and `NoneLicense` individuals (the same three
  conditions for NoAssertion); the 2.3.1 tokens are used here on purpose,
  because they are what SPDX expressions and the licence list still carry.
  [CONSENSUS]
  [SPDX 3.0.1, NoAssertionLicense](https://spdx.github.io/spdx-spec/v3.0.1/model/ExpandedLicensing/Individuals/NoAssertionLicense/).
- SPDX expressions: identifiers are case-insensitive but canonical case is
  recommended; operators `AND`, `OR`, `WITH` are case-sensitive; a licence
  outside the list is `LicenseRef-` plus letters, digits, `-` and `.`; `+`
  means or-later. The licence list is at version 3.29.0 (2026-09-16) with
  machine-readable data files published separately, and SPDX "endeavors to
  never change" an identifier, deprecating rather than renaming. [CONSENSUS]
  [SPDX license expressions](https://spdx.github.io/spdx-spec/v2.3.1/SPDX-license-expressions/),
  [SPDX license list](https://spdx.org/licenses/).
- Where a licence can be read, by source type. A DOI: Crossref's work record
  carries a `license` array; the live record for one entry already in the
  baseline returned `URL: https://creativecommons.org/licenses/by/4.0/`,
  `content-version: vor`, `delay-in-days: 0` and a `start` date, and the API
  offers `has-license`, `license.url`, `license.version` and `license.delay`
  filters, with a "polite pool" for clients that put a `mailto:` in the
  User-Agent. [EVIDENCE] [Crossref live record](https://api.crossref.org/works/10.1145/3586030),
  [Crossref REST API documentation](https://raw.githubusercontent.com/CrossRef/rest-api-doc/master/README.md).
  A DataCite DOI: `rights` with `rightsURI`, `rightsIdentifier`,
  `rightsIdentifierScheme` ("SPDX") and `schemeURI`. [CONSENSUS]
  [DataCite 4.6 rights](https://datacite-metadata-schema.readthedocs.io/en/4.6/properties/rights/).
  An arXiv id: the OAI-PMH `arXiv` metadata format "includes author names
  separated out, category and license information" at
  `https://oaipmh.arxiv.org/oai`. [CONSENSUS] [arXiv OAI-PMH](https://info.arxiv.org/help/oa).
  A GitHub repository: `GET /repos/{owner}/{repo}/license` returns a `license`
  object with `spdx_id` or `null`; GitHub "uses the open source Ruby Gem
  Licensee", which matches root licence files at a default confidence of 98 and
  reads package manifests and READMEs only when asked. [CONSENSUS]
  [GitHub licenses API](https://docs.github.com/en/rest/licenses/licenses),
  [licensee customizing](https://github.com/licensee/licensee/blob/main/docs/customizing.md).
  A web page: HTML's `rel="license"` indicates the terms for "the main content
  of the current document", with `copyright` as a legacy synonym and no rule for
  separating main content from the rest; Creative Commons marks pages with
  `rel="license"` plus RDFa `cc:attributionName` and `cc:attributionURL`;
  schema.org's `license` property takes "A license document that applies to
  this content, typically indicated by URL". [CONSENSUS]
  [HTML link types](https://html.spec.whatwg.org/multipage/links.html),
  [CC REL](https://wiki.creativecommons.org/wiki/CC_REL),
  [schema.org license](https://schema.org/license).
- Licence-to-SPDX mapping is not given by any of those sources: Crossref and CC
  REL return a URL, DataCite may return an SPDX id, GitHub returns one. A URL
  to identifier table (the CC licence URLs, the OSI pages) is a small data file
  abcd must own.
- ClearlyDefined distinguishes "declared" (what a component states) from
  "discovered" (what tools found in its files) and keys components by
  `type/provider/namespace/name/revision`; the same two-field split is what
  SPDX's declared versus concluded encodes. [CONSENSUS]
  [ClearlyDefined: using the data](https://docs.clearlydefined.io/docs/get-involved/using-data).
- The existing refresh already writes a per-URL baseline with `outcome`,
  `verification: automatic | manual` and `verified_on`, and preserves a current
  manual receipt untouched. A licence verdict is the same kind of fact (a check
  receipt with a date), not bibliographic data. [EVIDENCE: the repository's own
  `.abcd/citations-baseline.json` and `internal/core/cite/refresh.go`.]

### Options weighed

1. Verdict in CSL `custom` (one file per source): keeps everything about a
   source together, but puts a dated network receipt into a file that is
   otherwise pure bibliography and that share and ingest (itd-126) will move
   between repositories and corpora.
2. Verdict in the citation baseline keyed by the entry's resolver URL (the
   file the refresh already owns): keeps CSL pure and the gate's read path
   unchanged; the baseline grows `license: {spdx, source_url, checked_on,
   verification, reason}`.
3. Free-text licence field normalised on write: invites "MIT-spirit" again.

### Recommendation

**Path 2**, shaped to SPDX 2.3.1 declared-licence semantics and the existing
baseline. The verdict lives in the citation baseline (option 2) under the
entry's resolver URL, with the CSL entry unchanged; the gate refuses a new entry
with no verdict and accepts `NOASSERTION` or `NONE` only when the entry's
caveat is present. The verdict field accepts exactly: a licence expression
whose identifiers are on a pinned copy of the SPDX list (a CC0 data file
embedded at build, refreshed deliberately, not a Go module), a `LicenseRef-`
identifier with a `reason`, `NONE`, or `NOASSERTION` with a `reason`; anything
else is refused before write. Refresh resolves by source type in this order:
DOI through Crossref (or DataCite when Crossref has no record), arXiv through
OAI-PMH, a GitHub URL through the licence endpoint, any other URL through
`rel="license"`, CC REL and schema.org `license` in the fetched page; a URL
that maps to no identifier is recorded `NOASSERTION` with the URL seen. A
human-attested verdict is `verification: manual` with the same preserve rule
the link receipts already have, and ages on the same windows, since the
attestation is a reading of a page at a date like any other. Grandfathering ends
when the backfill lands, not at a date: the gate reads "has a verdict", and a
dated deadline would make it read the calendar. No Go dependency.

### Local fit risks

- [adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md): three
  new network endpoints (Crossref, DataCite, arXiv OAI, GitHub) enter the
  refresh verb; they must stay behind the explicit verb and the polite-pool
  identification must not carry a person's mail by default (the attribution
  rules refuse mails in committed files; a configured contact is the
  repository's, opt-in).
- The intent's scope excludes compatibility analysis; the declared versus
  concluded split must not creep into a "concluded" field abcd fills by
  judgement.
- Embedding the SPDX list is data under a licence (CC0) that still wants an
  `ACKNOWLEDGEMENTS.md` line in the change that lands it.

## itd-2609292108089653: ingesting and consulting sources are abcd verbs

### Findings

- Extraction quality is measured. On the trafilatura benchmark of 2026-10-02
  (990 documents, 2,951 text and 2,966 boilerplate segments) trafilatura 2.3.0
  scores F 0.926, readability-lxml 0.853, jusText 0.862, newspaper4k 0.803,
  boilerpy3 0.807. The benchmark is the tool's own. [EVIDENCE, self-reported]
  [trafilatura evaluation](https://trafilatura.readthedocs.io/en/latest/evaluation.html).
- A Go port exists: go-trafilatura v2.2.6 (Apache-2.0, Go 1.26 or newer),
  extracting from HTML the caller already fetched, with the maintainer's own
  benchmark placing it ahead of go-readability and go-domdistiller on two of
  three corpora and slowest of the three. Adopting it is a Go dependency and
  needs sign-off. [ANECDOTE/MARKETING for the figures; EVIDENCE that the port
  exists] [go-trafilatura](https://github.com/markusmobius/go-trafilatura).
- The "keyword stub" test the intent needs is the jusText block classifier:
  stopword density below 0.30 marks a block bad, above 0.32 good for long
  blocks; link density above 0.2 marks boilerplate; length thresholds 70 and
  200. A stored text that is only keywords has near-zero stopword density and
  fails that test with no model and no dependency. The density is computed
  against a per-language stoplist (the tool is called as
  `justext.justext(content, justext.get_stoplist("English"))`, with `-s Czech`
  on the command line), and the bounds are block-level defaults for HTML
  classification, so the check must pick the stoplist by the entry's language,
  or skip with a warning when the language is unknown, and recalibrate the
  bounds for whole stored texts. [CONSENSUS]
  [jusText algorithm](https://corpus.tools/wiki/Justext/Algorithm),
  [jusText README](https://github.com/miso-belica/jusText).
- PDF text in pure Go is weak: pdfcpu (Apache-2.0) lists extraction of
  "images, fonts, and metadata" and does not claim plain text; ledongthuc/pdf
  (BSD-3-Clause, last published 2026-10-20) does extract text but its own bug
  list says "the package is incomplete" with weak encrypted-file support.
  Poppler's `pdftotext` converts "to plain text" with reading-order or
  `-layout` modes and UTF-8 output, and is the tool the 2026-07-08 note already
  assumed. [CONSENSUS] [pdfcpu](https://github.com/pdfcpu/pdfcpu),
  [ledongthuc/pdf](https://pkg.go.dev/github.com/ledongthuc/pdf),
  [pdftotext manual](https://manpages.debian.org/testing/poppler-utils/pdftotext.1.en.html).
- Consult: the 2026-07-08 note's grep-first ruling stands; its named upgrade,
  SQLite FTS5, is reachable without CGO through modernc.org/sqlite v1.61.0
  (BSD-3-Clause, SQLite 3.54.1, 2026-10-21), whose documentation mentions fts5
  shadow tables, so FTS5 is compiled in. A Go dependency; sign-off needed if
  that rung is ever climbed. [CONSENSUS]
  [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite),
  [confidential-sources survey](2026-07-08-confidential-sources-provenance-sota.md).
- Fetching: [adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md)
  bars an implicit fetch, not an explicit one; the ingest page's "abcd fetches
  and converts nothing" is a page-level statement older than the ADR.

### Options weighed

1. Fetch and convert inside the binary with go-trafilatura and a PDF library
   (Path 1 for the libraries): best HTML quality, new dependencies, poor PDF.
2. Deterministic halves native (search by key and class, the stopword-density
   quality check at add time and in lint; the backfill of the six stubs as a
   documented act against the corpus, not a verb), with conversion through installed tools
   (`pandoc`, `pdftotext`, `trafilatura` CLI) when present and the agent's own
   fetch-and-paste when not (Path 2).
3. Everything stays in the command page (Path 3): the ruling of 2026-09-29 is
   against it.

### Recommendation

**Path 2.** `abcd source consult <query>` is a native, read-only search over
the converted corpus (key, class, keywords, aliases, then body), confidential
entries by key only. `abcd source add` gains the quality check: refuse when the
stored text's stopword density is under the jusText low bound or its token
count is under a floor, for an entry that claims a fetchable URL; `abcd source
lint` reports the same over existing entries. `abcd source ingest <url|file>`
is explicit, so it may fetch; it converts through an adapter chain on PATH
(`pdftotext` for PDF, `pandoc` for office formats, the trafilatura CLI or
`pandoc -f html` for HTML) and, when the chain is empty, writes the fetched
bytes to the corpus staging folder and prints the one-line instruction for the
agent to convert and re-run with `--text`. The seam is the converter interface
(bytes and media type in, text out), so go-trafilatura can replace the HTML
rung later under its own sign-off. The backfill of the six stub entries is a
documented act against a person's corpus, verified by `source lint` rather than
by a build criterion.

### Local fit risks

- [the-users-directory-is-theirs](../../principles/the-users-directory-is-theirs.md):
  staging fetched bytes must stay inside the corpus directory abcd owns.
- [consult-freely-cite-deliberately](../../principles/consult-freely-cite-deliberately.md):
  the consult output is read by an agent that may write public artefacts; the
  key-only rule for confidential hits must hold in `--json` as well as text.
- [script-first-mvp](../../principles/script-first-mvp.md): the stopword
  threshold and token floor want calibrating on the real corpus before they
  harden into the binary.

## itd-26: vendoring code with its papers

### Findings

- SWHID is an ISO/IEC standard (ISO/IEC 18670; the number and the 23 April
  2025 date come from the OpenChain announcement, not the spec page), an
  intrinsic identifier computed from content alone: a content id is the SHA1
  of `blob <length>\0` plus the bytes, "compatible with the way the current
  version of Git proceeds", though the spec adds that "Git compatibility is
  practical, but incidental and is not guaranteed to be maintained in future
  versions"; a directory id hashes the sorted entries with mode, name and id,
  like a git tree. Qualifiers carry context without changing the core:
  `origin=` (an origin "can change or disappear over time"), `visit=`,
  `anchor=`, `path=`, and `lines=` or `bytes=` for a fragment. A provenance
  card can therefore name each looted file by an id that survives the upstream
  repository's deletion and that a SHA-1 git can recompute offline; a SHA-256
  git repository computes different object ids and needs the SHA-1 computed
  separately. [CONSENSUS]
  [SWHID core identifiers](https://www.swhid.org/specification/v1.2/5.Core_identifiers/),
  [SWHID qualified identifiers](https://www.swhid.org/specification/v1.2/6.Qualified_identifiers/),
  [OpenChain on ISO/IEC 18670](https://openchainproject.org/news/2025/04/29/from-heritage-to-hash-swhid-becomes-a-global-iso-iec-standard).
- Existing vendoring tools record the commit, not the content. `git subtree
  --squash` writes `git-subtree-dir:` and `git-subtree-split:` trailers into
  the squash commit and needs "no special constructions (like '.gitmodules'
  files or gitlinks)"; vendir's lock records `sha`, `commitTitle` and `tags`
  per git source; Go's `vendor/modules.txt` lists "vendored packages and the
  module versions they were copied from" and the `go` command errors when it
  disagrees with `go.mod`. None records a licence. [CONSENSUS]
  [git-subtree source](https://raw.githubusercontent.com/git/git/master/contrib/subtree/git-subtree.sh),
  [git-subtree manual](https://raw.githubusercontent.com/git/git/master/contrib/subtree/git-subtree.adoc),
  [vendir lock](https://carvel.dev/vendir/docs/v0.46.x/vendir-lock-spec/),
  [Go modules reference](https://go.dev/ref/mod).
- Licence detection layers already have a reference implementation: licensee
  reads root licence files and `LICENSES/` by default, package manifests and
  READMEs only when enabled, matches at 98 per cent confidence, and treats a
  copyright-only file as "the project isn't licensed". GitHub exposes the
  result per repository and per `ref` as `spdx_id` or `null`. [CONSENSUS]
  [licensee](https://github.com/licensee/licensee),
  [licensee customizing](https://github.com/licensee/licensee/blob/main/docs/customizing.md),
  [GitHub licenses API](https://docs.github.com/en/rest/licenses/licenses).
- REUSE 3.3 gives the per-file declaration for the vendored tree: a header or
  a `.license` sidecar with `SPDX-FileCopyrightText` and
  `SPDX-License-Identifier`, or a `REUSE.toml` with `[[annotations]]` for
  directories where headers are impractical, and the licence text in
  `LICENSES/<id>.txt`. [CONSENSUS] [REUSE specification 3.3](https://reuse.software/spec-3.3/).
- SPDX expressions give the policy list its grammar (`MIT OR Apache-2.0`,
  `GPL-2.0-or-later`, `LicenseRef-` for a custom licence); `NOASSERTION` is a
  defined value, and under the intent's no-bypass rule it is a refusal.
  [CONSENSUS] [SPDX license expressions](https://spdx.github.io/spdx-spec/v2.3.1/SPDX-license-expressions/).
- Compatibility data exists in machine-readable form: OSADL's checklists
  encode obligations as `YOU MUST` and `YOU MUST NOT` atoms with a
  compatibility construct, in text and JSON. The intent scopes compatibility
  out in favour of an allow-list, so this is a later rung at most. [CONSENSUS]
  [OSADL checklists](https://www.osadl.org/OSADL-Open-Source-License-Checklists.oss-compliance-lists.0.html).

### Options weighed

1. `git subtree` as the mechanism (Path 1): records the split commit but not
   file selection, licence or rationale; the card still has to exist.
2. A native card shaped to SWHID, SPDX and REUSE, with GitHub's licence
   endpoint as the first detection layer and local licensee-style matching as
   the second (Path 2).
3. A bespoke card with a bare URL and SHA (Path 3): what the draft describes
   today; loses the content-addressed id that makes the card verifiable after
   the origin disappears.

### Recommendation

**Path 2.** The card keeps the draft's fields and adds, per file, the SWHID
content id and, for the set, the directory id, computed natively (it is git's
blob and tree hash, no dependency); `loot verify` recomputes them offline and
only then compares the licence text at the recorded SHA. `vendor/<source>/`
carries a `REUSE.toml` annotation naming the licence and copyright of the
looted files, so a REUSE-aware tool reads the same fact the card states.
Detection runs GitHub's `/license` endpoint first (it is the public corpus the
intent scopes to), then a native match of the root licence file against the
pinned SPDX text set from itd-164, then refusal with the file shown; README
prose is not consulted, and package manifests only as an opt-in layer, which
is how licensee reads them. This departs from the layered detector in the
intent's own open question (its layers 1–2 include package metadata); the
research supports manifests as opt-in, not as a default layer and not as
forbidden, and the planning interview should rule. The policy list is a list
of SPDX expressions, and `NOASSERTION` is never on it. The seam is the card's `files[]` entries: SWHIDs make a later
Software Heritage lookup an addition. No Go dependency.

### Local fit risks

- The intent is blocked by itd-36's provenance substrate; the fit-challenge
  should check that SWHID becomes the substrate's content key rather than a
  second hash beside it ([one-canonical-primitive](../../principles/one-canonical-primitive.md)).
- `vendor/` is a reserved name for the Go toolchain; a Go consumer looting a
  Go file into `vendor/<source>/` will confuse `go build` when
  `vendor/modules.txt` is absent or inconsistent (the reference says the `go`
  command errors on disagreement). The landing directory wants a ruling for Go
  projects.
- [adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md):
  `loot verify` fetches by design; it must stay an explicit verb and never run
  from a gate.

## itd-171: a decision points back to the conversation where it was reached

### Findings

- The W3C Web Annotation Data Model (Recommendation, 23 February 2017) is the
  standard for anchoring a passage in a text that may change: a
  `TextQuoteSelector` "MUST have exactly 1 exact property" and SHOULD have one
  `prefix` and one `suffix`; text "MUST be normalized before recording" and
  counted "in terms of unicode code points"; a `TextPositionSelector` is
  "very brittle with regards to changes to the resource"; several selectors
  may describe one segment and a consumer "MUST pick one". [CONSENSUS]
  [Web Annotation Data Model](https://www.w3.org/TR/annotation-model/).
- The operational recipe is Hypothesis's: store a range, a position and a quote
  selector (32 characters of prefix and suffix) and re-anchor in order: range,
  position, context-first fuzzy match near the expected offset, then
  whole-document fuzzy match, with an acceptance threshold on the difference.
  The post dates from 2013 and the approach has stood since. [CONSENSUS]
  [Fuzzy anchoring](https://web.hypothes.is/blog/fuzzy-anchoring/).
- Browsers carry the same idea as a URL fragment:
  `#:~:text=[prefix-,]start[,end][,-suffix]`, with the explicit design goal
  that a stale directive "degrade[s] gracefully into an ordinary link" and
  tells the reader the page changed. It is a Draft Community Group Report
  (13 December 2023), not a standard. [CONSENSUS]
  [Text fragments](https://wicg.github.io/scroll-to-text-fragment/).
- The host's own transcript already has a per-entry identifier. Community
  documentation of the JSONL describes a `uuid` per line, a `parentUuid`
  forming a tree, `sessionId` and an ISO timestamp; the official hooks page
  confirms `transcript_path` and `session_id`, warns the file "may lag the
  in-memory conversation", and documents a `prompt_id` UUID per user prompt,
  but does not itself document per-entry ids. [CONTESTED on stability: the
  per-entry `uuid` is community-described, not vendor-documented]
  [Claude Code hooks](https://code.claude.com/docs/en/hooks),
  [community JSONL description](https://claude-dev.tools/docs/jsonl-format).
- The commit-side pattern is settled: an id in a trailer, resolved by the tool
  that minted it. Gerrit's `Change-Id` is a footer line the commit-msg hook
  inserts "when none is defined yet" and survives amend, rebase and
  cherry-pick; Entire writes `Entire-Checkpoint: <id>` and keeps the session
  under `refs/entire/checkpoints/<shard>/<id>`, never on the active branch,
  redacting detected secrets best-effort; GitHub's cloud agent adds an
  `Agent-Logs-Url` trailer so "each commit message includes a link to the
  session logs". [CONSENSUS]
  [Gerrit Change-Id](https://gerrit-documentation.storage.googleapis.com/Documentation/3.12.0/user-changeid.html),
  [Entire checkpoints](https://docs.entire.com/cli/checkpoints.md),
  [Entire README](https://cdn.jsdelivr.net/gh/entireio/cli@main/README.md),
  [GitHub changelog, 2026-03-20](https://github.blog/changelog/2026-03-20-trace-any-copilot-coding-agent-commit-to-its-session-logs).
- The 2026-07-30 note already placed abcd on the "transcripts out of the
  repository, linked by trailer" side of that debate; this intent is the
  record-side link, not the commit-side one.
  [session-recording survey](2026-07-30-session-recording-sota.md).

### Options weighed

1. Entry `uuid` alone: machine-followable and cheap, but a host-specific id
   that redaction and a future format change can orphan, and the import route
   (other tools' transcripts) may have none.
2. Quote selector alone over the redacted text: survives any host and any
   re-redaction that leaves the quoted words intact; fails if redaction later
   rewrites inside the quote.
3. Both, Web Annotation style, with the position as a hint and the uuid as a
   context qualifier (Path 2).

### Recommendation

**Path 2**, shaped to the Web Annotation selectors. The anchor is a typed
cross-reference minted when the decision is recorded: `session_id`,
`source_sha256` of the stored record, the host entry id when the record has
one, and a `TextQuoteSelector` (`exact`, `prefix`, `suffix`, normalised,
computed from the redacted body, never the raw one) with a `TextPositionSelector`
as a hint. The reference is stored on the decision record (ADR, decision log
line, intent decision), never in the transcript, which the 2026-07-08 note
already settled for the ledger. `abcd history anchor <ref>` resolves it in
Hypothesis's order and reports which stage matched, so a stale anchor degrades
to "session found, passage moved" rather than to nothing. If a line-number or
byte-offset form is ever emitted it is the hint, not the anchor, which honours
the repository's existing refusal of positional references. No Go dependency.

### Local fit risks

- The history store's redaction rewrites spans in place; the selector must be
  minted after redaction, and a re-redaction that touches the quoted span must
  re-mint or mark the anchor suspect (compare the fingerprint rule in the
  [brief-to-record traceability note](2026-10-04-brief-to-record-traceability-sota.md)).
- A host entry id in a committed record is harmless, but a `prefix` or
  `suffix` lifted from a transcript is prose that left the local tier: it must
  pass the same banlist and identity scans as any committed text.
- The record is the floor, not the theory: a resolved anchor shows where the
  decision was reached, and the fit-challenge should ask whether a quoted span
  of a few hundred characters is enough for the facilitator's purpose or
  whether the whole turn is wanted.

## itd-91: a project declares once how it wants AI acknowledged

The repository's own `ATTRIBUTION` rules settle abcd's house policy; this
section researches only the configurable-preference gap and what the field
converged on since those rules were written.

### Findings

- The field has forked three ways, and each is current in 2026.
  Disclosure trailer: the Linux kernel's process document at 7.2 reads
  "Assisted-by: AGENT_NAME:MODEL_VERSION [TOOL1] [TOOL2]" with the example
  "Assisted-by: Claude:claude-3-opus coccinelle sparse"; the current document
  (7.3) reads "Assisted-by: LLM [TOOL1] [TOOL2]" with the example
  "Assisted-by: LLM coccinelle sparse", and in both "AI agents MUST NOT add
  Signed-off-by tags". Rocky Linux recommends an `Assisted-by:` trailer, MUST
  for content taken "directly from a tool without any changes", SHOULD
  otherwise; LLVM asks contributors to label substantial tool-generated
  content and gives `Assisted-by:` as the example trailer (page updated
  2026-10-22). [CONSENSUS on the trailer name; EVIDENCE that the kernel's
  value format changed between 7.2 and 7.3]
  [kernel coding assistants, 7.2](https://docs.kernel.org/7.2/process/coding-assistants.html),
  [kernel coding assistants, current](https://docs.kernel.org/process/coding-assistants.html),
  [Rocky Linux policy](https://docs.rockylinux.org/guides/contribute/ai-contribution-policy/),
  [LLVM AI tool policy](https://llvm.org/docs/AIToolPolicy.html).
  Co-authorship: Claude Code's default commit attribution is
  `Co-Authored-By: <model name> <noreply@anthropic.com>`; VS Code's
  `git.addAICoAuthor` shipped `off` in 1.110 and became on by default "for
  chat and agent workflows" in 1.118; GitHub's cloud coding agent records
  Copilot as author "with the human who gave Copilot the task marked as the
  co-author". GitHub's own `Co-authored-by` documentation counts a trailer
  toward a contributor's graph only when the mail is linked to an account.
  [CONSENSUS] [Claude Code settings reference](https://code.claude.com/docs/en/settings-reference),
  [VS Code 1.110](https://code.visualstudio.com/updates/v1_110),
  [VS Code 1.118](https://code.visualstudio.com/updates/v1_118),
  [GitHub changelog, 2026-03-20](https://github.blog/changelog/2026-03-20-trace-any-copilot-coding-agent-commit-to-its-session-logs),
  [GitHub co-authors](https://docs.github.com/en/pull-requests/committing-changes-to-your-project/creating-and-editing-commits/creating-a-commit-with-multiple-authors).
  Prohibition: Gentoo forbids contributing "any content that has been created
  with the assistance of Natural Language Processing artificial intelligence
  tools" (council vote 14 April 2024); QEMU will "DECLINE any contributions
  which are believed to include or derive from AI generated content" on DCO
  grounds, with exceptions by mailing-list approval; NetBSD presumes
  LLM-generated code "tainted" and requires core's written approval; git's
  SubmittingPatches defines no trailer and says it will "reject anything that
  looks AI generated". [CONSENSUS]
  [Gentoo AI policy](https://wiki.gentoo.org/wiki/Project:Council/AI_policy),
  [QEMU code provenance](https://www.qemu.org/docs/master/devel/code-provenance.html),
  [NetBSD commit guidelines](https://www.netbsd.org/developers/commit-guidelines.html),
  [git SubmittingPatches](https://raw.githubusercontent.com/git/git/master/Documentation/SubmittingPatches).
- Hosts already expose the preference as configuration, so the seam exists on
  their side. Claude Code's `attribution` key takes `commit` (a string, empty
  to hide), `pr` (a string, empty to hide) and `sessionUrl` (Boolean; the
  session link is a `Claude-Session` trailer on commits), or `false` for none
  (v2.1.281 or later); it is settable in "Any file", which includes the
  committed `.claude/settings.json`, and `includeCoAuthoredBy` is deprecated
  since v2.0.62. The settings page notes that user settings and local settings
  sit above the committed project file, so a committed preference is a default,
  not an enforcement. Two bounds from the same reference: `sessionUrl` covers
  only commits made from cloud and Remote Control sessions, and "Claude Code
  tells Claude that your own instructions about attribution, such as a
  CLAUDE.md or memory rule, take precedence over these commit and PR lines,
  unless the line is set in managed settings", so abcd's own instruction-file
  rule is itself one of the compile targets, not a bystander. [CONSENSUS]
  [Claude Code settings reference](https://code.claude.com/docs/en/settings-reference),
  [Claude Code settings](https://code.claude.com/docs/en/settings).
- git itself has the application machinery: `trailer.<alias>.key`,
  `ifExists` (`addIfDifferentNeighbor` by default; `replace`, `doNothing`),
  `ifMissing`, and `trailer.<alias>.cmd` to compute a value, with the manual's
  own example of a `commit-msg` hook that runs `git interpret-trailers` over
  the message file. A project preference compiles to trailer config plus a
  hook; the hook is the one place a human's own `git commit` is reached.
  [CONSENSUS] [git interpret-trailers](https://git-scm.com/docs/git-interpret-trailers).

### Options weighed

1. Record the preference in abcd config only and have abcd's own composers
   read it (what the draft describes): correct for artefacts abcd writes,
   silent for the host's and for a human's commits.
2. Record it once and compile it outward: to the host's attribution keys in the
   committed host settings file, to `trailer.*` git config and a `commit-msg`
   hook, and to the PR-body composer (Path 1 for the surfaces, Path 2 for the
   record).
3. Enforce only (a lint): out of scope by the draft's own ruling.

### Recommendation

**Path 1 for the surfaces, Path 2 for the record.** The preference is one
small structure in `.abcd/config.json` (`attribution: {stance, trailer_key,
value_form, pr_text}`) with five stances rather than the draft's four:
`none` (no disclosure line), `disclose` (a trailer, default key
`Assisted-by`, value form `vendor:model` as the house rule or `LLM tools` as
the kernel's current form), `coauthor` (with the authorship caveat), `custom`
(a template with `{vendor}`, `{model}`, `{date}` only), and `prohibit`, which
the field now uses as often as the others and which the draft's `none`
conflates with "do not disclose". The criterion's `Generated-by` option is a
`disclose` preset with `trailer_key: Generated-by`, so the five offered
options stay; the stance set is the stored form, not the menu. `abcd` compiles the choice outward in the
same act: the host settings file's `attribution.commit`, `attribution.pr` and
`attribution.sessionUrl: false`; `trailer.assisted.key`,
`trailer.assisted.ifExists replace` and a `commit-msg` hook that normalises
whatever trailer the host or the human wrote into the declared form; and the
PR-body composer. The seam is the compiler: a new host adds one target. No Go
dependency. Where the research contradicts the record: the house rule names
"kernel format" with a `Vendor:model` value; the kernel's current document has
moved to `LLM` plus tools and dropped the model, so "kernel-style" no longer
denotes one value form. The ruling stands; the selector should name the two
forms separately rather than call either "kernel-style".

### Local fit risks

- The `ATTRIBUTION` rules say a machine is never an author or committer and
  `Co-Authored-By` is never used for AI; a project that picks `coauthor` is
  choosing to fail abcd's own gate, so the selector must say which gates the
  choice disables, and `scripts/check-attribution.sh` must read the preference
  or stay abcd-only.
- Writing into the host's committed settings file is writing into a directory
  abcd does not own ([the-users-directory-is-theirs](../../principles/the-users-directory-is-theirs.md));
  the compile step needs consent and a marked block, the way the `AGENTS.md`
  conventions section is marked.
- Host-agnostic prose: the compiled targets name hosts, but user-facing docs
  must describe the preference, not the vendor keys, under the `harness/*`
  docs-lint rules.

## Not worth adopting

- W3C PROV as the ledger format (itd-127): the vocabulary is borrowed; the
  Entity, Activity, Agent document has no consumer at this scale, as the
  2026-07-08 note found.
- Typst as the primary renderer (itd-127): no CSL-JSON input, so the `custom`
  gates would be lost at the `.bib` export.
- A Zotero group library or any hosted bibliography service as the team seam
  (itd-126): it shares corpora and attachments, which is what must not travel.
- Reading licences from README prose or package manifests by default (itd-164,
  itd-26): the deployed detector declines to; the false-positive cost lands on
  the gate.
- A free-text licence field normalised on write (itd-164): it is how
  "MIT-spirit" entered the record.
- Vector retrieval for consult (itd-2609292108089653): rejected in 2026-07-08
  and nothing opened this pass changes it; FTS5 remains the named upgrade.
- `git subtree` as the loot mechanism (itd-26): it records a split commit and
  nothing the card needs beyond it.
- A per-loot licence override (itd-26): already ruled out on 2026-05-04; the
  OSADL data would make a later compatibility rung possible but the allow-list
  stands.
- Positional anchors into transcripts (itd-171): brittle by the standard's own
  note and refused by the repository's existing rule.
- A standing `git config` attribution value rather than a per-command one for
  reverts (itd-91): the `ATTRIBUTION` rules already forbid it; the compiled
  `trailer.*` config here adds a key and a hook, not a value.

## Not verified

- The Fedora AI-assisted contributions policy could not be opened (the site's
  bot protection refused every locale tried); its `Assisted-by:` recommendation
  is known only through search summaries and the Rocky Linux page that mirrors
  it. No link is given.
- Software Heritage's archive API (`resolve` and "save code now") was behind
  the same kind of protection; the SWHID standard itself was read from
  swhid.org. Whether a looted file can be looked up or archived on demand is
  unverified.
- The Robust Links specification (Memento) did not resolve at all; the Wayback
  availability API stood in for the archive-fallback finding.
- Pandoc's behaviour on a citation key missing from the bibliography (whether
  it warns, and so whether `--fail-if-warnings` catches it) is not stated in
  the manual or the citeproc README that were opened; the recommendation's
  post-render check does not rely on it.
- Entire's checkpoint id format and its full agent list were partly in the
  README and partly absent from the docs page opened; the trailer name and the
  refs location are confirmed.
- Whether the host's per-entry transcript `uuid` is stable across versions is
  described by community documentation only; the vendor's hooks page confirms
  the session id and the transcript path but not the per-entry id.
- The OSADL compatibility matrix page returned 403; the checklist project page
  was read instead, so the matrix's structure is described from it.
- go-trafilatura's benchmark figures are the maintainer's own; trafilatura's
  evaluation is the project's own. Neither has an independent replication that
  was opened.
- The web search budget for this session ran out before a search for pure-Go
  PDF extraction comparisons; the PDF finding rests on the three library pages
  opened directly.
