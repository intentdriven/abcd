---
schema_version: 1
id: "iss-2610100626211810"
slug: "an-abcd-question-is-refused-with-the-mode-reads-managed"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the verb-split sign-off interview, 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard_question.go"
remedy: "Derive the mode from the chip when the question gate admits a question (Product to product-thinker, Tech to facilitator, Setup ruled in the fix) and set it there, so a chipped question is never refused on the mode; reset to managed when the answer returns, from a PostToolUse hook on the question tool, rather than on the next prompt; drop the set-before and set-back steps from the asking rules and the mode page, keeping abcd mode for a stop that is not a question."
---

An abcd question is refused with 'the mode reads managed' whenever the agent has not run abcd mode just before it, although the question's own chip ('Tech Q3', 'Product Q2') already names whom it is for. The mode is reset two ways: the asking rules tell the agent to set it back to managed after every answer, and the prompt hook resets it on the next prompt the session receives while a question is marked open, which in a turn of several questions can be a peer session's message or a message the person sends mid-turn (inferred from a refusal on 2026-10-10 after a peer message arrived; not yet reproduced). So an interview of several questions is refused intermittently, and the person sees the hook's error line each time (iss-2610070637562567 measured that every refusal reaches them framed as an error). The product thinker chose on 2026-10-10 to fix this in code first, before the general rewriter.
