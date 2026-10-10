---
schema_version: 1
id: "iss-373"
slug: "three-adrs-end-with-leaked-agent-tool"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: ".abcd/development/decisions/adrs/0011-spec-terminology-rename.md"
resolution: "Deleted the two trailing tool-call scaffolding lines (</content>, </invoke>) from adr-11, adr-13, adr-20; tree-wide sweep now clean."
impact: internal
---

Three ADRs end with leaked agent tool-call closing tags committed into the durable record: a literal </content> line then </invoke>
## Evidence

- `.abcd/development/decisions/adrs/0011-spec-terminology-rename.md:85-86`
- `.abcd/development/decisions/adrs/0013-fn38-memory-single-writer-and-write-lint.md:78-79`
- `.abcd/development/decisions/adrs/0020-manifest-version-lockstep.md:188-189`

Each file ends with two literal byte lines `</content>` then `</invoke>` (confirmed via `cat -A`; not a code fence, not a render artefact). A tree-wide sweep for `</content>`/`</invoke>` across `.abcd/` returns exactly these three pairs. record-lint exits 0 — ungated.

## Adversarial verdict

CONFIRMED (substantive). Refuter read the file bytes and git history: leaked tool-call scaffolding, no legitimate reason, not prior art (distinct from iss-49 record cosmetics, iss-179 adr-6 citations). Fix: delete the two trailing lines from each file.
