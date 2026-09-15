---
meta: {? origin: ABCD-EVAL-REFUSED-FLOW-EXPLICIT-KEY}
---

# Invariants

1. The core is transport agnostic.
2. A reading receives its input only through the assembler.

The excluded key above sits behind YAML's explicit-key indicator inside a flow
mapping. The flow scan reads a key that follows a brace or a comma directly,
and an explicit key is not that shape — so the key travelled. The floor refuses
the construction rather than guessing at the name it hides.
