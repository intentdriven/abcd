---
schema_version: 1
id: "iss-2610160608394392"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Mark the audit_mode block at 05-intent.md:569-575 as a design target of itd-50 rather than a shipped key."
---

The brief at .abcd/development/brief/04-surfaces/05-intent.md:571-572 says record-only is 'the default, and today's behaviour' and that an absent audit_mode key resolves to record-only, in a section not marked staged. No shipped code reads audit_mode or audit_budget (no match outside tests in internal/ or cmd/), and the backing intent itd-50 is in intents/planned/.
