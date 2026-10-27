---
schema_version: 1
id: "iss-2610262035071845"
slug: "itd-42-s-full-grill-loads-the-naming"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 549 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:58"
remedy: "none (filed automatically)"
related_intents: [itd-42]
finding_ref: cfk-9cd6b222b73b99a8
---

itd-42's full grill loads the naming, scope and out-of-scope chapters as well as the invariants and principles; the intent chapter's pre-pass assembles four inputs and nothing else

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:58` — "A full grill always loads `02-constraints/03-invariants.md`, `02-constraints/04-naming.md`, `01-product/04-scope.md`, `06-delivery/03-out-of-scope.md`, and the `principles/` set"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:320` — "Assembles the four inputs and nothing else, which is the budget: the draft, the brief's numbered invariants (`02-constraints/03-invariants.md`), each principle's rule paragraph, and a one-line index of every other intent (id, title, shelf) on every shelf."

itd-42 says a full grill always loads 02-constraints/03-invariants.md, 02-constraints/04-naming.md, 01-product/04-scope.md, 06-delivery/03-out-of-scope.md and the principles set. The chapter says the pre-pass assembles four inputs and nothing else: the draft, the invariants, each principle's rule paragraph and a one-line index. The naming, scope and out-of-scope chapters are in one context set and explicitly excluded from the other. The index fields differ too: id, slug, headline and state over three shelves in one, against id, title and shelf over every shelf in the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 549.
