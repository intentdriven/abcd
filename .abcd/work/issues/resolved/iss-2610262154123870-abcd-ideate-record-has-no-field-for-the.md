---
schema_version: 1
id: "iss-2610262154123870"
slug: "abcd-ideate-record-has-no-field-for-the"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "ideate simpler-brief, recording a reframed verdict"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/ideate.go"
remedy: "Add a reframing field to the verdict document, required when verdict is reframed and refused otherwise, render it under its own heading in the record, and have the next line point at it."
resolution: "abcd ideate record now takes a reframing: a reframed verdict must carry one and any other verdict must not, it is redacted and cleaned like the idea, the record renders it under The reframing after leg 3, the result reports it, and the verb's next line points at that section. The command page and the ideate brief chapter say so."
impact: breaking
---

abcd ideate record has no field for the reframing a reframed verdict carries. commands/ideate.md says a reframed idea's graduation 'carries the reframing recorded in leg 3, not the original wording', and the verb's own next line says the same, but the verdict document (internal/core/ideate/ideate.go) holds only claims, hits, kill attempts with an outcome, the verdict and rejected alternatives: the reframing has to be smuggled in as a kill attempt marked survived, which a later reader or a graduating intent cannot find by its role.
