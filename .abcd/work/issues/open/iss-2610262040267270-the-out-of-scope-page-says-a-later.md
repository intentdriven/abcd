---
schema_version: 1
id: "iss-2610262040267270"
slug: "the-out-of-scope-page-says-a-later"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 931 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/03-out-of-scope.md:172"
remedy: "Rewrite the page's opening and its 'enters work by being scoped into a phase' sentence: sequencing is dependencies plus the lifecycle shelves (adr-2609212115255771); fold into the sweep iss-2609251618079479."
finding_ref: cfk-bd3d43f482289db0
refines: [iss-2609251618079479]
---

The out-of-scope page says a later intent enters work by being scoped into a phase; the build sequence says phases are retired and their documents are history.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/03-out-of-scope.md:172` — "A later-phase intent enters work by being scoped into a phase, then promoted to `planned/` via `/abcd:intent plan <itd-N>`."
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:7` — "the product phases that once bundled it are retired"

03-out-of-scope describes the live path into work as being scoped into a phase and then planned, and it frames the brief around eight planned phases. 01-build-sequence says the product phases that once bundled the intent set are retired (adr-2609212115255771), so no phase is left to scope an intent into.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 931.
