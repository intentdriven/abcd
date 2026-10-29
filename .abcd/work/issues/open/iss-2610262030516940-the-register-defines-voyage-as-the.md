---
schema_version: 1
id: "iss-2610262030516940"
slug: "the-register-defines-voyage-as-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 53 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:231"
remedy: "In 09-reflect.md (lines 201-205) replace 'the voyage' and 'the new voyage's store and brief' with plain words (the repository, the new repository's store and brief); voyage is the operations namespace only (adr-35)."
finding_ref: cfk-20ecf1c8f71f2743
---

The register defines voyage as the operator-level record of operations; the reflect chapter uses voyage for a project and its store

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:231` — "namespace (verb-side: what we did), as against `lifeboat` (noun-side: what gets carried)"
- End B: `.abcd/development/brief/04-surfaces/09-reflect.md:201` — "A lifeboat pack carries every retrospective the voyage produced, verbatim"

04-naming defines voyage as the operations namespace, what was done, set against the lifeboat as what is carried, while 09-reflect speaks of the retrospectives the voyage produced and of the new voyage's store and brief, using voyage for a repository's life rather than an operation.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 53.
