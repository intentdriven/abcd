---
schema_version: 1
id: "iss-2611252051312482"
slug: "refusal-hides-a-valid-impact-value"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "the refusal of a differing impact in intent plan"
remedy: "Repeat a flag value in a refusal where it is a member of the flag's closed set; keep withholding any other value."
---

intent plan refuses an impact that differs from the recorded one and hides the value the person typed, which is one of the three it accepts: the message says (argument 5, not repeated here) where additive belongs, so it does not say what was compared. Tested 2026-11-25.
