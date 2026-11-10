---
id: dcn-2611101241364203
slug: agents-md-s-rule-loader-paragraph-and
date: 2026-11-10
---
2026-11-10 — AGENTS.md's rule-loader paragraph and the managed block ahoy plants say only that the loader injects the matched rules and then the memory it recalls; the recall's layers, the context brackets and rules --prompt are described in the brief (05-internals/07-memory.md section 9) and in docs, not in the block. AGENTS.md sat 10 bytes under the 32 KiB a host may read (TestAgentsMDFitsTheSmallestReadersCap), so step 7 of spc-2610111627213138 adds one byte there; moving more of the loader's detail behind the rules loader is a separate change.
