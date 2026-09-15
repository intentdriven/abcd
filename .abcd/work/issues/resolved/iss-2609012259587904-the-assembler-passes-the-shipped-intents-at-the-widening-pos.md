---
schema_version: 1
id: "iss-2609012259587904"
slug: "the-assembler-passes-the-shipped-intents-at-the-widening-pos"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "iteration-2 conformance audit against the design framework v4 and the readings companion v4, 2026-09-01"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/reading/include.go"
resolution: "The shipped intent projection row's positions become entailment, comparative and detection, and the exclusion floor gains a widening-scoped entry for .abcd/development/intents/shipped so the manifest asserts the withdrawal and assertExclusions enforces it. The widening definition's admitted-sources list loses the shipped line and its prompt_version moves PATCH. The framework's widening object and the readings companion's section 5.2 both state that object without the shipped intents, which is what the maintainer ruled on 2026-09-02."
impact: fix
resolved_by:
  intent: "itd-194"
  spec: "spc-2609021003136831"
---

The assembler passes the shipped intents at the widening position, which the readings companion's widening object does not list

The readings companion (v4, section 5.2) gives the widening reading's object as
"Brief current text including the construal section; `brief/glossary/`;
`intents/disciplines/` including the selection-criteria discipline; `specs/`;
the shipped tree where one exists", and excludes `intents/drafts/` and
`intents/planned/` under the rule that a reading never sees the material whose
state it exists to change. The governing framework's W12 table gives the same
object without the disciplines and specs. Neither lists `intents/shipped/`.

The include table in `internal/core/reading/include.go` carries the shipped
intent projection with `Positions: allPositions`, so the widening position
receives every shipped intent projected to its press release, acceptance
criteria, scope conditions, mechanism and spec id. The committed `cold` preset
happens not to select the intent-projection kind at widening, so the shipped
`cold` run does not carry them, but the admission is the table's and a preset
can only narrow it.

There is an argument either way. A shipped intent is a committed product of the
framing, admissible under adr-55, and the widening reading's question is what
the construal admits that is "not present in what has been committed to", for
which the committed intents are the clearest statement of what has been
committed to. Against that, both design documents chose not to list them, and
the second assembler rule (a reading's object excludes what it exists to change)
does not reach shipped intents, so the omission reads as deliberate rather than
as an oversight the rule would have caught.

The table should match a ruled object. Either the widening row set drops
`intents/shipped/` to match the companion, or the companion's section 5.2 is
amended to list it with the ground above, and the include table's row comment
cites the ruling. Until one happens the manifest's exclusions and the
definition's Object section describe two different readings.

## Grounds

- pursued: we expect taking the row's positions from the two design documents, and asserting the withdrawal in the floor, to make the widening position's object what both documents state, because a reader can then check the refusal rather than infer it from a row's silence; a shipped intent reaching a widening bundle, or the entailment and detection positions losing them, would show it wrong
