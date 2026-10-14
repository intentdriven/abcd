---
schema_version: 1
id: "iss-2610112206188882"
slug: "history-staged-doc-says-checkout-only"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "security fix lane A, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/history.md"
remedy: "Update the sentence in commands/history.md and the generated surface so it names --root-sha as the way to read a repository's pile from anywhere"
---

The command surface's sentence for `abcd history staged` still says it refuses outside a git checkout, which stopped being true when --root-sha was added by the fix for iss-2609091717146700.
