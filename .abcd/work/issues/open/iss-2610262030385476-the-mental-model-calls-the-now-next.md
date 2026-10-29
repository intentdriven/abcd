---
schema_version: 1
id: "iss-2610262030385476"
slug: "the-mental-model-calls-the-now-next"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 25 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:3"
remedy: "Edit 03-mental-model.md line 3: itd-2609212103568351 is shipped; the bare `abcd` board and the site Status page render the Now / Next / Later block."
finding_ref: cfk-e61cfb0d8aaf58c5
---

The mental model calls the Now / Next / Later status block a planned intent; the scope page says the bare `/abcd` board renders it

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:3` — "the status block that renders it is itd-2609212103568351, planned"
- End B: `.abcd/development/brief/01-product/04-scope.md:3` — "the bare `/abcd` board renders it as the Now / Next / Later block"

The mental model says the status block that renders Now / Next / Later is itd-2609212103568351, which is planned. The scope page says, in the present tense, that the bare `/abcd` board renders the Now / Next / Later block and places each intent in it. One page treats the render as not yet built and the other as current.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 25.
