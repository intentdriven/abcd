# Readings

The charter of the readings family: the durable record of every cold-reading
assembly this repository performs.

A run's artefacts live at `.abcd/development/readings/<run-id>/`, where
`<run-id>` is minted as `rdg-<yymmddHHMMSS><rrrr>` (adr-45). The manifest
enumerates the items an assembly passed, by path, by field and by hash. It
carries no item content, so committing it needs no redaction.

It carries no timestamp field, but it is not timestamp-free: the run identifier
embeds a mint stamp by construction. What holds across two assemblies of one
repository state at one commit is that the bundle is byte-identical and the
manifest's items and exclusions are identical, the two manifests differing in
the run identifier and in nothing else. That is the determinism a re-run is
checked against, and it is why the manifest sits outside the amnesia eval's
byte comparison rather than inside it.

The manifest's content is cold, which is why committing it for reader audit is
safe. The manifest as evidence is warm: it reveals run timing and target
selection. Both halves hold at once, and the second is why this family sits
inside the read block and no reading receives it.

## What a reading sees

The table below is generated from `internal/core/reading/include.go`, the
single source of truth, and a test holds the two to each other. Editing this
section by hand fails that test; edit the Go table and re-render.

Inclusion is positive at every grain: a position, a source directory, a file
match and, where a record travels as a projection, a named field list. What the
table does not name is absent, including a record family invented after the
table was written.

Two rules bind the table itself (ruled 2026-08-28):

1. No include names a directory that contains a record family. The deny is
   measured from a row's own source downward, so naming a family's leaf bucket
   individually is legal and reaching into a family from above is not.
2. A reading's object excludes the material whose state that reading exists to
   change. The drafts asymmetry and the Audit Notes exclusion are its two
   instances, which is what makes the table derivable rather than remembered.

A row's field list is a contract rather than a census, and the two differ by
position today. The intent projection names five fields. `Scope Conditions` and
`Mechanism` are headings no shipped intent carries, so the shipped rows yield
three; two drafts already carry them, so the entailment position — the only one
that reads drafts — projects all five. A field a file does not carry contributes
no item.

A projected field ends where the redactor ends a section: at the next heading of
the same level or shallower. A subsection therefore travels with the field it
sits under, rather than being cut off at the first heading of any depth.

Two headings are the same heading when they fold to one another or render to
one another, the second measured by the site's own anchor slug — the equality the
whole floor is stated against.

Three shapes stay outside it and are disclosed rather than claimed: a heading
nested inside a blockquote or a list item, which no scan here reads as a
heading; one reaching an excluded title through a homoglyph or an invisible
format character, which slugs differently because the comparison is over code
points; and a double-encoded character reference, which decodes to the literal
text it renders as and is therefore a different heading, left as one on purpose.

Two further residues are shared with the site renderer and pre-date this
instrument. A fence delimiter indented four spaces is accepted here as a fence,
where CommonMark reads an indented code block, so a heading between two such
lines is masked and not seen. And a double-quoted scalar continued across lines,
carrying a brace on a continuation line, refuses: the scan reads one line at a
time, and the closing quote is not on it.

The heading signal is scoped to markdown, and the include table is where that
scope is stated: every row declares, in its `Floor` column, whether the
exclusion floor parses what the row admits, and the floor runs over exactly the
rows declared `parsed`. A heading and a frontmatter key are things a record
carries, and a source, test or configuration file carries neither, so an item
from one of those rows travels whole and is marked `unscanned` in the manifest
— per item, so a reader can tell a scan that ran from a scan that never ran.
The manifest's key and heading exclusions are asserted for the items marked
`parsed` and for no other; its directory, file and unreachable-path exclusions
are asserted for every item, because those are enforced by path and a path needs
no parse. Within markdown a heading
is recognised however it is spelled: a closing sequence of hashes is normalised
away, a heading inside a fenced block is an example rather than a field and is
left alone, and an underlined heading is refused rather than redacted, because
the scan this projection spans by does not model one.

The bare verb reads run-directory NAMES under the assembler's own scratch
directory, and nothing else from `.abcd/.work.local/`. That is a listing of this
instrument's own staging, not a reader of the local ledger tier, and it is
stated here so it is not mistaken for one: no reading, and no part of this
assembler, opens a file in that tier.

A walk row's source directory must exist, and its absence refuses the run: a
brief chapter or the glossary going missing would otherwise enumerate nothing
and report clean. A record store's lifecycle BUCKET is different — an empty
bucket is a legitimate state of the record — so those rows stay silent, which is
disclosed residue rather than a check.

The exclusion floor is asserted into every manifest, each entry with the signal
by which a reader detects it, so the exclusions are checkable rather than taken
on trust. Its key and heading half is fail-closed like its path half: a file
that still carries an excluded key or heading after redaction refuses the run
rather than travelling with the manifest asserting otherwise. So does a markdown
document the floor cannot resolve at all — a fence delimiter inside the
frontmatter block, a delimited block displaced from line 0 by blank lines,
whitespace or an HTML comment, a compact mapping nested in a block sequence, an
explicit key in a flow mapping, an attribute value that opens on the line after
its equals sign, and a raw heading element that is never closed. Each is refused
by name, naming the document, the line and the shape, because a control that
cannot examine an input refuses it rather than admitting it silently (adr-56). Prose-borne warmth inside an admitted chapter has no structural
signal: the chapter-level bound and the glossary discipline carry it, and it is
disclosed as residue.

<!-- BEGIN GENERATED: reading-include-table -->

### Include table

| Positions | Source | Matches | Suffixes | Fields | Store | Bucket | Kind | Floor | Admitting rule |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| widening, entailment, detection | `.abcd/development/brief/01-product` | `.md` | none | the whole file | every | every | `brief-section` | `parsed` | adr-55: the construal as it presently stands is committed record, admissible to every reader including a cold reading |
| widening, entailment, detection | `.abcd/development/brief/02-constraints` | `.md` | none | the whole file | every | every | `brief-section` | `parsed` | The constraints chapter states the platform, the dependency stance, the invariants and the naming a reading reads against |
| widening, entailment, detection | `.abcd/development/brief/04-surfaces` | `.md` | none | the whole file | every | every | `brief-section` | `parsed` | The surfaces chapter is brief current text, which both design documents name as a reading's object |
| widening, entailment, detection | `.abcd/development/brief/05-internals` | `.md` | none | the whole file | every | every | `brief-section` | `parsed` | The internals chapter is brief current text, which both design documents name as a reading's object |
| widening, entailment, detection | `.abcd/development/brief/06-delivery` | `.md` | none | the whole file | every | every | `brief-section` | `parsed` | The delivery chapter is brief current text, which both design documents name as a reading's object |
| widening, entailment, detection | `.abcd/development/brief/glossary` | `.md` | none | the whole file | every | every | `glossary-term` | `parsed` | adr-55: the glossary's committed terms are committed record; superseded terms and the reasoning that settled them are not |
| widening, entailment, detection | `.abcd/development/brief` | `00-meta.md` | none | the whole file | every | every | `brief-section` | `parsed` | The meta chapter is brief current text, which both design documents name as a reading's object; it is one file at the brief's root |
| widening, entailment, comparative, detection | `.abcd/development/intents/disciplines` | `.md` | none | the whole file | `itd` | `disciplines` | `discipline` | `parsed` | A discipline is a standing commitment the record already holds, named individually inside the intent family |
| entailment, detection | `.abcd/development/intents/shipped` | `.md` | none | `Press Release`, `Acceptance Criteria`, `Scope Conditions`, `Mechanism`, `spec_id` | `itd` | `shipped` | `intent-projection` | `parsed` | Assembler rule 2: a shipped intent travels as its claim record, so the Audit Notes and dispositions it also carries stay behind; not at widening, whose object neither design document states with them in it (ruled 2026-09-02, iss-2609012259587904) |
| widening, entailment, detection | `.abcd/development/specs` | `.md` | none | the whole file | `spc` | every | `spec` | `parsed` | The design record a capability was built against |
| entailment | `.abcd/development/intents/drafts` | `.md` | none | `Press Release`, `Acceptance Criteria`, `Scope Conditions`, `Mechanism`, `spec_id` | `itd` | `drafts` | `intent-projection` | `parsed` | Assembler rule 2: articulation precedes selection, so entailment sees the candidate set and the reading asked to widen it does not |
| entailment | `.abcd/development/intents/planned` | `.md` | none | `Press Release`, `Acceptance Criteria`, `Scope Conditions`, `Mechanism`, `spec_id` | `itd` | `planned` | `intent-projection` | `parsed` | Assembler rule 2: articulation precedes selection, so entailment sees the candidate set and the reading asked to widen it does not |
| widening, entailment, detection | `.` | none | `_test.go` | the whole file | every | every | `test` | `unscanned` | Admitted where a committed preset entry names this kind, and never examined: an item admitted here travels whole and marked `unscanned` in the manifest, because the exclusion floor's key and heading signals are record shapes only a markdown file carries |
| widening, entailment, detection | `.` | `.go` | none | the whole file | every | every | `source` | `unscanned` | Admitted where a committed preset entry names this kind, and never examined: an item admitted here travels whole and marked `unscanned` in the manifest, because the exclusion floor's key and heading signals are record shapes only a markdown file carries |
| widening, entailment, detection | `.` | `.md` | none | the whole file | every | every | `doc` | `parsed` | Assembler rule 1: the shipped tree is the delivered documentation and the root prose, with the record denied structurally |
| widening, entailment, detection | `.` | `.json`, `.yml`, `.yaml`, `.toml`, `.mod`, `.sum`, `Makefile` | none | the whole file | every | every | `config` | `unscanned` | Admitted where a committed preset entry names this kind, and never examined: an item admitted here travels whole and marked `unscanned` in the manifest, because the exclusion floor's key and heading signals are record shapes only a markdown file carries |
| comparative | `.abcd/work/issues/readings` | `.md` | none | `configuration`, `what_admits_it` | `rdi` | every | `candidate` | `parsed` | adr-2609021016272867: at the comparative position, and at no other, two body fields of one DERIVED widening run's items are admitted as that reading's candidate set — the candidate text is cold, and its fate is warm. Everything else in the readings store stays excluded: the run's manifest, the items' envelopes and patterns, every disposition, admission and surprise, and every other run |

### Exclusion floor

| Detail | Signal | Rule | Positions |
| --- | --- | --- | --- |
| `origin` | frontmatter key | field projection | every position |
| `production_mode` | frontmatter key | field projection | every position |
| `Audit Notes` | heading | field projection | every position |
| `Open Questions` | heading | field projection | every position |
| `Why This Matters` | heading | field projection | every position |
| `Scope Condition Dispositions` | heading | a reading's object excludes what it exists to change | every position |
| `.abcd/development/brief/03-evidence` | directory | verdict material: a prior verdict is revision history, the ground the Audit Notes exclusion rests on | every position |
| `.abcd/development/decisions` | directory | absent from the positive walk | every position |
| `.abcd/development/roadmap/rfcs` | directory | absent from the positive walk | every position |
| `.abcd/development/intents/superseded` | directory | absent from the positive walk | every position |
| `.abcd/development/plans` | directory | absent from the positive walk | every position |
| `.abcd/development/research/notes` | directory | absent from the positive walk | every position |
| `.abcd/work/issues` | directory | no include names a directory containing a record family | widening, entailment, detection |
| `.abcd/work/DECISIONS.md` | file | absent from the positive walk | every position |
| `.abcd/.work.local` | directory | no reading consumes the local ledger side, unconditionally and under no flag (brief invariant 14) | every position |
| `the lapse log` | record type in a denied path | absent from the positive walk | every position |
| `admission and selection grounds` | record type in a denied path | absent from the positive walk | every position |
| `.abcd/development/readings` | directory | the instrument's own output is never its input | every position |
| `agents` | directory | the instrument's own output is never its input | every position |
| `evals` | directory | the instrument's own output is never its input | every position |
| `internal/core/reading` | directory | the instrument's own output is never its input | every position |
| `.abcd/.work.local/transcripts` | directory | the store is out of tree by default, and inside it only under .abcd/.work.local, which is excluded above | every position |
| `.abcd/development/intents/drafts` | directory | a reading's object excludes what it exists to change | widening, comparative, detection |
| `.abcd/development/intents/planned` | directory | a reading's object excludes what it exists to change | widening, comparative, detection |
| `.abcd/development/intents/shipped` | directory | the widening object as the design documents state it | widening |
| `.abcd/work/issues/open` | directory | the comparative reading receives candidates and never their fate: the ledger's other families are excluded family by family, derived from the ledger's own directory list | comparative |
| `.abcd/work/issues/resolved` | directory | the comparative reading receives candidates and never their fate: the ledger's other families are excluded family by family, derived from the ledger's own directory list | comparative |
| `.abcd/work/issues/wontfix` | directory | the comparative reading receives candidates and never their fate: the ledger's other families are excluded family by family, derived from the ledger's own directory list | comparative |
| `every run other than the candidate run, and every field of the named run's items other than configuration and what_admits_it` | readings store | the instrument's own output is never its input, but for the one derived widening run adr-2609021016272867 admits | comparative |
| `.abcd/work/issues/dispositions` | directory | the comparative reading receives candidates and never their fate: the ledger's other families are excluded family by family, derived from the ledger's own directory list | comparative |
| `.abcd/work/issues/admissions` | directory | the comparative reading receives candidates and never their fate: the ledger's other families are excluded family by family, derived from the ledger's own directory list | comparative |
| `.abcd/work/issues/surprises` | directory | the comparative reading receives candidates and never their fate: the ledger's other families are excluded family by family, derived from the ledger's own directory list | comparative |

<!-- END GENERATED: reading-include-table -->

## What the assembler does not do

The bundle is pathless by construction: each item is a key, a material class
and its text, and only the manifest maps a key back to a path and a field. The
remaining half of the isolation, that a dispatching host grants the reader no
repository access, is a host obligation stated in each definition, never an
enforcement this binary performs.

Timing and target selection stay the operator's choice. The manifest and the
run record make that visible after the fact rather than preventing it.
