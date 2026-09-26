---
schema_version: 1
id: "iss-2609202015349672"
slug: "spec-close-help-does-not-say-that-the-close-creates-an-owed"
severity: "nitpick"
category: "documentation"
source: "agent-observation"
found_during: "Dessau pilot run, session gropiusllm-64, relayed to abcd-17 on 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
resolution: "abcd spec close --help names the OWED receipt, the abcd-review: OWED marker, the request path under .abcd/.work.local/reviews/ and the verb that answers it."
impact: fix
resolved_by:
  commit: "96811f71ace83f973adcfc309ee71ca77bcb679c"
---

spec close --help does not say that the close creates an owed fidelity-review receipt. At v0.9.0 the verb closes the spec, ships the intent when it was the last open spec, and also mints an OWED receipt (rcp-…), stamps an abcd-review: OWED marker into the intent body and writes the review request under .abcd/.work.local/reviews/; the help text names only the first two. A session in a managed repository (the Dessau pilot, gropiusllm-64, 2026-09-20) met the receipt as a surprise line in the output and called it useful and undocumented at the verb. The surface page for the intent lifecycle does describe the audit that follows the close; the help is where the verb is met. Wanted: one sentence in the close verb's Long help naming the receipt, the marker and the request path, so the caller knows the audit is now owed and where its input is.

## Grounds

- pursued: a caller learns at the verb that the close makes a fidelity review owed; a close help that omits the receipt, marker or request path would show it wrong
