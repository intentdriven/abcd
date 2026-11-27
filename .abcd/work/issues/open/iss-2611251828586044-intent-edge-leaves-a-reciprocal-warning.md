---
schema_version: 1
id: "iss-2611251828586044"
slug: "intent-edge-leaves-a-reciprocal-warning"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/README.md"
remedy: "Have the verb write the line on the named record too, or have the rule accept an edge the verb wrote. Grounds: a verb should leave a record its own gate reads as clean."
---

`abcd intent edge --builds-on` writes the edge on one record and record-lint then warns that the other record does not name it back, so the act the decomposition rules ask for leaves a warning each time: five such warnings on three intents on 2026-11-25. The named record is often shipped, where a hand edit is not expected.
