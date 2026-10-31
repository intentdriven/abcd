---
schema_version: 1
id: "iss-2610310331125678"
slug: "the-capture-chapter-s-exit-codes-leave"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md"
remedy: "Add the shared-main-checkout refusal (exit 3, nothing written) to the exit codes in 04-surfaces/06-capture.md."
refines: [iss-2610222001116066]
---

The capture chapter's exit codes leave out exit 3, which every capture write returns in a main checkout two or more live sessions started in.
