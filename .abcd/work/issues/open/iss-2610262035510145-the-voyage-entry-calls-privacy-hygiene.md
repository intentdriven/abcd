---
schema_version: 1
id: "iss-2610262035510145"
slug: "the-voyage-entry-calls-privacy-hygiene"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 602 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/voyage.md:35"
remedy: "none (filed automatically)"
finding_ref: cfk-e788471a7800d013
---

The voyage entry calls privacy-hygiene an 'audit rule' run by abcd's 'own audit'; the lint chapter says that verb is a lint and that 'audit' is reserved

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/voyage.md:35` — "own `privacy-hygiene` audit rule flags those in committed files"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:19` — "The verb applies rules about form, which adr-40's vocabulary names a lint"

The lint chapter says the conformance verb applies rules about form, which adr-40's vocabulary names a lint, and that `/abcd:audit` is reserved for itd-16's hash-chain fidelity surface. privacy-hygiene is a rule of that lint. voyage.md calls it 'abcd's own `privacy-hygiene` audit rule' and says an in-tree voyage 'would have made abcd fail its own audit', which uses the reserved word for the lint.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 602.
