---
schema_version: 1
id: "iss-2608261543489261"
slug: "local-model-prompt-sanitiser-before"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "abcd-ux-phase-session"
found_at: "UserPromptSubmit hook / scanner seam"
related_intents: [itd-2609292109005937]
deferred_after: "v0.11.1"
deferral_reason: "The product thinker's ruling J14 of 2026-09-29: plan the opt-in local-model prompt sanitiser. Filed as draft itd-2609292109005937. Owed: that draft's planning interview (the draft stays in drafts/ until a person plans it)."
---

Optional feature: a local model within the harness captures prompts and sanitises, cleans and PII-checks them before they are piped through to the harness's foundation model. A small on-machine model sits at the prompt seam (the UserPromptSubmit surface is the natural hook) and rewrites or flags secrets, private names and PII locally, so nothing sensitive leaves the machine unredacted — the redact-on-write posture the history store already takes for transcripts, applied upstream to live prompts. Composes with the existing scanner and banlist layers rather than replacing them, and with the privacy legibility thread of the UX work: the product thinker gets one plain claim, nothing leaves this machine unchecked. Opt-in adapter per the host-delegated default; a missing local model degrades loudly to pass-through, never silently.