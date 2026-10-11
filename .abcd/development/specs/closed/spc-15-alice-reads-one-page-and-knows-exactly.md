---
id: spc-15
slug: alice-reads-one-page-and-knows-exactly
intent: itd-100
---
# alice-reads-one-page-and-knows-exactly-where-abcd-stands-in

## Summary

spc-15 delivers itd-100: a public terminology crosswalk at
`docs/reference/terminology.md` mapping established agentic-AI terms to abcd's
position on each. Every row carries the term, a one-line established definition
with a footnote citation to a primary source, and abcd's position as exactly one
of **USES** (native vocabulary — the row names the verb or principle), **ADAPTS**
(different name, sharper meaning — the row says why), **REJECTS** (with the
recorded reason), or **WATCHING** (with the record id being watched). Every abcd
claim is grounded in the committed record; every established definition is
grounded in a fetch-verified primary source.

The term inventory and all positions were settled at the 2026-07-26 maintainer
grill (DECISIONS.md); the sweep evidence came from a six-cluster primary-source
research pass (protocols, core loop, context, safety, governance, operations)
plus a full record-mapping pass, all URLs fetch-verified 2026-07-26 and the two
bot-blocked OpenAI posts human-verified 2026-07-27.

## Scope

- **The page**: `docs/reference/terminology.md` — Diátaxis reference type, 26
  rows (8 USES / 10 ADAPTS / 7 WATCHING / 1 REJECTS), alphabetical order with a
  thematic mini-index (six clusters) at the top.
- **A row in `docs/reference/README.md`** naming the page.
- **A CHANGELOG entry** (user-facing addition).
- **Four ledger captures** for the WATCHING follow-ups the grill confirmed:
  SKILL.md-compatibility watch, AAIF/AGENTS.md stewardship citation watch,
  tamper-evident receipts, and payments-protocols-no-recorded-position (the
  STOP-condition outcome — the record is silent on payments, so no REJECTS was
  invented).
- **An ACKNOWLEDGEMENTS entry** for the practitioner post that seeded the
  crosswalk idea (inspiration credit only; the post is not a source and no
  citation on the page rests on it).

Out of scope:

- A structural docs-lint rule ("every table row carries at least one footnote")
  — assessed at implement time; deferred unless it slots into an existing rule
  shape without a bespoke engine. If deferred, that is recorded, not silent.
- Resolving iss-40 (glossary unification). The page is designed to be neutral to
  that outcome (see Design decisions).
- An MCP server, any protocol adoption, or any change to abcd behaviour — this
  is a documentation artefact.

## Design decisions (resolved in the 2026-07-26 maintainer grill)

1. **Mapping, not a registry.** The crosswalk holds no canonical definitions of
   native terms — external term, cited external definition, position, and an
   inline gloss of the native concept only. Canonical native definitions stay in
   the record's glossary (single-home direction under iss-40); cross-references
   are restricted to `docs/**` because `.abcd/development/` is excluded from the
   release artefact. Record ids (itd-N / iss-N / adr-N) appear as text, never as
   links.
2. **Citations are footnotes; vendor names live only there.** Bare markers in
   the table; full citation with DOI/URL in the footnote. Host and vendor names
   are confined to citation footnote lines, each carrying the sanctioned
   `<!-- docs-lint: allow -->` per-line escape — verified compatible with the
   `harness/*` banned-token family, whose matching and allow-escape are both
   line-scoped (`checkBannedTokens`, `internal/core/lint/lint.go`).
3. **Admission rule for terms**: a primary anchor only — an official
   specification, a standards body, a peer-reviewed paper with DOI, or
   multi-vendor primary engineering documentation (vendor engineering posts
   admissible only where they are the origin of the term). No single-author
   coinages, no aggregators. Consequences the grill blessed: "agentic
   workflow", "Policy Layer", "Agentic Pipeline", ANP, AGNTCY, and Mastercard's
   Verifiable Intent get no rows; x402 and the OpenAI/Stripe Agentic Commerce
   Protocol fold into the AP2 footnote (noting the "ACP" abbreviation collision
   with IBM's retired Agent Communication Protocol, merged into A2A in 2025);
   the policy-engine anchor (NIST SP 800-207) folds into the policy-as-code row.
4. **Positions never outrun the record.** A2A and AP2 are WATCHING, not REJECTS
   — itd-33 is a draft and the payments silence is a capture, because a REJECTS
   row requires a recorded reason and inventing one is the named failure mode.
   Honesty notes ship in-row where the record demands them (prompt-injection
   canaries: the lint code is reserved and execution is a design target, neither
   shipped; MCP: adapter shape recorded, no MCP server ships today).
5. **British English** prose; US English stays code-side. The persona and quote
   conventions of the intent corpus do not apply to the reference page (no
   personas on the page).

## Approach

Single hand-authored Markdown page. The mini-index groups the 26 alphabetical
rows into the six research clusters; each index entry is a same-page anchor.
Footnotes use standard Markdown footnote syntax and sit at the end of the page.
The page must pass `abcd docs` (docs-currency lint) with zero blockers: no
change-narration tokens, all relative links resolving, vendor tokens only on
allow-escaped footnote lines.

Delivery lands as small atomic commits on `docs/itd-100-terminology-crosswalk`:
record promotion (this spec + itd-100 to planned/), the page + README row +
CHANGELOG + ACKNOWLEDGEMENTS, and the ledger captures — reviewed per the run
contract (ruthless review + docs-currency review), PR opened and left for the
maintainer; never merged by the agent.

## Acceptance-criteria satisfaction

- **AC1 (row shape)** — the table's columns are term / established meaning
  (footnoted) / position, position vocabulary fixed to the four labels; review
  checks every row.
- **AC2 (primary sources)** — the footnote corpus is exactly the fetch-verified
  sweep output recorded in the grill package; no aggregator URLs.
- **AC3 (record-grounded positions)** — each position cites its record artefact
  by id or path gloss; WATCHING rows name an open record id (itd-16, itd-29,
  itd-33, iss-62, plus the captures minted in this delivery, iss-138–iss-141).
- **AC4 (lint-clean)** — `abcd docs` runs clean on the branch; vendor names
  appear only on footnote lines carrying the allow escape.
- **AC5 (docs/**-only cross-references)** — the page links only within
  `docs/**`; native terms are glossed inline, never deep-linked into the
  development record.
