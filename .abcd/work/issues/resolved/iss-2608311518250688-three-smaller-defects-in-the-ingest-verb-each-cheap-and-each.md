---
schema_version: 1
id: "iss-2608311518250688"
slug: "three-smaller-defects-in-the-ingest-verb-each-cheap-and-each"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "itd-185 fidelity audit rcp-fe3450ca55ff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest.go"
resolution: "All three. A definition that does not resolve now refuses the run through the recording path, so the refusal record exists with an empty regime and a reason saying why, rather than the run being refused with nothing durable to find it by. The bounded refusal list's elision entry names no item in either durable record: the ordinal is omitted from the JSON and the list-level reason renders it without one. U+2800 BRAILLE PATTERN BLANK is folded to a space, so a pattern that renders as nothing is refused at every regime; folding rather than correcting the residue was chosen because the residue's claim is about what renders, and a braille pattern carrying dots still lands."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

Three smaller defects in the ingest verb, each cheap and each an instance of a class this workstream has already recorded. A definition-drift refusal returns bare after instrument identity is proven, so the run is refused and no refusal record is written, contradicting both the criterion that says a list-level refusal writes one and the plugin surface's own corrected sentence. The refusal-list elision entry reaches the DURABLE record carrying ordinal zero, so a refused run writes item 0 into its refusal record although there is no item 0; the terminal renderer has the branch that suppresses it and the record writer does not. And the braille pattern blank is accepted as a provenance value: it renders as nothing but is neither a format rune nor a default-ignorable nor a variation selector nor whitespace, so it clears the blankness test. That last one does not breach the criterion's letter, which speaks of an empty or absent field, but it does breach the residue's claim that every invisible rune is folded away.

## Grounds

- pursued: a refused run that leaves no record cannot be found, a record naming an item 0 sends its reader after nothing, and a provenance that renders blank asserts a basis the reading does not carry; a drift refusal with no record, an ordinal on the elision entry, or a blank-rendering pattern accepted would each show this wrong
