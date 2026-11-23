---
schema_version: 1
id: "iss-2611221756090856"
slug: "the-grounds-package-s-comments-describe"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "mining the parked phase-9 clone, 2026-11-22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/grounds/grounds.go"
remedy: "Reword the doc comments on Parse and on Grounds.String in internal/core/grounds/grounds.go so neither names a frontmatter scalar as a carrier of the token grammar: that value is written as a Grounds bullet, passed as a --grounds operand and printed, and to no frontmatter. Leave the scalar half of the spaceRunRe comment as it is, since admission records still write a frontmatter grounds folded through it and reframe records carry the key too; widen only its bullet half, which names the intent record alone where issue records carry the bullet too."
---

Two doc comments in the grounds package name a carrier the token grammar no longer has. Parse's comment in internal/core/grounds/grounds.go calls the `<token>: <text>` grammar the one spelling written to a `grounds:` frontmatter scalar and to a bullet alike, and the comment on `Grounds.String` calls its value what a frontmatter scalar carries. No write path puts that value in frontmatter: it is written as a bullet under the record's Grounds heading, passed as a `--grounds` operand and printed. On an issue record the schema tolerates a frontmatter `grounds:` as a legacy key whose value nothing reads, and the record lint refuses it and names the section it belongs in. A reader of either comment learns of a writer the tree does not have.

A frontmatter `grounds:` is not retired everywhere, and the comment on spaceRunRe is right to name a YAML scalar in the ledger. Admission and reframe records each write a required `grounds:` key, free text and not the token grammar, and the admission writer folds that text through `Fold`. That half of the comment stays. Its other half names a Markdown bullet on the intent record, where issue records carry the bullet too.

A detection reading of 2026-09-02 found the stale grammar comment, and the answer given to that item accepted it and promised the comment a fix that no record then carried. An earlier wording of this record said no write path produces a frontmatter `grounds:` at all and asked for the spaceRunRe comment to be reworded; a review on 2026-11-22 showed both wrong, and the text and the remedy were narrowed to the above.
