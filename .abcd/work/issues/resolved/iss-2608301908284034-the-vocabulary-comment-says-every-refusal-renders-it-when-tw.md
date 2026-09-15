---
schema_version: 1
id: "iss-2608301908284034"
slug: "the-vocabulary-comment-says-every-refusal-renders-it-when-tw"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-179-fix-delta-ruthless"
found_at: "internal/core/grounds/grounds.go"
resolution: "The comment is scoped to the two token refusals that render the vocabulary, and degenerateWords is named at its own site as the copy that is a gate, with the drift record's enumeration corrected from five code sites to six."
impact: fix
resolved_by:
  intent: "itd-179"
---

the vocabulary comment says every refusal renders it when two of thirteen do and the drift record omits the one copy that is a gate

Found by the itd-179 fix-delta ruthless review. Branch-introduced by `9ad43b60`,
which is the commit that was CORRECTING an overstated claim on the same symbol.

The new text says "Every refusal this package raises renders it
(vocabularyList)". The package raises 13 `fmt.Errorf` refusals -- nine in
`grounds.go`, four in `record.go` -- and exactly TWO render the vocabulary list:
`Parse`'s grammar refusal and `ParseToken`'s. The other eleven render nothing of
the kind.

The second half is the one that matters. `iss-2608301836222858` enumerates the
literal-spelling copies as four sites in the CLI, one in `capture/grounds.go`,
plus two docs -- and omits `degenerateWords`, eighty lines below the corrected
comment in this same file, which spells the three tokens again and whose own doc
calls it "the vocabulary itself".

That omitted copy is the only one that is a GATE rather than a message. Add a
fourth token to `Vocabulary` and `ValidateText` silently stops refusing a
grounds text made solely of it, while the refusal it does not raise still says
the text "only repeats the vocabulary". So a record about documentation drift
was hiding a behavioural divergence.

Remedy: scope the comment to the token refusals, and add `degenerateWords` to
the enumeration in iss-2608301836222858 -- flagged as the gate copy, so whoever
consolidates does that one first.

## Grounds

- pursued: We expect naming the gate copy where it lives to outlast the record that lists it, because the next person to add a token reads the map and not the ledger.
