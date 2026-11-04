---
schema_version: 1
id: "iss-2611040554239939"
slug: "04-surfaces-02-disembark-md-306-says"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md"
remedy: "Say that a disembark ingest always reports no model, or add an optional model field to the four payload shapes and agent prompts."
---

04-surfaces/02-disembark.md:306 says model_reported is read from the payload's own model field. Every lifeboat payload decodes with unknown fields refused (:175), so adding model to a press-release, principles, review or lessons payload exits 2 with a malformed-JSON refusal, and no agent prompt's shape declares it. The receipt on a disembark ingest therefore always reads model reported not reported.
