---
schema_version: 1
id: "iss-2608290808193471"
slug: "nothing-ties-a-merged-implementation-to-closing-its-spec-so"
severity: "major"
category: "process"
source: "impl-review"
found_during: "intent-implementation-run"
found_at: ".abcd/development/specs/open"
---

Nothing ties a merged implementation to closing its spec, so twelve specs stayed open and twelve intents stayed planned after the code that realises them landed on main: abcd spec close is the verb that advances the intent planned to shipped, closes the spec open to closed, and emits the fidelity-audit receipt, but no gate, lint rule or CI step notices that a spec is still open while its acceptance criteria are demonstrably delivered. The consequence is not cosmetic: the changelog composer reads the shipped-intent and resolved-issue folders, so a release cut derived while the intents sit in planned announces the resolved issues and omits every intent the release actually delivers. Found after the itd-150..162 implementation merged with all fourteen issues resolved but all twelve specs left open. Wants a detector for the shape 'spec open, linked intent planned, and the spec's own scope files changed since the spec was written', which is the same reference-graph the dangling-supersedes ratchet already walks.

Reframed 2026-08-29 under adr-55: this is not a decision waiting for anyone. Closing a specification once its implementation has landed is a step in the loop, and the loop stops only to obtain a verdict. It is performed, with the gate as the safety net. The earlier objection, that shipping an intent is a maintainer decision, rested on collapsing the product thinker and the facilitator into one word.
