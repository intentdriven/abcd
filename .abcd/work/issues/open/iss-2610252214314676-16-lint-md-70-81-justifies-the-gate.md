---
schema_version: 1
id: "iss-2610252214314676"
slug: "16-lint-md-70-81-justifies-the-gate"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S04"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md"
remedy: "Decide the bucket by adr-40's rule: either restate both targets as consuming the scanner's or the gates' findings to decide one action (a gate), or rebucket them as lint. Amend the Sub-verbs table and the rationale in the same change."
---

16-lint.md:70-81 justifies the gate bucket for the outbound and site targets by CI consumption ('wired into CI to make a binary pass/fail decision'; 'its exit is the release job's publish decision') and says 'the distinction is the one adr-40 draws'. adr-40:136-137 says 'A lint's exit code being consumed by CI does not make it a gate; producing findings is what names it', and :113-114 says a surface that produces findings is lint, review or audit.
