# SOTA survey: confidential source consultation, provenance ledgers, and paper reconstruction

Grounding for [itd-76](../../intents/shipped/itd-76-source-provenance-ledger.md)
and the [convention-first scaffold plan](../../plans/2026-07-08-confidential-sources-scaffold.md).
Question: what is the best current (2024–2026) shape for a **local-only**
system in which an agent may *consult* confidential documents, never *cites*
them automatically, records source→decision influence in an append-only
ledger, and later reconstructs an academic paper (PDF + HTML) from that trail?

Five sub-questions, each with the ranked finding and the evidence tier
(official spec / widely adopted practice / practitioner anecdote).

## 1. Bibliography format — CSL-JSON wins

**Chosen: CSL-JSON as the single source of truth; `.bib` generated on demand.**

- CSL-JSON's reserved `custom` field is *specified* for exactly this job:
  key–value data outside the citation model, ignored by citeproc during
  rendering — so `confidential`, `permission_status`, `keywords`, `aliases`
  ride along without ever surfacing in formatted output ([CSL 1.0.2
  spec][csl-spec], [input schema][csl-schema]; official spec). Pandoc consumes
  CSL-JSON natively ([manual][pandoc]; official docs), and it is Zotero's
  native export.
- BibTeX/BibLaTeX tolerates unknown fields but has no schema; custom-field
  semantics are exporter-specific ([Better BibTeX extra fields][better-bibtex];
  adopted practice). Keep it as a generated artifact for LaTeX toolchains.
- Hayagriva (Typst's YAML format) defines a closed field list with no
  sanctioned custom-field mechanism ([format spec][hayagriva]; official docs)
  — no home for a `confidential` flag; Typst reads `.bib` anyway.
- A public bibliography is one filter: `custom.confidential != true`.

## 2. Provenance ledger — JSONL with PROV-shaped vocabulary

**Chosen: append-only JSONL, one record per influence; ADRs reference ledger
entries, never the reverse.**

- No de-facto standard exists; 2025–26 agent-provenance work converges on
  append-only JSONL event logs ([evidence-tracing survey][agent-evidence];
  practitioner consensus). RAG audit practice contributes the load-bearing
  linkage — claim ↔ source ↔ locator — that makes later paper reconstruction
  possible ([FINOS AI governance MI-13][finos-mi13]; industry guidance).
- W3C PROV is the only true standard ([PROV-JSON][prov-json]) but its
  Entity/Activity/Agent ceremony has no consumer at solo-developer scale —
  *borrow the vocabulary* (`influence`, derivation kinds) so a later export is
  mechanical; do not adopt the format.
- ADR practice is already append-only ("supersede and link, never edit" —
  [adr.github.io][adr-org]; adopted practice), matching this repo's
  `DECISIONS.md` convention: the ledger is the machine layer beneath it.

## 3. Local consultation — grep over a converted corpus; no RAG

**Chosen: extract every document to markdown/text once (pandoc / pdftotext);
agents search it with plain grep. SQLite FTS5 is the recorded upgrade path;
vector RAG is rejected at this scale.**

- The strongest-evidenced finding of the survey: agentic grep displaced
  embedding indexes in major coding agents in 2025, with direct benchmark
  support ([why grep beat embeddings on a SWE-bench agent][grep-vs-embeddings];
  [analysis of the no-indexing design][no-indexing]; adopted practice).
  A corpus of tens-to-hundreds of documents fits the paradigm trivially.
- Mitigation for prose (where vocabulary mismatch bites harder than in code):
  grep hand-written `keywords:` frontmatter, author names, and citation keys —
  not only content strings (practitioner nuance, [grep-vs-semantic
  caveat][grep-nuance]).
- If recall degrades: single-file SQLite FTS5 (BM25), optionally hybrid with
  local vectors ([sqlite-vec hybrid search][sqlite-vec]; practitioner
  anecdote). Full local RAG stacks add indexing pipelines and failure modes a
  grep answers in milliseconds — overkill (consensus across the surveyed
  write-ups).

## 4. Leakage guardrails — generate the denylist from the bibliography

**Chosen: structural separation (confidential strings exist only in the
user-level corpus and untracked banlists) plus mechanical scanning derived
from the marked entries.**

- The two-layer split this repo already runs (public banned tokens in CI lint;
  private patterns in an untracked pre-commit banlist, because a public rule
  cannot contain the secret it bans) is the right architecture —
  [itd-74](../../intents/shipped/itd-74-name-banlist.md) generalises it.
- The new move: *derive* the private patterns from the bibliography's
  `confidential: true` entries (title, aliases, full author names) so the
  banlist cannot drift from the corpus. Custom-rule secret scanners at
  pre-commit and CI are the established analogue ([gitleaks custom rules at
  pre-commit][gitleaks-precommit]; adopted practice). Prose linters with
  forbidden-term rules serve the same role in docs pipelines ([Vale][vale];
  adopted practice) — this repo's docs-lint banned-token family already covers
  that surface.
- Hooks defend against honest mistakes, not bypass; this repo already bans
  `--no-verify`. No off-the-shelf taint tracking for LLM output exists — the
  ledger's `cited_publicly` flag plus a release-time check (no rendered
  citation whose entry forbids it) approximates it.

## 5. Paper reconstruction — Quarto

**Chosen: Quarto, one markdown source, dual render (PDF via bundled Typst;
HTML), citeproc over the CSL-JSON.**

- Quarto is the only candidate hitting every requirement natively, and —
  decisive here — **project profiles + conditional content** give a
  *structural* citation firewall: the public profile renders from the filtered
  bibliography and simply cannot see confidential entries
  ([profiles][quarto-profiles], [conditional content][quarto-conditional],
  [Typst output][quarto-typst]; official docs). Install weight: one binary.
- Bare Pandoc + Makefile is the same engine minus the profile machinery —
  acceptable fallback, more glue.
- Typst alone: no real HTML story, closed bibliography schema (§1). Manubot's
  cite-by-public-identifier model is structurally wrong for sources that have
  no public identifier ([manubot][manubot]; official docs) — anti-recommended
  for this use case.

## Synthesis

CSL-JSON bibliography (+ `custom` block) → grep-consulted converted corpus →
append-only JSONL ledger keyed to decisions → banlist patterns generated from
the confidential entries into the itd-74 private guard layer → Quarto
public/internal profiles for eventual PDF + HTML. Nothing needs a service,
daemon, or index; the two bespoke pieces (ledger schema, release-time citation
check) are deliberately tiny because the field genuinely has no standard there.

## References

[csl-spec]: https://docs.citationstyles.org/en/stable/specification.html "Citation Style Language 1.0.2 specification"
[csl-schema]: https://github.com/citation-style-language/schema/blob/master/schemas/input/csl-data.json "CSL-JSON input schema (csl-data.json)"
[pandoc]: https://pandoc.org/MANUAL.html "Pandoc user's guide — citeproc and bibliography formats"
[better-bibtex]: https://retorque.re/zotero-better-bibtex/exporting/extra-fields/ "Better BibTeX for Zotero — extra fields and custom-field round-tripping"
[hayagriva]: https://github.com/typst/hayagriva "Hayagriva — Typst bibliography file format"
[prov-json]: https://www.w3.org/submissions/prov-json/ "PROV-JSON — W3C member submission"
[adr-org]: https://adr.github.io/ "Architectural Decision Records — homepage and conventions"
[agent-evidence]: https://arxiv.org/pdf/2606.04990 "Survey of evidence tracing and provenance in LLM agents (arXiv 2606.04990)"
[finos-mi13]: https://air-governance-framework.finos.org/mitigations/mi-13_providing-citations-and-source-traceability-for-ai-generated-information.html "FINOS AI governance framework — MI-13 citations and source traceability"
[grep-vs-embeddings]: https://jxnl.co/writing/2025/09/11/why-grep-beat-embeddings-in-our-swe-bench-agent-lessons-from-augment/ "Why grep beat embeddings in a SWE-bench agent (Liu / Augment)"
[no-indexing]: https://vadim.blog/claude-code-no-indexing/ "Analysis of a major coding agent's grep-over-index retrieval design"
[grep-nuance]: https://www.nuss-and-bolts.com/p/on-the-lost-nuance-of-grep-vs-semantic "On the lost nuance of grep vs semantic search"
[sqlite-vec]: https://alexgarcia.xyz/blog/2024/sqlite-vec-hybrid-search/index.html "Hybrid full-text + vector search in SQLite (Garcia)"
[gitleaks-precommit]: https://m3ssap0.github.io/2023/09/29/pre-commit-gitleaks.html "Custom gitleaks rules in a pre-commit hook"
[vale]: https://vale.sh/ "Vale — syntax-aware prose linter"
[quarto-profiles]: https://quarto.org/docs/projects/profiles.html "Quarto project profiles"
[quarto-conditional]: https://quarto.org/docs/authoring/conditional.html "Quarto conditional content"
[quarto-typst]: https://quarto.org/docs/output-formats/typst.html "Quarto Typst PDF output"
[manubot]: https://manubot.org/ "Manubot — git-based manuscripts with automated citation resolution"
