---
schema_version: 1
id: "iss-2608300829308000"
slug: "itd-183-fifth-round-nits"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-183 fifth-round ruthless review, 2026-08-30"
found_at: "internal/core/reading/assemble.go (outDirLabel), internal/core/reading/project.go (flowKeyRe, htmlTagRe), internal/core/reading/assemble_test.go"
resolution: "Three nits, two of them false-positive refusals a day of ordinary writing would have hit. The flow-mapping scan is anchored, so a quoted reason string that merely quotes a flow mapping no longer refuses — this corpus writes long quoted reasons, and the unanchored scan was one sentence from a repository that could not assemble. The tag pattern no longer strips an autolink, so a heading carrying a URL stays that heading. The default run directory now names itself in a non-empty refusal, since the operator supplied no spelling for a directory the assembler chose. And the amp-entity probe asserts the refusal it expects rather than skipping past it: with one-pass decoding it renders as Audit and Notes, which slugs onto the excluded title."
impact: internal
---

itd-183 fifth-round ruthless nits: outDirLabel is empty for the default run directory so a populated default directory would be refused with a blank name (reachable only on a run-id collision); flowKeyRe is unanchored and refuses an excluded key name inside a quoted scalar, and htmlTagRe strips an autolink so a heading followed by one refuses — both fail closed but the corpus's long quoted reason strings in flow mappings are one sentence away from the first; the 'an amp entity' case in the assembler test is assembled and then skipped before its assertion, a probe that tests nothing.
