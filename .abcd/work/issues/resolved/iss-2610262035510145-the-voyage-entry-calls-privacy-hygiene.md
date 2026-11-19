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
remedy: "In voyage.md replace 'privacy-hygiene audit rule' and 'fail its own audit' with 'lint rule' and 'fail its own lint'."
finding_ref: cfk-e788471a7800d013
resolution: "voyage.md now calls privacy-hygiene a lint rule and says an in-tree voyage would fail abcd's own lint, matching the lint chapter's vocabulary; a repolint test holds the wording."
impact: internal
resolved_by:
  commit: "d602e1bfe0d0f1990b4fb20b399a79dd76937f25"
---

The voyage entry calls privacy-hygiene an 'audit rule' run by abcd's 'own audit'; the lint chapter says that verb is a lint and that 'audit' is reserved

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/voyage.md:35` — "own `privacy-hygiene` audit rule flags those in committed files"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:19` — "The verb applies rules about form, which adr-40's vocabulary names a lint"

The lint chapter says the conformance verb applies rules about form, which adr-40's vocabulary names a lint, and that `/abcd:audit` is reserved for itd-16's hash-chain fidelity surface. privacy-hygiene is a rule of that lint. voyage.md calls it 'abcd's own `privacy-hygiene` audit rule' and says an in-tree voyage 'would have made abcd fail its own audit', which uses the reserved word for the lint.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 602.

## Grounds

- pursued: the voyage entry uses the lint chapter's word 'lint' for privacy-hygiene and no longer uses the reserved 'audit'; TestTheVoyageEntryCallsPrivacyHygieneALintRule failing, or 'audit rule'/'own audit' reappearing in voyage.md, would show it wrong.
