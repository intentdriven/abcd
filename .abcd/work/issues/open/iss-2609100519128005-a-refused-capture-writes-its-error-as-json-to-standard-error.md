---
schema_version: 1
id: "iss-2609100519128005"
slug: "a-refused-capture-writes-its-error-as-json-to-standard-error"
severity: "major"
category: "ux"
source: "agent-finding"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
---

A refused capture writes its error as JSON to standard error and leaves standard output empty, and an operator concluded from that shape that two captures had been silently lost. The conclusion was wrong and the shape that produced it is real, so both belong in the record. Tested on the published release and on the current source: an unknown category is refused with exit status one, the error is emitted as a JSON object on standard error, standard output is empty, and no record is written, verified against the ledger and a clean working tree afterwards. Nothing was lost. What the operator saw was a machine-readable invocation that produced no machine-readable output, in a pipeline that did not surface the exit status, for a flag whose accepted values are named nowhere in the help. Each of those alone is survivable and together they read as silent loss, which is why the operator re-ran the captures and reported data loss in good faith. The lesson is not that the refusal is wrong, because a refusal that writes nothing is exactly right. It is that a machine-readable mode should put its outcome where a machine-readable consumer looks, and that an error naming an invalid value while withholding the valid set turns one round trip into several and makes an operator doubt the store rather than the flag.
