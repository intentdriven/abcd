---
schema_version: 1
id: "iss-2610262033433919"
slug: "the-intent-chapter-says-the-quoted-text"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 432 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:318"
remedy: "Change the create row at 05-intent.md line 318 to say the draft seeds prompts for Why This Matters, Mechanism, Scope Conditions and Acceptance Criteria, as the matrix says."
finding_ref: cfk-2de1a33b0185f7a8
---

The intent chapter says the quoted-text create seeds Why This Matters and Acceptance Criteria; the verification matrix says it also seeds Mechanism and Scope Conditions.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:318` — "with Why This Matters and Acceptance Criteria seeded as prompts for the human to fill; assigns `itd-N` and derives the slug from the text"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:53` — "with Why This Matters, Mechanism, Scope Conditions and Acceptance Criteria seeded as prompts to be replaced"

05-intent's create row gives Why This Matters and Acceptance Criteria as the prompts seeded into the new draft. The matrix's Intent capture row says the same create seeds 'Why This Matters, Mechanism, Scope Conditions and Acceptance Criteria'. The seed either carries the two claim sections or it does not, so a test written from one page fails against the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 432.
