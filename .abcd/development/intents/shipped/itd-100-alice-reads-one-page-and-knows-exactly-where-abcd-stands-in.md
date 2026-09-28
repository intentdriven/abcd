---
id: itd-100
slug: alice-reads-one-page-and-knows-exactly-where-abcd-stands-in
spec_id: spc-15
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# Alice Reads One Page and Knows Exactly Where abcd Stands in the Agentic-AI Landscape

## Press Release

> **Alice reads one page and knows exactly where abcd stands in the agentic-AI landscape.** Alice evaluates developer tooling for their team. Every tool describes itself in its own invented vocabulary, and they have to reverse-engineer how it relates to the terms the industry actually uses — MCP, the agent loop, context engineering, guardrails, policy-as-code, evaluations. abcd gives them a single reference page: a terminology crosswalk mapping each established agentic-AI term to how abcd incorporates it (naming the native verb or principle), adapts it under a sharper name, deliberately rejects it for a recorded reason, or is still watching it via a ledger entry they can follow. Every established definition cites a primary source — a spec, a standards body, or a peer-reviewed paper — and every claim about abcd is verified against the committed record, never memory. "I stopped reverse-engineering the invented names," said Alice. "One page told me, in the vocabulary I already use, what this tool does — and, rarer still, what it deliberately does not."

## Why This Matters

A tool that invents its own vocabulary asks every evaluator to do translation work it should have done itself, and the translation is where trust is won or lost: an unmapped term reads as either ignorance of the field or marketing evasion. The crosswalk makes the mapping a maintained, cited artefact instead of folklore — and it is deliberately honest in both directions, recording not only what abcd incorporates but what it rejects and why, with every rejection traceable to the committed record. A REJECTS row with a recorded reason is a stronger credibility signal than any feature claim.

## Acceptance Criteria

- Given Alice opens `docs/reference/terminology.md`, when they read any row, then it carries the term, a one-line established definition with a footnote citation, and abcd's position as exactly one of USES / ADAPTS / REJECTS / WATCHING.
- Given any footnote is followed, then it resolves to a primary source — a specification, a standards body, a peer-reviewed paper with DOI, or the engineering document where the term originated — never an aggregator or secondary write-up.
- Given any abcd-position claim in the table, then it is verifiable against a committed record artefact (a verb, principle, ADR, intent, or ledger id); WATCHING rows cite an open ledger entry by id.
- Given `abcd docs` (docs-currency lint) runs over the repo, then the page passes, with host and vendor names confined to citation footnote lines under the sanctioned allow escape.
- Given the page links to abcd-native concepts, then every cross-reference targets `docs/**` only; native terms carry an inline gloss rather than a deep link into the development record.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-8d75f3749de3 -->
Fidelity review — receipt rcp-8d75f3749de3 (verifier intent-fidelity-reviewer claude-fable-5).

Provenance: intent-fidelity-reviewer@claude-fable-5 · rubric_hash sha256:d1f6f6e7e9db1ceea44dd78279a564d2ced6b6904ffd7461dca652fbfac8042f · prompt_hash sha256:95792472ae74ca0469f69a51c618946e0d33cb1380032460099ed4b469d67e86
Input attestations: diff:3f0ff9b..2998402 (PR #148, merge 2998402)@sha256:fec2fe2c0c7203cf07d5039ebab4f61245fe341ea58332946b2b7cdd0a8cc217; rubric:.abcd/.work.local/reviews/rcp-8d75f3749de3.request.md@sha256:d1f6f6e7e9db1ceea44dd78279a564d2ced6b6904ffd7461dca652fbfac8042f;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: grep-verified: 26 table rows, all 26 carry a footnote marker, all 26 open the position column with exactly one of the four labels (8 USES / 10 ADAPTS / 7 WATCHING / 1 REJECTS, matching spc-15's declared distribution); zero rows carry two labels or none
  evidence: docs/reference/terminology.md:40 — "| Term | Established meaning | abcd's position |"
  evidence: docs/reference/terminology.md:42 — "| **Agent harness** | The infrastructure around the model …[^harness] | **USES** —"
  evidence: .abcd/development/specs/closed/spc-15-alice-reads-one-page-and-knows-exactly-where-abcd-stands-in.md:29 — "26 rows (8 USES / 10 ADAPTS / 7 WATCHING / 1 REJECTS)"
- ac-2 — MET: every footnote (lines 71-121) cites a specification, standards body, DOI-bearing paper, or origin engineering doc — no aggregators; the delivered diff includes a fetch-validation commit (52 URLs + 11 DOIs on this page, zero broken), and my independent spot-check of the least familiar URL (learn.chatgpt.com/docs/sandboxing) resolved to the genuine primary sandboxing doc
  evidence: docs/reference/terminology.md:119 — "IETF, RFC 9162, \"Certificate Transparency Version 2.0\", December 2021, DOI 10.17487/RFC9162"
  evidence: docs/reference/terminology.md:115 — "Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks\", NeurIPS 2020, DOI 10.48550/arXiv.2005.11401"
  evidence: commit f6d8b22 (docs: correct citation drift found by the validation sweep) — "fetched all 84 citation elements across the docs (52 URLs and 11 DOIs on the terminology page alone): zero broken links"
- ac-3 — MET_WITH_CONCERNS: all 21 cited record ids resolve to committed artefacts with matching lifecycle state (itd-3 shipped, itd-29 planned, itd-33/16/17/18/51 drafts, itd-5/81 disciplines, adr-23/24/25/27/29/35, iss-26/62/138–141 all in issues/open/) and content spot-checks match the claims; CONCERN: three of seven WATCHING rows (A2A, durable execution, observability) cite draft/planned intent or ADR ids but no open issues-ledger iss-N id — a divergence from the criterion's letter that spc-15 records as the signed-off widening to "an open record id"
  evidence: .abcd/development/decisions/adrs/0035-lifeboat-as-coverage-experiment.md:94 — "The graveyard's cite-or-be-dropped validator generalises"
  evidence: .abcd/development/intents/disciplines/itd-81-judge-calibration.md:135 — "known-good cases are ≥40% of the corpus"
  evidence: .abcd/development/brief/05-internals/08-skills.md:3 — "abcd ships zero skills"
  evidence: .abcd/work/issues/open/iss-138-payments-protocols-no-position.md:2 — "id: \"iss-138\""
  evidence: docs/reference/terminology.md:47 — "**WATCHING** — a draft record (itd-33) leans against it"
  evidence: .abcd/development/specs/closed/spc-15-alice-reads-one-page-and-knows-exactly-where-abcd-stands-in.md:110 — "WATCHING rows name an open record id (itd-16, itd-29, itd-33, iss-62"
- ac-4 — MET: `abcd docs lint` run 2026-07-27 at HEAD ab8876a exits 0 with 0 blockers (the single WARN is in docs/reference/cli/commands.md:308, a different file) and terminology.md is byte-identical to the merged state (git log 2998402..HEAD on the file is empty); a vendor-token grep over the page body above the Sources section finds none, and each footnote line carrying a token the line-scoped harness/* family bans (Codex, Gemini, claude-domain URLs) carries the sanctioned allow escape
  evidence: abcd docs lint (run output) — "abcd docs lint — 1 finding(s), 0 blocker(s)"
  evidence: .abcd/docs-lint.json:14 — "\"id\":\"harness/claude-code\"…\"allow_context\":[\"(?i)< !--\\\\s*docs-lint:\\\\s*allow\\\\b\"]"
  evidence: docs/reference/terminology.md:71 — "https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents. < !-- docs-lint: allow -- >"
- ac-5 — MET: grep for markdown links finds zero in-repo links anywhere in terminology.md — record ids appear only as plain text with an explicit inline-gloss statement, so no cross-reference leaves docs/**; the one inbound link lives in docs/reference/README.md within docs/**
  evidence: docs/reference/terminology.md:19 — "Identifiers such as `itd-N`, `iss-N`, `adr-N`, and `spc-N` refer to entries in abcd's development record"
  evidence: docs/reference/terminology.md:21 — "native terms are glossed inline rather than deep-linked"
  evidence: docs/reference/README.md:7 — "`terminology.md` (linked) — the terminology crosswalk"

Gap audit:
- honoured:
  - a single reference page mapping each established term to USES / ADAPTS / REJECTS / WATCHING
    evidence: docs/reference/terminology.md:40 — "| Term | Established meaning | abcd's position |"
  - every established definition cites a primary source, verified rather than assumed
    evidence: commit f6d8b22 (commit message) — "zero broken links, every DOI resolves to the cited work"
  - honest in both directions — a REJECTS with a recorded reason, and a payments silence captured as iss-138 instead of an invented rejection
    evidence: docs/reference/terminology.md:45 — "captured as iss-138 rather than dressed up as a rejection"
    evidence: .abcd/development/brief/05-internals/08-skills.md:3 — "abcd ships zero skills"
  - every claim about abcd is verified against the committed record, never memory
    evidence: .abcd/work/DECISIONS.md (delivered grill line, 2026-07-26) — "AP2 via new capture — record silent, REJECTS never invented"
  - the page ships wired into the user-facing surface with a CHANGELOG entry and an inspiration credit landed in the same change
    evidence: docs/reference/README.md:7 — "`terminology.md` (linked)"
    evidence: CHANGELOG.md (delivered hunk) — "A public terminology crosswalk at `docs/reference/terminology.md`"
    evidence: ACKNOWLEDGEMENTS.md (delivered hunk) — "Credited as the prompt, not a source"
- diverged:
  - press release: watching positions are followable "via a ledger entry" — delivered: three of seven WATCHING rows (A2A, durable execution, observability) track via draft/planned intent or ADR ids with no iss-N ledger entry, per the spec's recorded widening to "open record id"
    evidence: docs/reference/terminology.md:52 — "**WATCHING** — planned run resilience (itd-29)"
    evidence: .abcd/development/specs/closed/spc-15-alice-reads-one-page-and-knows-exactly-where-abcd-stands-in.md:110 — "WATCHING rows name an open record id"
  - press release: primary sources are "a spec, a standards body, or a peer-reviewed paper" — delivered admission rule additionally accepts origin vendor engineering docs (consistent with ac-2's fuller wording, and recorded as spec design decision 3)
    evidence: docs/reference/terminology.md:17 — "or engineering documentation from the vendors where the term originated"
    evidence: .abcd/development/specs/closed/spc-15-alice-reads-one-page-and-knows-exactly-where-abcd-stands-in.md:70 — "vendor engineering posts admissible only where they are the origin of the term"
- missing:
  - spc-15's conditional promise that deferring the structural docs-lint rule ("every table row carries at least one footnote") would be "recorded, not silent" — no such rule ships in the diff and no implement-time deferral record appears in the delivered DECISIONS.md line or ledger captures
    evidence: .abcd/development/specs/closed/spc-15-alice-reads-one-page-and-knows-exactly-where-abcd-stands-in.md:46 — "If deferred, that is recorded, not silent"
    evidence: .abcd/work/DECISIONS.md (delivered grill line, 2026-07-26) — "no mention of the structural lint-rule deferral"
