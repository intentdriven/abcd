# Writing style

The canonical style guide for prose committed to this repository — one
reference page holding every writing rule, which other surfaces point at
rather than restate.

Every rule carries an **enforcement** label, and that label is a fact about
shipped machinery, never an aspiration:

- **machine-enforced**: A shipped `abcd docs lint` rule checks it; the label
  names the rule id.
- **review**: Humans (and reviewing agents) check it. A rule stays labelled
  `review` until its lint ships, however firmly it is agreed.

The list-item em-dash rule is machine-enforced as a banned token. The colon
and semicolon casing rules are `review` by nature — no machine check holds
them without misfiring on legitimate prose, and
[adr-54](https://github.com/intentdriven/abcd/blob/main/.abcd/development/decisions/adrs/0054-punctuation-enforcement-stays-mechanical-only.md)
records the corpus evidence.

A machine check is promoted, not born, blocking: a new rule enters as a
`warn`, the corpus is fixed, and only a corpus-clean rule is promoted to
`blocker` — so a gate never lands as a wall of pre-existing findings.

## Language

| Rule | Enforcement |
|---|---|
| User-facing prose is British English. | review (the `spelling/*` docs-lint family flags common US spellings as advisory warns; the label stays `review` because warns gate nothing) |
| Code-side text is US English: identifiers, code comments, strings, flags, JSON keys. | review |

The split runs at the code boundary, not the file boundary; a British-English
reference page documenting a `--color` flag spells the flag as the code does. <!-- docs-lint: allow — the flag name is code-side text -->

## Tense

| Rule | Enforcement |
|---|---|
| Docs state present reality: what is, never what was or what is planned. History lives in git; rationale lives in the development record. | review |
| Change-narration tokens ("previously", "formerly", and their kin) are blocked in `docs/` and `README.md`. <!-- docs-lint: allow — this row documents the banned tokens themselves --> | machine-enforced (the `present_tense/*` docs-lint family) |

## Page types

| Rule | Enforcement |
|---|---|
| Documentation follows [Diátaxis](https://diataxis.fr/): one type per page (tutorial, how-to, reference, explanation), and each `docs/` folder holds one type. | review |

## Audience

| Rule | Enforcement |
|---|---|
| `docs/` is written for human readers, in one register; there are no parallel agent-facing versions of any page ([adr-53](https://github.com/intentdriven/abcd/blob/main/.abcd/development/decisions/adrs/0053-audience-by-placement.md)). The machine audience is served by machine surfaces: `--json` payloads, the generated CLI reference, and the injected rules. | review |
| A page's density is set by its Diátaxis type, never by an audience guess: A tutorial explains a term at first use; a reference states it and links the crosswalk. A page too dense for its reader is mis-typed or mis-placed, never re-registered. | review |
| Every section is understandable in isolation: no "as mentioned above", related information kept adjacent, exact error messages quoted verbatim. | review |
| A repo-specific term of art links its [`terminology.md`](terminology.md) entry at first use on a page. | review |

## Punctuation

| Rule | Enforcement |
|---|---|
| Em dashes are allowed in running prose. | n/a (permission, not a check) |
| Em dashes are not used inside list items: A pivot in a list item takes a colon instead. | machine-enforced (`punctuation/em-dash-in-list-item`); an em dash on a list item's wrapped continuation line is outside the line-based pattern's sight and stays `review` (adr-54) |
| After a colon: A capital letter. | review, permanently: a lowercase-by-design name (abcd, gofmt, macOS) legitimately follows a colon, and no machine check tells it from a violation (adr-54) |
| After a semicolon: lower case (the semicolon joins clauses of one sentence). | review, permanently: proper nouns and citation reference lists legitimately open with a capital after a semicolon (adr-54) |
| A list of three or more items takes a serial comma before the final conjunction: "brief, intents, and ADRs", not "brief, intents and ADRs". | review |

## Structure

Checks that already ship in `abcd docs lint`, listed here so the page states
the whole machine-enforced surface:

| Rule | Enforcement |
|---|---|
| Relative links resolve. | machine-enforced (`links_resolve`) |
| No stray root-level markdown beyond the allowlisted set. | machine-enforced (`stray_root_docs`) |
| Citations are well-formed, crosswalked, and within source policy. | machine-enforced (the `citation_*` rules) |
| User-facing prose stays host-agnostic; naming a specific tool is confined to attribution. | machine-enforced (the `harness/*` rules) |
| Reserved names the repository has banned do not appear (e.g. a record file naming itself). | machine-enforced (the verb-managed `names/*` rules) |
| No live agent-session URL and no tool attribution footer in user-facing prose. | machine-enforced (`harness_leak`) |

## Escapes

A line that legitimately trips a **banned-token** rule — the `present_tense/*`,
`punctuation/*`, `spelling/*`, `harness/*` and `names/*` families — carries
`<!-- docs-lint: allow -->` with a reason (a quoted title keeping its original
spelling, a code-side name, attribution); the escape is deliberate and
reviewable, never a default. The other
machine-enforced rules have no line escape: `links_resolve`, `stray_root_docs`
and the `citation_*` rules are satisfied by fixing the link, the file placement,
or the citation itself, not by annotating the line.
