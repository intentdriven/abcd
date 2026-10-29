---
schema_version: 1
id: "iss-2610262032399341"
slug: "adapters-says-a-user-docs-reader-copies"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 299 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/02-adapters.md:357"
remedy: "Rewrite the 'Lifeboat source readers' section of 02-adapters.md to match the code (Source interface with three tiers: git, conventions, native) and mark the user_docs reader, which no pack builds, as a design target."
finding_ref: cfk-35e6a8982d86d03b
---

adapters says a user_docs reader copies the user docs verbatim into the lifeboat; disembark's output shape has only ADRs under docs/

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/02-adapters.md:357` — "`docs/{tutorials,guides,reference,explanation}/` | Mirror verbatim"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:152` — "ADRs copied verbatim"

The adapters chapter lists a disembark source reader that copies docs/{tutorials,guides,reference,explanation}/ verbatim. The disembark chapter says the plan lists every file a pack writes, and its output tree carries only ADRs under docs/. A user-docs copy would be a path the output shape does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 299.
