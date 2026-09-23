---
schema_version: 1
id: "iss-2608261543489261"
slug: "local-model-prompt-sanitiser-before-foundation-model"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "abcd-ux-phase-session"
found_at: "UserPromptSubmit hook / scanner seam"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: opt-in local-model prompt sanitiser at UserPromptSubmit has no planned intent). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Optional feature: a local model within the harness captures prompts and sanitises, cleans and PII-checks them before they are piped through to the harness's foundation model. A small on-machine model sits at the prompt seam (the UserPromptSubmit surface is the natural hook) and rewrites or flags secrets, private names and PII locally, so nothing sensitive leaves the machine unredacted — the redact-on-write posture the history store already takes for transcripts, applied upstream to live prompts. Composes with the existing scanner and banlist layers rather than replacing them, and with the privacy legibility thread of the UX work: the product thinker gets one plain claim, nothing leaves this machine unchecked. Opt-in adapter per the host-delegated default; a missing local model degrades loudly to pass-through, never silently.