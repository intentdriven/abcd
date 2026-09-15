# Terminology Glossary

<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

This directory is the abcd project's **one canonical glossary**. Each term is a Markdown file with
YAML frontmatter that defines the term's meaning, bounded context, and usage constraints. Cross-cutting
vocabulary — the nouns the record uses in prose, with their aliases and forbidden synonyms — is
registered here and nowhere else.

**The glossary is a deliberate frame surface.** The committed terms are the framing's
machine-visible fingerprint: what the project chose to name, what it refused to call things (the
forbidden synonyms), and where it drew each bounded context are the parts of its framing a machine
can read. Curating a term here is a framing act, not housekeeping — a rename or a new forbidden
synonym changes how every later reader, human or automated, construes the record.

The neighbouring registry in [`../02-constraints/04-naming.md`](../02-constraints/04-naming.md) is a
different artefact: it registers the maritime naming convention and the controlled enums (command
names, closed-vocabulary field values) that individual specs pin. It is not a glossary and holds no
term files.

---

## Format Specification

Every term file MUST begin with a YAML frontmatter block (`---` delimiters), optionally preceded by a
whole-line HTML comment (the attribution header). The body of the file (below the closing `---`)
provides narrative context, examples, and cross-references.

The field tables below are the source of truth for the frontmatter shape; abcd ships no separate
JSON-schema file for it.

### Required Frontmatter Fields

| Field | Type | Description |
|-------|------|-------------|
| `term` | string | Canonical lowercase name (no spaces) |
| `bounded_context` | string | Lowercase identifier matching the parent directory name. The current contexts are the subdirectories the [Term Index](#term-index) lists — a new context is a new subdirectory, nothing else to edit. |
| `definition` | string (≥10 chars) | Precise, unambiguous definition |
| `aliases` | array | Acceptable alternative names |
| `forbidden_synonyms` | array | Words that MUST NOT substitute for this term |
| `status` | enum | `draft`, `stable`, or `deprecated` |
| `introduced_in` | string | Version or intent ID when this term was coined |

### Optional Frontmatter Fields

| Field | Type | Description |
|-------|------|-------------|
| `starts_when` | string\|null | Condition/event that initiates this concept (lifecycle terms) |
| `ends_when` | string\|null | Condition/event that concludes this concept (lifecycle terms) |
| `not_to_be_confused_with` | string\|null | Related term that is commonly confused with this one |
| `versions` | array\|null | Version history records for the term definition |

### Constraints

- `forbidden_synonyms` and `aliases` MUST NOT overlap.
- When `status` is `stable` and either `starts_when` or `ends_when` is set, both must be
  present and non-null.
- `bounded_context` must match the name of the parent directory.

---

## Directory Layout

The glossary lives at `.abcd/development/brief/glossary/` (the home, per adr-30). The tree below is
rendered from the directory itself — see [Enforcement](#enforcement).

<!-- BEGIN GENERATED: glossary-layout -->
```
glossary/
├── README.md
├── _template.md
├── core/
│   ├── README.md
│   ├── brief.md
│   ├── construal.md
│   ├── disembark.md
│   ├── intent.md
│   ├── ledger.md
│   ├── lifeboat.md
│   ├── loop.md
│   ├── oracle.md
│   ├── persona.md
│   ├── phase.md
│   ├── plan.md
│   ├── reading-position.md
│   ├── record.md
│   ├── roadmap.md
│   ├── spec.md
│   ├── surface.md
│   ├── transport.md
│   └── voyage.md
├── distribution/
│   ├── README.md
│   ├── end-user.md
│   ├── release.md
│   └── version.md
├── interview/
│   ├── README.md
│   ├── embark.md
│   └── session.md
└── ledger/
    ├── README.md
    ├── admission.md
    ├── cold-reading.md
    ├── construal.md
    ├── disposition.md
    ├── lapse.md
    ├── position.md
    ├── read-block.md
    ├── regime.md
    └── warm.md
```
<!-- END GENERATED: glossary-layout -->

---

## Enforcement

Two mechanical checks read this directory. Neither is a full schema validator — the field tables
above are the shape's specification, and conformance to them is a review responsibility.

- **`GL002` forbidden synonyms** (`internal/core/lint`, run by the record-lint gate — `go run
  ./cmd/record-lint`). The rule walks this directory as the single source of truth for what a
  forbidden synonym is, then flags live prose that substitutes an *enforced* synonym for its
  canonical term. Enforcement is a configured subset of the declared synonyms, because most of them
  are common English words: a repo opts words in through the `forbidden_synonyms` rule in its
  `.abcd/record-lint.json`, and a word the glossary does not forbid is refused as a configuration
  error. This repo currently enforces one — `epic` (itd-43) — via that rule's `enforce` list.
- **The index drift gate** (`internal/core/glossary`, run by `go test ./internal/core/glossary/`).
  The Directory Layout and Term Index blocks above and below are rendered from the term files; the
  test fails the build when the committed README no longer matches what the directory holds. A term
  file that lands without an index row, or a bounded context that never reaches the index, is a build
  failure rather than a silent omission. Regenerate by copying the rendered block the failing test
  prints.

---

## Adding a New Term

### Manually

1. Copy `_template.md` to the appropriate subdirectory (see the [Term Index](#term-index) for the current contexts, or create a new context directory).
2. Fill in all required frontmatter fields.
3. Write a narrative body below the closing `---`.
4. Run `go run ./cmd/record-lint` and `go test ./internal/core/glossary/`.
5. Copy the rendered blocks the drift gate prints into the generated regions above and below — the
   index is derived from the term files, never hand-typed.

### Via `/abcd:intent grill` (glossary-aware mode)

When a project has this glossary directory, `/abcd:intent grill` enters
**glossary-aware mode** and can write new term files inline during the grill session.

**How grill writes back:**

- When the interview surfaces a new noun the user wants to pin, grill offers to write it
  immediately (never batched to the end of the session — per Pocock's `/grill-with-docs`
  pattern). The user confirms the term name, bounded context, and definition before any write.
- New terms are written to `glossary/<bounded_context>/<term>.md` using **atomic write**
  (`<file>.tmp` + POSIX `rename(2)`). A `kill -9` mid-session cannot corrupt existing term files.
- All grill-written terms receive `status: draft` and `introduced_in: <current-intent-id>`.
- Grill also detects **forbidden synonyms** in the intent body and proposes canonical replacements
  (using canonical display names in body prose, qualified IDs in `glossary_terms_used` only).
- When a term exists in multiple bounded contexts, grill asks the user to disambiguate and
  optionally sets `contexts: [...]` on the intent frontmatter.

**Body-prose vs machine-field distinction:**

Grill NEVER writes qualified IDs (`core/persona`) into intent body prose. Body prose uses
canonical display names (`persona`). Qualified IDs appear only in machine fields
(`glossary_terms_used`, ADR cross-refs, lint output). The optional inline form
`[persona](glossary:core/persona)` is the only way a qualified ID may appear in body prose.

**ADR offers:**

Grill offers to draft an ADR only when all three Pocock clauses pass:
hard-to-reverse + surprising-without-context + real-trade-off. If any clause fails, no offer
is made. ADRs are written to `.abcd/development/decisions/adrs/` with atomic write.

The complete write-back protocol is a **design target** of `/abcd:intent grill`'s glossary-aware mode; no shipped file documents it yet.

---

## Term Index

<!-- BEGIN GENERATED: glossary-index -->
### core/

| Term | Status | Definition |
|---|---|---|
| [brief](core/brief.md) | stable | The living root document that holds a project's purpose, constraints, and success criteria — always the project's current state, revised in place as the project moves. |
| [construal](core/construal.md) | stable | The statement of what the situation is being treated as, in one or two sentences, held at the top of the brief's framing chapter as the frame a widening reading reads against; one of adr-55's three framing surfaces, beside the committed glossary terms and the committed scope. The ledger context's entry governs the term inside the cold-reading experiment. |
| [disembark](core/disembark.md) | stable | The act of packing a lifeboat — `abcd disembark <source-repo> to <dest>` reads a source repository without writing to it and distils its settled artefacts, decisions, and configuration into a portable lifeboat directory at a destination outside that repository, which a fresh context can later unpack via `/abcd:embark`. |
| [intent](core/intent.md) | stable | A press-release-shaped description of a feature written before implementation begins, capturing the user problem, proposed solution, and success criteria. |
| [ledger](core/ledger.md) | stable | An append-or-move store a command writes and a human reads back, the issue ledger under .abcd/work/issues/ when the word stands bare; inside the cold-reading experiment the word means the warm material and its stores, which the read-block keeps from a reading (the ledger context's entries govern that sense). Four further ledgers share the word and are always named in full. |
| [lifeboat](core/lifeboat.md) | stable | A portable directory artefact packed by `/abcd:disembark` that captures the distilled knowledge and configuration of a source project so it can be unpacked into a fresh context by `/abcd:embark`. It always lands outside the source repository, at an operator-chosen destination. |
| [loop](core/loop.md) | draft | The record loop — brief to intent to spec to shipped work to audited verdict and back onto the brief — which shipping closes twice, once by grading the acceptance criteria and once by rewriting the brief passage. Two other loops carry the word and are always qualified: the autonomous-run loop and the lifeboat round-trip. |
| [oracle](core/oracle.md) | stable | An AI model invoked to review, reason over, or validate a project's artefacts — host-delegated by default, or reached through an opt-in oracle adapter. |
| [persona](core/persona.md) | stable | A placeholder stakeholder character drawn from the abcd personas registry, used in press releases, intents, and design documents to represent a real user archetype without using real names. |
| [phase](core/phase.md) | stable | An ordered stretch of development work that bundles a set of intents and brief plumbing-phases and ends in a milestone; abcd's sequencing layer, recorded as a document in roadmap/phases/. Unqualified it always carries that sense, the brief's own numbered build milestones being plumbing-phases. |
| [plan](core/plan.md) | stable | The maintainer's sign-off act `abcd intent plan <itd-N>`, which mints a spec, links both sides and moves a draft intent to planned/. Three further senses share the word — the ordered build plan the phase docs hold, a dated design plan under development/plans/, and a session's planning brief — and each is qualified where it appears. |
| [reading-position](core/reading-position.md) | stable | One of the four questions a cold reading can be commissioned to answer — widening, entailment, comparative or detection. The position fixes the reading's object, its question and the supply regime its output is validated against; `abcd reading assemble --position` names it. |
| [record](core/record.md) | stable | One identified, filed document that a command mints and a lint gate reads — an itd-N, spc-N, adr-N, iss-N or rdg-id. "The development record" is the whole durable corpus those records make up, and "a record family" is one lifecycle-bucketed set of them; each of the three is qualified where the other two could be read. |
| [roadmap](core/roadmap.md) | stable | The sequencing folder .abcd/development/roadmap/, which holds the phase docs and the RFCs. Its README is the roadmap dashboard, a separate sense — a live status render that reads the native spec store and the intent buckets rather than the phase docs. |
| [spec](core/spec.md) | stable | A specced block of work in abcd's native spec store that implements one or more intents, broken into ordered tasks with acceptance criteria. |
| [surface](core/surface.md) | stable | A verb's front door — the markdown command file under commands/ plus the transport package under internal/surface/ that reaches the core. "A surface chapter" is the brief's design record for one such front door, and "a rendered surface" is a public text held to the repository's identity block; both are qualified. |
| [transport](core/transport.md) | stable | The mechanism by which curated context and artefacts are packaged and delivered to an oracle for review or reasoning. |
| [voyage](core/voyage.md) | stable | The operations namespace at `~/.abcd/voyage/<source-root-sha>/` — an append-only record of what abcd *did* to produce a lifeboat (every disembark and embark run), as against the lifeboat itself, which is what gets carried. |

### distribution/

| Term | Status | Definition |
|---|---|---|
| [end-user](distribution/end-user.md) | stable | A person who installs and runs published abcd from the repo marketplace — the consumer of a release, distinct from a persona (a modelled archetype used in intents and briefs). |
| [release](distribution/release.md) | stable | A published, version-tagged snapshot of abcd — the act and the artefact of cutting a curated release from the single repo, carrying a version and a changelog entry. |
| [version](distribution/version.md) | stable | A strict-SemVer string stamped into the curated release artifact at cut time and carried as the git tag of the single repo, identifying a published snapshot of abcd for install and update. It is an OUTPUT of publishing, distinct from the internal sequencing unit (phase). |

### interview/

| Term | Status | Definition |
|---|---|---|
| [embark](interview/embark.md) | stable | The opening move of a grill session in which the oracle reads the target intent, identifies the primary ambiguities, and poses the first round of Socratic questions. |
| [session](interview/session.md) | stable | One complete interactive exchange between a human and the abcd grill sub-verb, spanning all rounds of Socratic questioning through PRD synthesis for a single intent or brief section. |

### ledger/

| Term | Status | Definition |
|---|---|---|
| [admission](ledger/admission.md) | draft | The researcher's act of taking a widening reading's proposed configuration into the candidate set, recorded with grounds, as distinct from declining it. |
| [cold-reading](ledger/cold-reading.md) | draft | A reading of committed artefacts by a disinterested party that has command of established patterns and no investment in the framing, receiving its input only through the assembler and producing items that each name the pattern they apply. |
| [construal](ledger/construal.md) | draft | The statement of what the situation is being treated as, in one or two sentences, held in the brief's framing chapter as the frame a widening reading reads against. |
| [disposition](ledger/disposition.md) | draft | The researcher's recorded response to one reading item, written as a separate record keyed to the item, in one of four states: accepted, rejected, declined or held. |
| [lapse](ledger/lapse.md) | draft | A recorded point at which the recording discipline was suspended, deferred or evaded, captured as its own category in the issue ledger and timestamped at the lapse rather than at write-up. |
| [position](ledger/position.md) | draft | One of the four places in the loop at which a cold reading is commissioned, each with its own object, question and supply regime: widening, entailment, comparative and detection. |
| [read-block](ledger/read-block.md) | draft | The wall that keeps ledger content from a cold reading: positive inclusion at the assembler, field projection out of files that hold both cold and warm material, a manifest that enumerates what was passed, and an eval that fails when warm material reaches a reading. |
| [regime](ledger/regime.md) | draft | The licence a reading holds at its position, naming what it may produce and what the ingest verb refuses or flags: generative, explicative, evaluative or registrative. |
| [warm](ledger/warm.md) | draft | The researcher's reserved reasoning, and the ledger material it is performed against: frame origination and admission, selection, explanation, the judgement of when to stop, and every record of how a prior tension was raised or settled. |
<!-- END GENERATED: glossary-index -->

---

## Exemptions

`GL002` reads every term file here for its forbidden-synonym set, and it never scans this directory
back: a term file names its own forbidden synonyms legitimately, so the glossary is always exempt from
its own rule. Beyond that, a repo scopes the rule in its `.abcd/record-lint.json` `forbidden_synonyms`
entry — `exempt_prefixes` skips whole path prefixes (the historical, git-tracked records), and
`allow_context` suppresses a line matching a named pattern (an external token that merely mentions the
word).

The case those escapes exist for is a **negative-test corpus**: the issue-schema validator's fixture
(spc-18) deliberately embeds the strings a validator must reject, so scanning it for those same
strings is a guaranteed false positive on every run. A corpus of that shape belongs behind
`exempt_prefixes`, named in the config with its reason, rather than left to fire and be ignored.

## Acknowledgements

Term file format adapted from [mattpocock/skills](https://github.com/mattpocock/skills) (MIT licence).
The `/abcd:intent grill` sub-verb's glossary capture pattern draws from the `grill-with-docs` skill
in that repository.
