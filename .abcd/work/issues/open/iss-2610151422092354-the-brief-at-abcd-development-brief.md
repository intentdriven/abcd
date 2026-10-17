---
schema_version: 1
id: "iss-2610151422092354"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-memory.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/37-memory.md:6-7 to match the shipped surface: Recall through the prompt hook is already shipped: internal/surface/cli/rules_prompt.go (recallMemory, header says spc-2610111627213138 step 2) calls internal/core/recall on every prompt, the 04-surfaces/README.md library/memory table says notes are brought…"
---

The brief at .abcd/development/brief/04-surfaces/37-memory.md:6-7 says: Recall of a note through the prompt hook is the same spec's later step; this chapter covers the store and the verb. The shipped surface differs: Recall through the prompt hook is already shipped: internal/surface/cli/rules_prompt.go (recallMemory, header says spc-2610111627213138 step 2) calls internal/core/recall on every prompt, the 04-surfaces/README.md library/memory table says notes are brought into a session automatically when a prompt matches their recall words, and commands/memory.md line 76 describes the prompt-match recall. It is not a later step. (v0.14.2 release crosscheck, false-claim, raised by sonnet in a29.)
