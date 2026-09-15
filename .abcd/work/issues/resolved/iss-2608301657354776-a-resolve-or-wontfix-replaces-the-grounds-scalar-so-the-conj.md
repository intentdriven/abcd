---
schema_version: 1
id: "iss-2608301657354776"
slug: "a-resolve-or-wontfix-replaces-the-grounds-scalar-so-the-conj"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-5-ruthless"
found_at: "internal/core/capture/workflow.go"
resolution: "The ledger's grounds are now an append-only `## Grounds` body section, mirroring the intent half: promote, resolve and wontfix each append a bullet, the reader parses the section, and the frontmatter scalar is retired with 16 committed records migrated. The record form (heading, reader, append, readability check) moved to core/grounds and the markdown-body machinery under it to a new core/mdrecord leaf, so both record families share one definition. The proposed refusal was not taken: all three routes require grounds, so refusing an overwrite would make a promoted issue impossible to resolve."
impact: fix
resolved_by:
  intent: "itd-179"
---

a resolve or wontfix replaces the grounds scalar so the conjecture a promote recorded is silently destroyed

Found by the round-5 ruthless review, which executed the sequence rather than
inferring it. Branch-introduced: the field is new here.

The ledger's `grounds` frontmatter scalar is single-valued, and
`setScalarField` REPLACES. So `Promote` records a conjecture, and a later
`Resolve` or `Wontfix` on the same issue overwrites it. The promote conjecture
is then gone from the record and from everywhere: promote writes grounds
nowhere else, and the minted draft carries `promoted_from` and a seed body but
no grounds entry.

Nothing warns, no count changes, the result reports success. And the loss is
unavoidable rather than accidental, because `capture resolve` now REQUIRES
`--grounds`. Fourteen records already in `resolved/` carry `promoted_to`, so
this is the ledger's mainline sequence and not a corner.

What makes it a defect rather than a tradeoff is that the sibling writer argues
the opposite at length for the same data. `intent/grounds.go` states that
recording is APPEND-ONLY, because the earlier conjecture is precisely what a
later reader checks the outcome against, and that rewriting it would leave the
record saying only what was believed last. The ledger half performs exactly the
rewrite the intent half declares unacceptable. spc-57's Ledger-side bullet is
silent, and `Issue.Grounds`'s doc says only that promote, resolve and wontfix
stamp it.

Remedy, either way but explicitly: refuse a transition that would overwrite a
non-empty `grounds`, matching the double-promote refusal already inside the same
locked section; or, if last-write-wins is intended, say so in `Issue.Grounds`
and in spc-57 so the loss is a chosen tradeoff. An append-only list here is a
schema decision and is not to be added on this branch without one.

## Grounds

- pursued: we expect one shared record form to be what makes the overwrite unrepresentable rather than merely refused, and a promote-then-resolve sequence that keeps both conjectures in order is what would show it wrong
