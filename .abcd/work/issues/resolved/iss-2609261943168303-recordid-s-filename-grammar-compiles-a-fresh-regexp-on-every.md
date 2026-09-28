---
schema_version: 1
id: "iss-2609261943168303"
slug: "recordid-s-filename-grammar-compiles-a-fresh-regexp-on-every"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: #728 macOS cancel"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/recordid/recordid.go"
resolution: "idRe compiles each family's pattern once and shares it through a sync.Map keyed on the family tag; deriving one id drops from 89 allocations to 3, pinned by TestFilenameGrammarIsCompiledOncePerFamily."
impact: internal
resolved_by:
  commit: "e25a0b293"
---

recordid's filename grammar compiles a fresh regexp on every call: idRe (behind FilenameNumRe) runs regexp.MustCompile each time, and both per-file readers call it once per record file, the resolver's fileID in internal/core/recordid/resolve.go and peers' fileID in internal/core/peers/read.go. Every resolver scan and every peers scan, which the bare board runs on each render, therefore recompiles the same pattern once per record in the checkout: 29% of the CPU of the bare-board tests in internal/surface/cli, and several times that under -race, where it inflates the macOS check leg that iss-2609261924541555 records running past the merge queue's cap. The grammar is a pure function of a fixed family prefix, so one compiled pattern per prefix answers identically.

## Grounds

- pursued: every resolver and peers scan reads the grammar once per record file without compiling it; FilenameNumRe returning a distinct pattern per call, or fileID allocating more than 8 times per id, would show it wrong.
